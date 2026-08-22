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
