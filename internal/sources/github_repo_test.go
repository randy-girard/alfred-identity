package sources

import (
	"os"
	"testing"
)

func TestResolveGitHubRepo(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", DefaultGitHubRepo},
		{"p99-identity/gui", DefaultGitHubRepo},
		{"P99-Identity/GUI", DefaultGitHubRepo},
		{"alfred-identity/app", DefaultGitHubRepo},
		{"randy-girard/alfred-identity", "randy-girard/alfred-identity"},
		{"acme/custom", "acme/custom"},
	}
	for _, tc := range tests {
		if got := ResolveGitHubRepo(tc.in); got != tc.want {
			t.Fatalf("ResolveGitHubRepo(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeGitHubRepoPersists(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.json"
	raw := `{"github_repo":"p99-identity/gui"}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Get().GitHubRepo != DefaultGitHubRepo {
		t.Fatalf("after load: %q", m.Get().GitHubRepo)
	}
	m2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Get().GitHubRepo != DefaultGitHubRepo {
		t.Fatalf("after reload: %q", m2.Get().GitHubRepo)
	}
}
