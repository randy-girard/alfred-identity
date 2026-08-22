package main

import (
	"encoding/json"
	"testing"

	"github.com/alfred-identity/app/internal/sso"
)

func TestSSOAdminAddAccountSuccess(t *testing.T) {
	a, _ := testAppWithConfig(t)
	connectAppSSOAdmin(t, a, func(typ string, data []byte) map[string]any {
		switch typ {
		case "auth":
			return mockFullState(true)
		case "admin_add_account":
			var msg struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(data, &msg)
			return map[string]any{
				"type": "admin_result", "request_id": msg.RequestID, "ok": true, "account_id": int64(5),
			}
		default:
			return nil
		}
	})
	if err := a.SSOAdminAddAccount("box", "secret", ""); err != nil {
		t.Fatal(err)
	}
}

func TestSSOAdminWrappersValidate(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()

	if err := a.SSOAdminAddAccount("", "", ""); err == nil {
		t.Fatal("not connected")
	}
	c := sso.NewClient()
	c.SetStateForTest(sso.TestClientState{Connected: true})
	a.sso = c
	if err := a.SSOAdminAddAccount("user", "pass", ""); err == nil {
		t.Fatal("not admin")
	}

	admin := sso.NewClient()
	admin.SetStateForTest(sso.TestClientState{Connected: true, Admin: true})
	a.sso = admin

	if err := a.SSOAdminAddAccount("", "pass", ""); err == nil {
		t.Fatal("username required")
	}
	if err := a.SSOAdminAddAccount("user", "", ""); err == nil {
		t.Fatal("password required")
	}
	if err := a.SSOAdminUpdateAccount(0, "", false, false, "", false); err == nil {
		t.Fatal("account required")
	}
	if err := a.SSOAdminUpdateAccount(1, "", false, false, "", false); err == nil {
		t.Fatal("nothing to update")
	}
	if err := a.SSOAdminAddAlias("", 1); err == nil {
		t.Fatal("alias required")
	}
	if err := a.SSOAdminAddAlias("a", 0); err == nil {
		t.Fatal("account required")
	}
	if err := a.SSOAdminRemoveTag("", 1); err == nil {
		t.Fatal("tag required")
	}
	if err := a.SSOAdminSetUserRoles(0, nil); err == nil {
		t.Fatal("user required")
	}
}
