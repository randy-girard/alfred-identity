package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckInvalidRepo(t *testing.T) {
	if _, err := Check(context.Background(), "bad", "1.0.0"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckWithServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/app/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.0","html_url":"https://example.com/r"}`))
	}))
	defer srv.Close()

	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()

	res, err := Check(context.Background(), "acme/app", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpdateAvailable || res.Latest != "v1.2.0" || res.ReleaseURL == "" {
		t.Fatalf("%+v", res)
	}

	res, err = Check(context.Background(), "acme/app", "v1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if res.UpdateAvailable {
		t.Fatalf("should be up to date: %+v", res)
	}

	// Current newer than GitHub "latest" must not offer a downgrade.
	res, err = Check(context.Background(), "acme/app", "2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if res.UpdateAvailable || res.CanApply {
		t.Fatalf("should not offer downgrade: %+v", res)
	}
}

func TestCheckNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()

	if _, err := Check(context.Background(), "acme/missing", "0.1.0"); err == nil {
		t.Fatal("expected error when release is missing")
	}
}

func TestCheckHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()
	if _, err := Check(context.Background(), "acme/app", "1.0.0"); err == nil {
		t.Fatal("expected error")
	}
}
