package app

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/alfred-identity/app/internal/sources"
	"github.com/alfred-identity/app/internal/sso"
)

func TestGetStatusReflectsConfig(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	st := a.GetStatus()
	if st.Version != Version || st.ConnectionMode != string(sources.ConnectionDisabled) {
		t.Fatalf("%+v", st)
	}
	if st.SSOConnected || st.ProxyEnabled {
		t.Fatal("expected disconnected idle status")
	}
}

func TestGetStatusWithConnectedSSO(t *testing.T) {
	a, _ := testAppWithConfig(t)
	c := sso.NewClient()
	c.SetStateForTest(sso.TestClientState{
		Connected: true,
		Admin:     true,
		UserID:    7,
		Accounts: []sso.AccountMeta{
			{ID: 2, Username: "beta", Aliases: []string{"b"}},
			{ID: 1, Username: "alpha"},
		},
	})
	a.sso = c

	st := a.GetStatus()
	if !st.SSOConnected || !st.SSOIsAdmin || st.SSOUserID != 7 {
		t.Fatalf("%+v", st)
	}
	if len(st.SSOAccounts) != 2 || st.SSOAccounts[0].Username != "alpha" {
		t.Fatalf("accounts=%+v", st.SSOAccounts)
	}
}

func TestGetSourcesAndSaveSource(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()

	if got := a.GetSources(); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
	for _, src := range []sources.Source{
		{},
		{Name: "only-name"},
		{Name: "new", Host: "127.0.0.1:8181"},
	} {
		if _, err := a.SaveSource(src); err == nil {
			t.Fatalf("expected error for %+v", src)
		}
	}

	dto, err := a.SaveSource(sources.Source{Name: "Guild", Host: "127.0.0.1:8181", Token: "secret"})
	if err != nil || dto.Name != "Guild" || dto.Host != "127.0.0.1:8181" {
		t.Fatalf("dto=%+v err=%v", dto, err)
	}
	if len(a.GetSources()) != 1 || a.GetStatus().ActiveSource != dto.ID {
		t.Fatalf("sources=%+v active=%q", a.GetSources(), a.GetStatus().ActiveSource)
	}
}

func TestSetActiveSourceWithoutSSO(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	dto, err := a.SaveSource(sources.Source{Name: "G", Host: "127.0.0.1:8181", Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.SetActiveSource(dto.ID); err != nil {
		t.Fatal(err)
	}
	if a.GetStatus().ActiveSource != dto.ID {
		t.Fatalf("active=%q", a.GetStatus().ActiveSource)
	}
}

func TestDeleteSourceRequiresConfirm(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	dto, err := a.SaveSource(sources.Source{Name: "G", Host: "127.0.0.1:8181", Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	a.ctx = nil
	if err := a.DeleteSource(dto.ID); err != nil {
		t.Fatal(err)
	}
	if len(a.GetSources()) != 0 {
		t.Fatal("expected source removed")
	}
}

func TestSaveSourceUpdatesExisting(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	dto, err := a.SaveSource(sources.Source{Name: "Guild", Host: "127.0.0.1:8181", Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := a.SaveSource(sources.Source{ID: dto.ID, Name: "Renamed", Host: dto.Host})
	if err != nil || updated.Name != "Renamed" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
}

func TestSetActiveSourceLoginSSO(t *testing.T) {
	a, _ := testAppWithConfig(t)
	wsURL, cleanupWS := startAppMockSSO(t, func(typ string, data []byte) map[string]any {
		if typ == "auth" {
			return mockFullState(false)
		}
		return nil
	})
	defer cleanupWS()

	u, err := url.Parse(wsURL)
	if err != nil {
		t.Fatal(err)
	}
	a.sso = sso.NewClient()
	dto, err := a.SaveSource(sources.Source{Name: "G", Host: u.Host, Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	_ = a.cfg.Update(func(c *sources.Config) { c.ConnectionMode = sources.ConnectionLoginSSO })
	if err := a.SetActiveSource(dto.ID); err != nil {
		t.Fatal(err)
	}
	waitAppSSO(t, a.sso)
}

func TestSetListenPortAndStopProxy(t *testing.T) {
	a, dir := testAppWithConfig(t)
	if err := a.SetListenPort(0); err == nil {
		t.Fatal("expected invalid port")
	}
	if err := a.SetListenPort(70000); err == nil {
		t.Fatal("expected out of range")
	}
	port := reserveUDPPort(t)
	if err := a.SetListenPort(port); err != nil {
		t.Fatal(err)
	}
	if a.cfg.Get().ListenAddr != "127.0.0.1:"+strconv.Itoa(port) {
		t.Fatalf("listen=%q", a.cfg.Get().ListenAddr)
	}

	_, cleanup := configureProxy(t, a, dir)
	defer cleanup()
	if err := a.SetConnectionMode(string(sources.ConnectionLoginOnly)); err != nil {
		t.Fatal(err)
	}
	a.StopProxy()
	if a.proxy != nil || a.cfg.Mode() != sources.ConnectionDisabled {
		t.Fatalf("proxy=%v mode=%s", a.proxy, a.cfg.Mode())
	}
}
