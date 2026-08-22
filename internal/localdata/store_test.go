package localdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveDeleteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	acc, chars := DefaultPaths(dir)
	s := &Store{AccountsPath: acc, CharactersPath: chars}
	if err := s.UpsertAccount("main", "pw", []string{"box"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertCharacter("main", "Hero"); err != nil {
		t.Fatal(err)
	}

	s2 := &Store{AccountsPath: acc, CharactersPath: chars}
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Accounts) != 1 || s2.Accounts[0].Password != "pw" {
		t.Fatalf("accounts %#v", s2.Accounts)
	}
	if len(s2.Characters) != 1 || s2.Characters[0].Name != "Hero" {
		t.Fatalf("chars %#v", s2.Characters)
	}

	if err := s2.DeleteCharacter("Hero"); err != nil {
		t.Fatal(err)
	}
	if err := s2.DeleteAccount("main"); err != nil {
		t.Fatal(err)
	}
	s3 := &Store{AccountsPath: acc, CharactersPath: chars}
	if err := s3.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s3.Accounts) != 0 || len(s3.Characters) != 0 {
		t.Fatalf("expected empty after delete")
	}
}

func TestResolveLoginAliasAndBusy(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := s.UpsertAccount("acc1", "p1", []string{"shared"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAccount("acc2", "p2", []string{"shared"}); err != nil {
		t.Fatal(err)
	}
	res := s.ResolveLogin("shared", nil)
	if !res.Matched || res.Chosen == nil || !res.ViaAlias {
		t.Fatalf("%#v", res)
	}
	busy := map[string]bool{"acc1": true, "acc2": true}
	res = s.ResolveLogin("shared", busy)
	if !res.Matched || !res.AllBusy || res.Chosen != nil {
		t.Fatalf("busy alias %#v", res)
	}
	res = s.ResolveLogin("acc1", map[string]bool{"acc1": true})
	if !res.Matched || !res.AllBusy || res.Error == "" {
		t.Fatalf("busy exact %#v", res)
	}
	res = s.ResolveLogin("missing", nil)
	if res.Matched {
		t.Fatal("expected no match")
	}
}

func TestLoadMissingFiles(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "missing-a.csv"),
		CharactersPath: filepath.Join(dir, "missing-c.csv"),
	}
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if s.Accounts != nil || s.Characters != nil {
		t.Fatalf("%#v %#v", s.Accounts, s.Characters)
	}
	// empty characters file
	if err := os.WriteFile(s.CharactersPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.loadCharacters(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertAccountKeepsPassword(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := s.UpsertAccount("main", "secret", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAccount("main", "", []string{"nick"}); err != nil {
		t.Fatal(err)
	}
	if s.Accounts[0].Password != "secret" {
		t.Fatalf("%#v", s.Accounts[0])
	}
	found := false
	for _, al := range s.Accounts[0].Aliases {
		if al == "nick" {
			found = true
		}
	}
	if !found {
		t.Fatalf("aliases %#v", s.Accounts[0].Aliases)
	}
}

func TestLoadLegacyCharactersHeader(t *testing.T) {
	dir := t.TempDir()
	acc := filepath.Join(dir, "a.csv")
	chars := filepath.Join(dir, "c.csv")
	if err := os.WriteFile(chars, []byte("name,account\nHero,main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Store{AccountsPath: acc, CharactersPath: chars}
	if err := s.loadCharacters(); err != nil {
		t.Fatal(err)
	}
	if len(s.Characters) != 1 || s.Characters[0].Name != "Hero" || s.Characters[0].Account != "main" {
		t.Fatalf("%#v", s.Characters)
	}
}

func TestUpsertCharacterUpdate(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := s.UpsertCharacter("acc1", "Hero"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertCharacter("acc2", "Hero"); err != nil {
		t.Fatal(err)
	}
	if len(s.Characters) != 1 || s.Characters[0].Account != "acc2" {
		t.Fatalf("%#v", s.Characters)
	}
	if err := s.UpsertCharacter("", "Hero"); err == nil {
		t.Fatal("expected account required")
	}
}

func TestDeleteAccountMissingAndExportEmpty(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := s.DeleteAccount("nope"); err == nil {
		t.Fatal("expected not found")
	}
	if _, err := s.ExportAccountsCSV(""); err == nil {
		t.Fatal("path required")
	}
	out := filepath.Join(dir, "empty.csv")
	n, err := s.ExportAccountsCSV(out)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestDeleteAccountDropsCharactersAndExport(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := s.UpsertAccount("main", "pw", []string{"box"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertCharacter("main", "Hero"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertAccount("keep", "pw", nil); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "export.csv")
	n, err := s.ExportAccountsCSV(out)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := s.DeleteAccount("main"); err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts) != 1 || s.Accounts[0].Name != "keep" {
		t.Fatalf("%#v", s.Accounts)
	}
	if len(s.Characters) != 0 {
		t.Fatalf("chars %#v", s.Characters)
	}
}

func TestLoadMissingFilesOK(t *testing.T) {
	dir := t.TempDir()
	s := &Store{
		AccountsPath:   filepath.Join(dir, "missing_a.csv"),
		CharactersPath: filepath.Join(dir, "missing_c.csv"),
	}
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts) != 0 || len(s.Characters) != 0 {
		t.Fatal("expected empty")
	}
}
