package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDownloadFileOKAndHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			_, _ = w.Write([]byte("payload"))
			return
		}
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	if err := downloadFile(context.Background(), srv.URL+"/ok", dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "payload" {
		t.Fatalf("%q err=%v", b, err)
	}
	if err := downloadFile(context.Background(), srv.URL+"/bad", filepath.Join(dir, "bad.bin")); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestFindPayloadDarwinApp(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := t.TempDir()
	app := filepath.Join(dir, "Alfred Identity.app", "Contents", "MacOS")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Alfred Identity"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findPayload(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "Alfred Identity.app") {
		t.Fatalf("got %q", got)
	}
	empty := t.TempDir()
	if _, err := findPayload(empty); err == nil {
		t.Fatal("expected missing .app error")
	}
}

func TestReplaceInstallUnknownKind(t *testing.T) {
	if err := replaceInstall(installTarget{Kind: "weird"}, "x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestReplaceDirRoundTrip(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "App.app")
	src := filepath.Join(parent, "New.app")
	if err := os.MkdirAll(filepath.Join(dest, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "Contents", "old"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "Contents", "new"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceDir(dest, src); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dest, "Contents", "new"))
	if err != nil || string(b) != "new" {
		t.Fatalf("%q err=%v", b, err)
	}
}
