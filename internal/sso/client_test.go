package sso

import "testing"

func TestClientConnectedStateDisconnect(t *testing.T) {
	c := NewClient()
	if c.Connected() {
		t.Fatal("new client should be disconnected")
	}
	if len(c.State().Accounts) != 0 {
		t.Fatal("empty state")
	}
	c.mu.Lock()
	c.connected = true
	c.isAdmin = true
	c.state = FullState{Accounts: []AccountMeta{{ID: 1, Username: "x"}}}
	c.directory = []DirectoryUser{{ID: 1}}
	c.groups = []GroupDetail{{ID: 1, Name: "g"}}
	c.mu.Unlock()
	if !c.Connected() || !c.IsAdmin() {
		t.Fatal("flags")
	}
	if len(c.State().Accounts) != 1 || len(c.Directory()) != 1 || len(c.Groups()) != 1 {
		t.Fatal("getters")
	}
	c.Disconnect()
	if c.Connected() || c.IsAdmin() {
		t.Fatal("disconnect flags")
	}
	if len(c.State().Accounts) != 0 || len(c.Directory()) != 0 || len(c.Groups()) != 0 {
		t.Fatal("disconnect cleared state")
	}
}
