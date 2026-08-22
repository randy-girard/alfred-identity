package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alfred-identity/app/internal/sources"
	"github.com/alfred-identity/app/internal/sso"
	"github.com/alfred-identity/app/internal/updatecheck"
	"github.com/coder/websocket"
)

func TestCheckUpdateInvalidRepo(t *testing.T) {
	a, _ := testAppWithConfig(t)
	if err := a.cfg.Update(func(c *sources.Config) { c.GitHubRepo = "bad" }); err != nil {
		t.Fatal(err)
	}
	info, err := a.CheckUpdate()
	if err != nil || info.Error == "" || info.Current != Version {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestCheckUpdateWithMockGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://example.com/r"}`))
	}))
	defer srv.Close()
	restore := updatecheck.SetAPIBaseForTest(srv.URL)
	defer restore()

	a, _ := testAppWithConfig(t)
	_ = a.cfg.Update(func(c *sources.Config) { c.GitHubRepo = "acme/app" })
	info, err := a.CheckUpdate()
	if err != nil || !info.UpdateAvailable || info.Latest != "v9.9.9" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestSetEQDirectory(t *testing.T) {
	a, dir := testAppWithConfig(t)
	if err := a.SetEQDirectory(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected missing dir error")
	}
	eqDir := filepath.Join(dir, "EverQuest")
	if err := os.MkdirAll(eqDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := a.SetEQDirectory(eqDir); err != nil {
		t.Fatal(err)
	}
	if a.cfg.Get().EQDirectory != eqDir || a.watcher == nil {
		t.Fatalf("eq=%q watcher=%v", a.cfg.Get().EQDirectory, a.watcher)
	}
	st, err := a.GetEqHostState()
	if err != nil || st.Current != "" {
		t.Fatalf("state=%+v err=%v", st, err)
	}
}

func TestSetProxyEnabledDisabled(t *testing.T) {
	a, dir := testAppWithConfig(t)
	a.sso = sso.NewClient()
	_, cleanup := configureProxy(t, a, dir)
	defer cleanup()
	if err := a.SetConnectionMode(string(sources.ConnectionLoginOnly)); err != nil {
		t.Fatal(err)
	}
	if err := a.SetProxyEnabled(false); err != nil {
		t.Fatal(err)
	}
	if a.proxy != nil || a.cfg.Mode() != sources.ConnectionDisabled {
		t.Fatalf("proxy=%v mode=%s", a.proxy, a.cfg.Mode())
	}
}

func mockFullState(isAdmin bool) map[string]any {
	return map[string]any{
		"type":           "full_state",
		"user_id":        int64(42),
		"discord_id":     "discord-1",
		"display_name":   "Tester",
		"is_admin":       isAdmin,
		"state":          map[string]any{"accounts": []any{}, "online": []any{}},
		"admin":          map[string]any{"users": []any{}, "roles": []any{}},
		"directory":      []any{},
		"groups":         []any{},
		"share_activity": map[string]any{"logins": []any{}, "online": []any{}},
	}
}

func connectAppSSOAdmin(t *testing.T, a *App, onMessage func(typ string, data []byte) map[string]any) {
	t.Helper()
	wsURL, cleanup := startAppMockSSO(t, onMessage)
	t.Cleanup(cleanup)
	a.sso = sso.NewClient()
	if err := a.sso.Connect(a.ctx, wsURL, "tok", "gui/test"); err != nil {
		t.Fatal(err)
	}
	waitAppSSOAdmin(t, a.sso)
}

func waitAppSSOAdmin(t *testing.T, c *sso.Client) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.Connected() && c.IsAdmin() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout connected=%v admin=%v", c.Connected(), c.IsAdmin())
}

func startAppMockSSO(t *testing.T, onMessage func(typ string, data []byte) map[string]any) (wsURL string, cleanup func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var tip struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &tip) != nil {
				continue
			}
			if resp := onMessage(tip.Type, data); resp != nil {
				out, _ := json.Marshal(resp)
				_ = conn.Write(ctx, websocket.MessageText, out)
			}
		}
	}))
	return "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/ws", func() {
		cancel()
		srv.Close()
	}
}

func waitAppSSO(t *testing.T, c *sso.Client) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.Connected() && c.UserID() != 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout connected=%v user=%d", c.Connected(), c.UserID())
}

func TestApplyConnectionModeLoginSSOConnects(t *testing.T) {
	a, dir := testAppWithConfig(t)
	_, cleanupProxy := configureProxy(t, a, dir)
	defer cleanupProxy()

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
	if _, err := a.SaveSource(sources.Source{Name: "G", Host: u.Host, Token: "tok"}); err != nil {
		t.Fatal(err)
	}

	if err := a.SetConnectionMode(string(sources.ConnectionLoginSSO)); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.SetConnectionMode(string(sources.ConnectionDisabled)) }()
	waitAppSSO(t, a.sso)
	if a.proxy == nil {
		t.Fatal("expected proxy running")
	}
}

func TestShareLocalAccountRoundTrip(t *testing.T) {
	a, _ := testAppWithConfig(t)
	wsURL, cleanupWS := startAppMockSSO(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return mockFullState(false)
		case "share_account":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "share_result", "request_id": msg.RequestID, "ok": true, "account_id": int64(3),
			}
		default:
			return nil
		}
	})
	defer cleanupWS()

	a.sso = sso.NewClient()
	if err := a.sso.Connect(a.ctx, wsURL, "tok", "gui/test"); err != nil {
		t.Fatal(err)
	}
	waitAppSSO(t, a.sso)

	if err := a.SaveLocalAccount("tank", "secret", nil); err != nil {
		t.Fatal(err)
	}
	if err := a.ShareLocalAccount("tank", []int64{2}); err != nil {
		t.Fatal(err)
	}
}

func TestUnshareLocalAccountRoundTrip(t *testing.T) {
	a, _ := testAppWithConfig(t)
	connectAppSSO(t, a, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return mockFullState(false)
		case "unshare_account":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "share_result", "request_id": msg.RequestID, "ok": true,
			}
		default:
			return nil
		}
	})

	if err := a.UnshareLocalAccount("tank"); err != nil {
		t.Fatal(err)
	}
}

func connectAppSSO(t *testing.T, a *App, onMessage func(typ string, data []byte) map[string]any) {
	t.Helper()
	wsURL, cleanup := startAppMockSSO(t, onMessage)
	t.Cleanup(cleanup)
	a.sso = sso.NewClient()
	if err := a.sso.Connect(a.ctx, wsURL, "tok", "gui/test"); err != nil {
		t.Fatal(err)
	}
	waitAppSSO(t, a.sso)
}

func TestStartProxyEnablesLoginSSO(t *testing.T) {
	a, dir := testAppWithConfig(t)
	_, cleanupProxy := configureProxy(t, a, dir)
	defer cleanupProxy()

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
	if _, err := a.SaveSource(sources.Source{Name: "G", Host: u.Host, Token: "tok"}); err != nil {
		t.Fatal(err)
	}

	if err := a.StartProxy(); err != nil {
		t.Fatal(err)
	}
	defer a.StopProxy()
	waitAppSSO(t, a.sso)
	if a.proxy == nil || a.cfg.Mode() != sources.ConnectionLoginSSO {
		t.Fatalf("proxy=%v mode=%s", a.proxy, a.cfg.Mode())
	}
}
