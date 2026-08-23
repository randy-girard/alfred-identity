package sources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Get().Sources) != 0 {
		t.Fatalf("new install should have empty sources, got %#v", m.Get().Sources)
	}
	if err := m.UpsertSource(Source{
		ID: "1", Name: "Guild", Host: "127.0.0.1:8181", Token: "tok", Notes: "n",
	}, ""); err != nil {
		t.Fatal(err)
	}
	_ = m.Update(func(c *Config) { c.ActiveSourceID = "1"; c.ConnectionMode = ConnectionLoginSSO })

	m2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := m2.Get()
	if len(cfg.Sources) != 1 || cfg.Sources[0].Token != "tok" {
		t.Fatalf("%#v", cfg.Sources)
	}
	if cfg.ActiveSourceID != "1" || cfg.ConnectionMode != ConnectionLoginSSO {
		t.Fatalf("%#v", cfg)
	}
	src, ok := m2.Active()
	if !ok || !src.CanConnect() {
		t.Fatalf("active %#v ok=%v", src, ok)
	}
}

func TestMigrateLegacyProxyAndURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{
  "proxy_enabled": true,
  "sources": [{"id":"x","name":"Old","url":"ws://10.0.0.5:8181/ws/sso","token":"t"}],
  "active_source_id": "x"
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := m.Get()
	if cfg.ConnectionMode != ConnectionLoginSSO {
		t.Fatalf("mode %q", cfg.ConnectionMode)
	}
	if cfg.ProxyEnabled {
		t.Fatal("legacy proxy_enabled should be cleared")
	}
	if cfg.Sources[0].Host != "10.0.0.5:8181" || cfg.Sources[0].URL != "" {
		t.Fatalf("%#v", cfg.Sources[0])
	}
}

func TestMigrateLegacyGitHubRepo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{"github_repo":"alfred-identity/app"}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Get().GitHubRepo != DefaultGitHubRepo {
		t.Fatalf("got %q want %q", m.Get().GitHubRepo, DefaultGitHubRepo)
	}
}

func TestDeleteSource(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(filepath.Join(dir, "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = m.UpsertSource(Source{ID: "a", Name: "A", Host: "127.0.0.1:1", Token: "t"}, "")
	_ = m.UpsertSource(Source{ID: "b", Name: "B", Host: "127.0.0.1:2", Token: "t"}, "")
	_ = m.Update(func(c *Config) { c.ActiveSourceID = "a" })
	if err := m.DeleteSource("a"); err != nil {
		t.Fatal(err)
	}
	cfg := m.Get()
	if len(cfg.Sources) != 1 || cfg.Sources[0].ID != "b" {
		t.Fatalf("%#v", cfg.Sources)
	}
}

func TestConnectionModeLabelAndManagerMode(t *testing.T) {
	if ConnectionLoginSSO.Label() != "Login w/ SSO" {
		t.Fatal(ConnectionLoginSSO.Label())
	}
	if ConnectionLoginOnly.Label() != "Login Only" {
		t.Fatal(ConnectionLoginOnly.Label())
	}
	if ConnectionMode("bogus").Label() != "Disabled" {
		t.Fatal("bogus label")
	}
	dir := t.TempDir()
	m, err := Load(filepath.Join(dir, "m.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Update(func(c *Config) { c.ConnectionMode = ConnectionLoginOnly })
	if m.Mode() != ConnectionLoginOnly {
		t.Fatalf("%q", m.Mode())
	}
}

func TestUpsertSourceKeepsTokenUnlessHostChanges(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(filepath.Join(dir, "u.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = m.UpsertSource(Source{ID: "1", Name: "G", Host: "127.0.0.1:8181", Token: "secret"}, "")
	_ = m.UpsertSource(Source{ID: "1", Name: "G", Host: "127.0.0.1:8181", Token: ""}, "127.0.0.1:8181")
	src, _ := m.Active()
	// Active may be empty; find by id
	var found Source
	for _, s := range m.Get().Sources {
		if s.ID == "1" {
			found = s
		}
	}
	if found.Token != "secret" {
		t.Fatalf("token cleared: %#v", found)
	}
	_ = m.UpsertSource(Source{ID: "1", Name: "G", Host: "10.0.0.1:8181", Token: ""}, "127.0.0.1:8181")
	for _, s := range m.Get().Sources {
		if s.ID == "1" && s.Token != "" {
			t.Fatalf("expected token cleared on host change: %#v", s)
		}
	}
	_ = src
}
