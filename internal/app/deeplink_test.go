package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/sources"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func TestApplySourceDeepLinkAddsAndUpdates(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = nil
	a.ctx = nil

	link, err := sources.EncodeSourceDeepLink(sources.Source{
		Name:  "GoodGuys",
		Host:  "identity.example.com",
		Token: "first-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	a.applySourceDeepLink(link)
	got := a.GetSources()
	if len(got) != 1 || got[0].Name != "GoodGuys" || got[0].Host != "identity.example.com" {
		t.Fatalf("sources=%+v", got)
	}
	id := got[0].ID

	link2, err := sources.EncodeSourceDeepLink(sources.Source{
		Name:  "GoodGuys",
		Host:  "identity.example.com",
		Token: "rotated-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	a.lastDeepLink = ""
	a.applySourceDeepLink(link2)
	got = a.GetSources()
	if len(got) != 1 || got[0].ID != id {
		t.Fatalf("expected update in place, got %+v", got)
	}
	cfg := a.cfg.Get()
	if len(cfg.Sources) != 1 || cfg.Sources[0].Token != "rotated-token" {
		t.Fatalf("token not updated: %+v", cfg.Sources)
	}

	a.lastDeepLink = ""
	a.applySourceDeepLink(link2)
	if len(a.GetSources()) != 1 {
		t.Fatal("identical re-import must not add a second source")
	}

	other, err := sources.EncodeSourceDeepLink(sources.Source{
		Name:  "OtherGuild",
		Host:  "other.example.com",
		Token: "other-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	a.lastDeepLink = ""
	a.applySourceDeepLink(other)
	got = a.GetSources()
	if len(got) != 2 {
		t.Fatalf("different host should add, got %+v", got)
	}
	hosts := map[string]bool{}
	for _, s := range got {
		hosts[s.Host] = true
	}
	if !hosts["identity.example.com"] || !hosts["other.example.com"] {
		t.Fatalf("hosts=%v", hosts)
	}
}

func TestSaveSourceSameHostReusesID(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = nil
	first, err := a.SaveSource(sources.Source{Name: "GoodGuys", Host: "identity.example.com", Token: "a"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.SaveSource(sources.Source{Name: "GoodGuys", Host: "IDENTITY.EXAMPLE.COM", Token: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same id, first=%s second=%s", first.ID, second.ID)
	}
	if len(a.GetSources()) != 1 {
		t.Fatalf("sources=%+v", a.GetSources())
	}
	if a.cfg.Get().Sources[0].Token != "b" {
		t.Fatalf("token=%q", a.cfg.Get().Sources[0].Token)
	}
}

func TestApplySourceDeepLinkInvalid(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.ctx = nil
	a.applySourceDeepLink("https://example.com/open-alfred?d=not-base64")
	if len(a.GetSources()) != 0 {
		t.Fatal("expected no source")
	}
}

func TestHandleOpenURLQueuesBeforeConfig(t *testing.T) {
	a := &App{}
	a.HandleOpenURL("alfred-identity://import?d=abc")
	if a.pendingDeepLink == "" {
		t.Fatal("expected queued link")
	}
}

func TestHandleSecondInstanceDeepLinkSkipsAlreadyRunning(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.ctx = nil
	link, err := sources.EncodeSourceDeepLink(sources.Source{Name: "G", Host: "h.example.com", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	a.handleSecondInstance(options.SecondInstanceData{Args: []string{link}})
	if len(a.GetSources()) != 1 {
		t.Fatalf("sources=%+v", a.GetSources())
	}
}

func TestWriteTakePendingDeepLink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, pendingDeepLinkFile)
	if err := os.WriteFile(path, []byte("alfred-identity://import?d=abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "alfred-identity://import?d=abc" {
		t.Fatalf("%q %v", b, err)
	}
	_ = os.Remove(path)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected removed")
	}
}
