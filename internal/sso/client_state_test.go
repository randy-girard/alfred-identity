package sso

import "testing"

func TestSetStateForTest(t *testing.T) {
	c := NewClient()
	c.SetStateForTest(TestClientState{
		Connected: true,
		Admin:     true,
		UserID:    3,
		Accounts:  []AccountMeta{{ID: 1, Username: "a"}},
		Directory: []DirectoryUser{{ID: 1}},
		Groups:    []GroupDetail{{ID: 2, Name: "g"}},
		AdminUsers: []AdminUser{{ID: 1}},
	})
	if !c.Connected() || !c.IsAdmin() || c.UserID() != 3 {
		t.Fatal("flags")
	}
	if len(c.State().Accounts) != 1 || len(c.Directory()) != 1 || len(c.Groups()) != 1 {
		t.Fatal("collections")
	}
	if len(c.Admin().Users) != 1 {
		t.Fatal("admin state")
	}
}
