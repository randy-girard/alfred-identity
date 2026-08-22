package localdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportAccountsCSV(t *testing.T) {
	dir := t.TempDir()
	store := &Store{
		AccountsPath:   filepath.Join(dir, "accounts.csv"),
		CharactersPath: filepath.Join(dir, "chars.csv"),
	}
	if err := store.UpsertAccount("existing", "oldpass", []string{"ex"}); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(dir, "import.csv")
	body := "name,password,aliases\nexisting,newpass,ex|alias2\nnewbie,secret,nick\n"
	if err := os.WriteFile(src, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	added, updated, err := store.ImportAccountsCSV(src)
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 || updated != 1 {
		t.Fatalf("added=%d updated=%d", added, updated)
	}
	if len(store.Accounts) != 2 {
		t.Fatalf("len=%d", len(store.Accounts))
	}
	for _, a := range store.Accounts {
		if a.Name == "existing" && a.Password != "newpass" {
			t.Fatalf("password not updated: %q", a.Password)
		}
		if a.Name == "newbie" && a.Password != "secret" {
			t.Fatalf("newbie bad: %#v", a)
		}
	}
}

func TestExportAccountsCSVRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := &Store{
		AccountsPath:   filepath.Join(dir, "accounts.csv"),
		CharactersPath: filepath.Join(dir, "chars.csv"),
	}
	if err := store.UpsertAccount("tank", "pw1", []string{"box"}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertAccount("main", "pw2", nil); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "export.csv")
	n, err := store.ExportAccountsCSV(out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	parsed, err := ParseAccountsCSV(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 {
		t.Fatalf("parsed len=%d", len(parsed))
	}
	byName := map[string]Account{}
	for _, a := range parsed {
		byName[a.Name] = a
	}
	if byName["tank"].Password != "pw1" {
		t.Fatalf("tank: %#v", byName["tank"])
	}
	foundAlias := false
	for _, al := range byName["tank"].Aliases {
		if strings.EqualFold(al, "box") {
			foundAlias = true
		}
	}
	if !foundAlias {
		t.Fatalf("missing alias: %#v", byName["tank"].Aliases)
	}
}
