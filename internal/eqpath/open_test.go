package eqpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenInFileManagerValidation(t *testing.T) {
	if err := OpenInFileManager(""); err == nil {
		t.Fatal("empty path")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := OpenInFileManager(f); err == nil {
		t.Fatal("file should fail")
	}
	if err := OpenInFileManager(dir); err != nil {
		t.Fatal(err)
	}
}
