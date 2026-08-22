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
