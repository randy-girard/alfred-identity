package sources

import (
	"path/filepath"
	"testing"
)

func TestCanConnect(t *testing.T) {
	if (Source{}).CanConnect() {
		t.Fatal("empty")
	}
	if (Source{Host: "127.0.0.1:1"}).CanConnect() {
		t.Fatal("host only")
	}
	if (Source{Token: "t"}).CanConnect() {
		t.Fatal("token only")
	}
	if !(Source{Host: " ws://127.0.0.1:8181/ws ", Token: "t"}).CanConnect() {
		t.Fatal("normalized host+token")
	}
}

func TestDeleteSourceActiveSwitch(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(filepath.Join(dir, "d.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = m.UpsertSource(Source{ID: "a", Name: "A", Host: "127.0.0.1:1", Token: "t"}, "")
	_ = m.UpsertSource(Source{ID: "b", Name: "B", Host: "127.0.0.1:2", Token: "t"}, "")
	_ = m.Update(func(c *Config) { c.ActiveSourceID = "a" })
	if err := m.DeleteSource("a"); err != nil {
		t.Fatal(err)
	}
	if m.Get().ActiveSourceID != "b" {
		t.Fatalf("active=%q", m.Get().ActiveSourceID)
	}
	if err := m.DeleteSource("b"); err != nil {
		t.Fatal(err)
	}
	if m.Get().ActiveSourceID != "" {
		t.Fatalf("expected cleared active, got %q", m.Get().ActiveSourceID)
	}
}
