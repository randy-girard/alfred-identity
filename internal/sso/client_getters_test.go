package sso

import (
	"context"
	"testing"
)

func TestClientGettersAndLoginAuthDisconnected(t *testing.T) {
	c := NewClient()
	if c.UserID() != 0 {
		t.Fatal("userid")
	}
	act := c.ShareActivity()
	if act.Logins == nil || act.Online == nil || len(act.Logins) != 0 {
		t.Fatalf("%#v", act)
	}
	if len(c.Admin().Users) != 0 {
		t.Fatal("admin empty")
	}

	c.mu.Lock()
	c.userID = 42
	c.admin = AdminState{Users: []AdminUser{{ID: 1}}}
	c.shareAct = ShareActivity{
		Logins: []ShareLoginEntry{{AccountID: 9}},
		Online: []ShareOnlineEntry{{AccountID: 9}},
	}
	canceled := false
	c.cancel = func() { canceled = true }
	c.mu.Unlock()

	if c.UserID() != 42 {
		t.Fatal("userid set")
	}
	if len(c.Admin().Users) != 1 || len(c.ShareActivity().Logins) != 1 {
		t.Fatal("getters")
	}
	if _, err := c.LoginAuth(context.Background(), "rid", "user"); err == nil {
		t.Fatal("expected not connected")
	}
	c.Disconnect()
	if !canceled {
		t.Fatal("disconnect should cancel")
	}
	if len(c.Admin().Users) != 0 || len(c.ShareActivity().Logins) != 0 {
		t.Fatal("disconnect cleared admin/share")
	}
}
