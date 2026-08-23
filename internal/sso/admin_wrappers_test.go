package sso

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAdminMissingWrappersRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	seen := map[string]bool{}
	wsURL, cleanup := startMockSSOServer(t, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return fullStateMessage(true)
		case "admin_add_tag", "admin_remove_alias", "admin_remove_character",
			"admin_remove_account", "admin_set_user_access":
			seen[typ] = true
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

	if _, err := c.AdminAddTag(ctx, "raid", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdminRemoveAlias(ctx, "alt", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdminRemoveCharacter(ctx, "Hero", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdminRemoveAccount(ctx, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdminSetUserAccess(ctx, 9, true); err != nil {
		t.Fatal(err)
	}
	for _, typ := range []string{
		"admin_add_tag", "admin_remove_alias", "admin_remove_character",
		"admin_remove_account", "admin_set_user_access",
	} {
		if !seen[typ] {
			t.Fatalf("missing RPC %s", typ)
		}
	}
}
