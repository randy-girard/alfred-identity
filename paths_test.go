package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyConfigDir(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, "alfred-identity-gui")
	target := filepath.Join(home, ConfigDirName)
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "marker"), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	migrateLegacyConfigDir(home, target)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("legacy should be renamed away")
	}
}
