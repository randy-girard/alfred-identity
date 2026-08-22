package eqhost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableMapsAllInterfacesListenAddr(t *testing.T) {
	dir := t.TempDir()
	changed, err := Enable(dir, "0.0.0.0:7777")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got := Describe(dir)
	if !strings.Contains(got, "Host=127.0.0.1:7777") {
		t.Fatalf("got %q", got)
	}
}

func TestEnableRejectsEmptyListen(t *testing.T) {
	dir := t.TempDir()
	if _, err := Enable(dir, ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnableDisableRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "eqhost.txt")
	if err := os.WriteFile(orig, []byte("[LoginServer]\nHost=login.example:5998\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := Enable(dir, "127.0.0.1:6998")
	if err != nil || !changed {
		t.Fatalf("enable changed=%v err=%v", changed, err)
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

func TestEnableCreatesWithoutPriorAndSkipsExistingBak(t *testing.T) {
	dir := t.TempDir()
	changed, err := Enable(dir, "127.0.0.1:1")
	if err != nil || !changed {
		t.Fatalf("enable changed=%v err=%v", changed, err)
	}
	if Describe(dir) == "" {
		t.Fatal("expected describe")
	}
	if Describe(filepath.Join(dir, "missing")) != "" {
		t.Fatal("missing dir describe")
	}
	bak := filepath.Join(dir, "eqhost.txt.bak")
	if err := os.WriteFile(bak, []byte("KEEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "eqhost.txt"), []byte("ORIG"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err = Enable(dir, "127.0.0.1:2")
	if err != nil || !changed {
		t.Fatalf("enable changed=%v err=%v", changed, err)
	}
	got, err := os.ReadFile(bak)
	if err != nil || string(got) != "KEEP" {
		t.Fatalf("bak should be preserved: %q %v", got, err)
	}
}

func TestEnableSkipsWhenHostAlreadyMatches(t *testing.T) {
	dir := t.TempDir()
	content := "[LoginServer]\nHost=127.0.0.1:6998\n"
	if err := os.WriteFile(filepath.Join(dir, "eqhost.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := Enable(dir, "127.0.0.1:6998")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no write when host already matches")
	}
	if _, err := os.Stat(filepath.Join(dir, "eqhost.txt.bak")); err == nil {
		t.Fatal("should not create backup when unchanged")
	}
}

func TestEnableSkipsLocalhostHostLine(t *testing.T) {
	dir := t.TempDir()
	content := "Host=localhost:6998\n"
	if err := os.WriteFile(filepath.Join(dir, "eqhost.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := Enable(dir, "127.0.0.1:6998")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected localhost host line to match 127.0.0.1 listen addr")
	}
}

func TestEnableSkipsHostLineWithoutLoginServerSection(t *testing.T) {
	dir := t.TempDir()
	content := "Host=127.0.0.1:6998\n"
	if err := os.WriteFile(filepath.Join(dir, "eqhost.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := Enable(dir, "127.0.0.1:6998")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no write when Host= already points at proxy")
	}
}

func TestReadWriteAndEnsureBackup(t *testing.T) {
	dir := t.TempDir()
	cur, err := Read(dir)
	if err != nil || cur != "" {
		t.Fatalf("missing current: %q err=%v", cur, err)
	}
	if HasBackup(dir) {
		t.Fatal("no backup yet")
	}
	if err := os.WriteFile(filepath.Join(dir, "eqhost.txt"), []byte("ORIG\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, "NEW\n"); err != nil {
		t.Fatal(err)
	}
	bak, err := ReadBackup(dir)
	if err != nil || bak != "ORIG\n" {
		t.Fatalf("backup %q err=%v", bak, err)
	}
	got, err := Read(dir)
	if err != nil || got != "NEW\n" {
		t.Fatalf("current %q err=%v", got, err)
	}
	if err := RestoreBackup(dir); err != nil {
		t.Fatal(err)
	}
	restored, err := Read(dir)
	if err != nil || restored != "ORIG\n" {
		t.Fatalf("restored %q err=%v", restored, err)
	}
}
