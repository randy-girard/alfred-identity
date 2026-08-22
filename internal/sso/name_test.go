package sso

import "testing"

func TestNameInMetadataIncludesCharacters(t *testing.T) {
	c := NewClient()
	c.mu.Lock()
	c.state = FullState{
		Accounts: []AccountMeta{{
			ID: 1, Username: "eqbox",
			Aliases:    []string{"boxalias"},
			Tags:       []string{"raid"},
			Characters: []string{"MyHero"},
		}},
	}
	c.mu.Unlock()

	for _, name := range []string{"MyHero", "myhero", "eqbox", "boxalias", "raid"} {
		if !c.NameInMetadata(name) {
			t.Fatalf("NameInMetadata(%q) = false", name)
		}
	}
	if c.NameInMetadata("missing") {
		t.Fatal("unexpected match")
	}
}

func TestOnlineAccountIDs(t *testing.T) {
	c := NewClient()
	c.mu.Lock()
	c.state = FullState{Online: []OnlineEntry{{AccountID: 7}, {AccountID: 8}}}
	c.mu.Unlock()
	got := c.OnlineAccountIDs()
	if !got[7] || !got[8] || len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	c.mu.Lock()
	c.state = FullState{}
	c.mu.Unlock()
	if len(c.OnlineAccountIDs()) != 0 {
		t.Fatal("expected empty")
	}
}
