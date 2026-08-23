package sso

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestLoginAuthRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg struct {
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
			Username  string `json:"username"`
		}
		if json.Unmarshal(data, &msg) != nil || msg.Type != "login_auth" {
			return
		}
		resp, _ := json.Marshal(map[string]any{
			"type":                   "login_auth_response",
			"request_id":             msg.RequestID,
			"real_user":              "realuser",
			"encrypted_credentials":  "YQ==",
			"account_id":             42,
		})
		_ = conn.Write(ctx, websocket.MessageText, resp)
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := NewClient()
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()
	go c.readLoop(ctx)

	res, err := c.LoginAuth(ctx, "req-1", "alias")
	if err != nil {
		t.Fatal(err)
	}
	if res.RealUser != "realuser" || res.CipherB64 != "YQ==" || res.AccountID != 42 || res.Error != "" {
		t.Fatalf("res=%+v", res)
	}
}

func TestLoginAuthWithRetryReturnsProviderError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg struct {
			RequestID string `json:"request_id"`
		}
		_ = json.Unmarshal(data, &msg)
		resp, _ := json.Marshal(map[string]any{
			"type":       "login_auth_response",
			"request_id": msg.RequestID,
			"error":      "not_found",
		})
		_ = conn.Write(ctx, websocket.MessageText, resp)
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := NewClient()
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()
	go c.readLoop(ctx)

	res, err := c.LoginAuthWithRetry(ctx, "req-1", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if res.Error != "not_found" {
		t.Fatalf("res=%+v", res)
	}
}

func TestLoginAuthNotConnected(t *testing.T) {
	c := NewClient()
	_, err := c.LoginAuth(context.Background(), "id", "user")
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("err=%v", err)
	}
}

func TestRequestRPCRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg struct {
			RequestID string `json:"request_id"`
		}
		_ = json.Unmarshal(data, &msg)
		resp, _ := json.Marshal(map[string]any{
			"type":       "admin_result",
			"request_id": msg.RequestID,
			"ok":         true,
			"account_id": 7,
		})
		_ = conn.Write(ctx, websocket.MessageText, resp)
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := NewClient()
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()
	go c.readLoop(ctx)

	res, err := c.requestRPC(ctx, map[string]any{
		"type": "admin_ping", "request_id": "adm-1",
	})
	if err != nil || !res.OK || res.AccountID != 7 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAdminRPCRequiresAdmin(t *testing.T) {
	c := NewClient()
	_, err := c.adminRPC(context.Background(), map[string]any{"type": "admin_ping"})
	if err == nil || !strings.Contains(err.Error(), "admin access required") {
		t.Fatalf("err=%v", err)
	}
}

func fullStateMessage(isAdmin bool) map[string]any {
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

func waitForSSOState(t *testing.T, c *Client, wantAdmin bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.UserID() != 0 && (!wantAdmin || c.IsAdmin()) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout connected=%v admin=%v user=%d", c.Connected(), c.IsAdmin(), c.UserID())
}

func startMockSSOServer(t *testing.T, onMessage func(typ string, data []byte) map[string]any) (wsURL string, cleanup func()) {
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
	wsURL = "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/ws"
	return wsURL, func() {
		cancel()
		srv.Close()
	}
}

func TestClientPingGetsPong(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pings := make(chan struct{}, 4)
	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth", "get_state":
			return fullStateMessage(false)
		case "ping":
			select {
			case pings <- struct{}{}:
			default:
			}
			return map[string]any{"type": "pong"}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, false)

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if err := c.writeJSON(ctx, conn, map[string]any{"type": "ping"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-pings:
	case <-time.After(2 * time.Second):
		t.Fatal("expected server to see ping")
	}
	if !c.Connected() {
		t.Fatal("disconnected after ping/pong")
	}
}

func TestKeepaliveLoopSendsPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pings := make(chan struct{}, 8)
	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth", "get_state":
			return fullStateMessage(false)
		case "ping":
			select {
			case pings <- struct{}{}:
			default:
			}
			return map[string]any{"type": "pong"}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	c.SetKeepaliveIntervalForTest(40 * time.Millisecond)
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, false)

	select {
	case <-pings:
	case <-time.After(2 * time.Second):
		t.Fatal("expected keepalive ping")
	}
	if !c.Connected() {
		t.Fatal("disconnected during keepalive")
	}
}

func TestConnectReceivesFullState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth", "get_state":
			return fullStateMessage(true)
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	if !c.Connected() || !c.IsAdmin() || c.UserID() != 42 {
		t.Fatalf("connected=%v admin=%v user=%d", c.Connected(), c.IsAdmin(), c.UserID())
	}
}

func TestRefreshStateWaitsForFullState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		if typ == "auth" || typ == "get_state" {
			return fullStateMessage(false)
		}
		return nil
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, false)

	if err := c.RefreshState(ctx); err != nil {
		t.Fatal(err)
	}
	if c.IsAdmin() {
		t.Fatal("expected non-admin state")
	}
}

func TestHeartbeatWhenConnected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	heartbeatSeen := make(chan struct{}, 1)
	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(false)
		case "heartbeat":
			select {
			case heartbeatSeen <- struct{}{}:
			default:
			}
		}
		return nil
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, false)

	if err := c.Heartbeat(ctx, "Hero", false); err != nil {
		t.Fatal(err)
	}
	select {
	case <-heartbeatSeen:
	case <-time.After(2 * time.Second):
		t.Fatal("expected heartbeat message")
	}
}

func TestAdminRemoveTagRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_remove_tag":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID, "ok": true,
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	res, err := c.AdminRemoveTag(ctx, "raid", 7)
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAdminAddCharacterRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_add_character":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID, "ok": true,
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	res, err := c.AdminAddCharacter(ctx, "Hero", 7)
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAdminAddAccountRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_add_account":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID,
				"ok": true, "account_id": int64(99),
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	res, err := c.AdminAddAccount(ctx, "newbox", "secret", "")
	if err != nil || !res.OK || res.AccountID != 99 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAdminAddAliasRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_add_alias":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID, "ok": true,
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	res, err := c.AdminAddAlias(ctx, "box", 7)
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAdminSetUserRolesRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_set_user_roles":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID, "ok": true,
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	res, err := c.AdminSetUserRoles(ctx, 3, []string{"role1"})
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAdminUpdateAccountRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_update_account":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID, "ok": true,
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, true)

	disabled := true
	res, err := c.AdminUpdateAccount(ctx, 7, nil, &disabled, nil)
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestUnshareAccountRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(false)
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
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, false)

	res, err := c.UnshareAccount(ctx, "box")
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestShareAccountRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(false)
		case "share_account":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "share_result", "request_id": msg.RequestID, "ok": true, "account_id": int64(5),
			}
		default:
			return nil
		}
	})
	defer cleanup()

	c := NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	waitForSSOState(t, c, false)

	res, err := c.ShareAccount(ctx, "box", "pw", nil, nil, nil, nil)
	if err != nil || !res.OK || res.AccountID != 5 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}
