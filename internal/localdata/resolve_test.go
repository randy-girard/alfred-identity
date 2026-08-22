package localdata

import (
	"path/filepath"
	"testing"
)

func TestResolveLoginByCharacterName(t *testing.T) {
	dir := t.TempDir()
	accPath := filepath.Join(dir, "accounts.csv")
	charPath := filepath.Join(dir, "chars.csv")
	s := &Store{AccountsPath: accPath, CharactersPath: charPath}
	if err := s.UpsertAccount("eqbox1", "secret", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertCharacter("eqbox1", "MyHero"); err != nil {
		t.Fatal(err)
	}
	res := s.ResolveLogin("myhero", nil)
	if !res.Matched || res.Chosen == nil {
		t.Fatalf("expected character login match, got %#v", res)
	}
	if res.Chosen.Name != "eqbox1" || res.Chosen.Password != "secret" {
		t.Fatalf("chosen=%#v", res.Chosen)
	}
}
