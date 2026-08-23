package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alfred-identity/app/internal/sso"
	"github.com/coder/websocket"
)

func startMockSSOForApp(t *testing.T, onType func(typ string)) (wsURL string, cleanup func()) {
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
			onType(tip.Type)
			switch tip.Type {
			case "auth", "get_state":
				out, _ := json.Marshal(map[string]any{
					"type": "full_state",
					"accounts": []any{},
					"online":   []any{},
				})
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

func TestSyncSSOPresenceWithConnectedClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seen := make(chan string, 4)
	wsURL, cleanup := startMockSSOForApp(t, func(typ string) {
		if typ == "heartbeat" {
			select {
			case seen <- typ:
			default:
			}
		}
	})
	defer cleanup()

	c := sso.NewClient()
	if err := c.Connect(ctx, wsURL, "token", "gui/test"); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()
	deadline := time.Now().Add(2 * time.Second)
	for !c.Connected() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !c.Connected() {
		t.Fatal("not connected")
	}

	a, _ := testAppWithConfig(t)
	a.sso = c
	a.syncSSOPresence(ctx, []string{"Hero", "Alt"})

	got := 0
	deadline = time.Now().Add(2 * time.Second)
	for got < 2 && time.Now().Before(deadline) {
		select {
		case <-seen:
			got++
		case <-time.After(50 * time.Millisecond):
		}
	}
	if got < 2 {
		t.Fatalf("expected 2 heartbeats, got %d", got)
	}
}

func TestSyncSSOPresenceSkipsWhenDisconnected(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	a.syncSSOPresence(context.Background(), []string{"Hero"})
}
