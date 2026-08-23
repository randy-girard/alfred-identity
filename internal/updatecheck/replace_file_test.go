package updatecheck

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReplaceFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "appbin")
	src := filepath.Join(dir, "newbin")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(dest, src); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "new" {
		t.Fatalf("%q err=%v", b, err)
	}
}

func TestCopyTreeWithFileAndDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "f.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "sub", "f.txt"))
	if err != nil || string(b) != "hi" {
		t.Fatalf("%q err=%v", b, err)
	}
}

func TestClearQuarantineNonDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		dir := t.TempDir()
		if err := clearQuarantine(dir); err != nil {
			// xattr may still succeed on temp dirs
			t.Log(err)
		}
		return
	}
	if err := clearQuarantine("/tmp"); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceInstallFileKind(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin")
	src := filepath.Join(dir, "payload")
	_ = os.WriteFile(dest, []byte("old"), 0o755)
	_ = os.WriteFile(src, []byte("new"), 0o755)
	if err := replaceInstall(installTarget{Path: dest, Kind: "file"}, src); err != nil {
		t.Fatal(err)
	}
}
