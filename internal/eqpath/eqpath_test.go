package eqpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInstall(t *testing.T) {
	if err := ValidateInstall(""); err == nil {
		t.Fatal("empty should fail")
	}
	dir := t.TempDir()
	if err := ValidateInstall(dir); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "file")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInstall(f); err == nil {
		t.Fatal("file should fail")
	}
}

func TestLogsDirFindsLogs(t *testing.T) {
	root := t.TempDir()
	logs := filepath.Join(root, "Logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := LogsDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != logs {
		t.Fatalf("got %q want %q", got, logs)
	}
}

func TestLogsDirDefaultWhenMissing(t *testing.T) {
	root := t.TempDir()
	got, err := LogsDir(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "Logs")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
