package p99proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(cfg, []byte(`[DEFAULT]
local_accounts_file = my_accounts.csv
local_characters_file = my_chars.csv
eq_directory = C:\EverQuest
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "my_accounts.csv"), []byte("a,b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inst, err := ParseConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !inst.HasAccounts {
		t.Fatal("expected accounts file")
	}
	if inst.AccountsCSV != filepath.Join(dir, "my_accounts.csv") {
		t.Fatalf("accounts=%q", inst.AccountsCSV)
	}
	if inst.EQDirectory != `C:\EverQuest` {
		t.Fatalf("eq=%q", inst.EQDirectory)
	}
}

func TestDiscoverFindsConfigInSubdir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "P99 Login Proxy")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(sub, ConfigFileName)
	if err := os.WriteFile(cfg, []byte("[DEFAULT]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	found := Discover(root)
	if !installPathsContain(found, cfg) {
		t.Fatalf("expected %q in found=%+v", cfg, found)
	}
}

func installPathsContain(found []Install, configPath string) bool {
	for _, inst := range found {
		if inst.ConfigPath == configPath {
			return true
		}
	}
	return false
}

func TestDiscoverFindsConfigTwoLevelsUnderDocuments(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "Documents")
	inst := filepath.Join(docs, "P99LoginProxy")
	if err := os.MkdirAll(inst, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(inst, ConfigFileName)
	if err := os.WriteFile(cfg, []byte("[DEFAULT]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	found := Discover(docs)
	if !installPathsContain(found, cfg) {
		t.Fatalf("expected %q in found=%+v", cfg, found)
	}
}
