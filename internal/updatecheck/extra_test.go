package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestApplyEmptyURL(t *testing.T) {
	if err := Apply(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestDarwinAppBundle(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Alfred Identity.app", "Contents", "MacOS", "Alfred Identity")
	got, err := darwinAppBundle(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Dir(filepath.Dir(filepath.Dir(app))) {
		t.Fatalf("got %q", got)
	}
	if _, err := darwinAppBundle(filepath.Join(t.TempDir(), "bin")); err == nil {
		t.Fatal("expected error outside .app")
	}
	bad := filepath.Join(t.TempDir(), "Foo.app", "Weird", "MacOS", "bin")
	if _, err := darwinAppBundle(bad); err == nil {
		t.Fatal("expected unexpected layout error")
	}
}

func TestCheckInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer srv.Close()
	restore := SetAPIBaseForTest(srv.URL)
	defer restore()
	if _, err := Check(context.Background(), "acme/app", "1.0.0"); err == nil {
		t.Fatal("expected decode error")
	}
}
