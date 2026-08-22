package eqhost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableDisableRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "eqhost.txt")
	if err := os.WriteFile(orig, []byte("[LoginServer]\nHost=login.example:5998\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Enable(dir, "127.0.0.1:6998"); err != nil {
		t.Fatal(err)
	}
	got := Describe(dir)
	if !strings.Contains(got, "Host=127.0.0.1:6998") {
		t.Fatalf("enable content: %q", got)
	}
	bak := orig + ".bak"
	if _, err := os.Stat(bak); err != nil {
		t.Fatal("expected backup")
	}
	if err := Disable(dir); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(orig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(restored), "login.example:5998") {
		t.Fatalf("restored: %q", restored)
	}
}

func TestDisableWithoutBackup(t *testing.T) {
	dir := t.TempDir()
	if err := Disable(dir); err == nil {
		t.Fatal("expected error")
	}
}
