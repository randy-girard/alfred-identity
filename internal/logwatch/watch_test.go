package logwatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCharacterFromFilename(t *testing.T) {
	if got := characterFromFilename("eqlog_MyHero_P1999Green.txt"); got != "MyHero" {
		t.Fatalf("%q", got)
	}
	if got := characterFromFilename("eqlog_.txt"); got != "" {
		t.Fatalf("empty char: %q", got)
	}
	if got := characterFromFilename("other.txt"); got != "" {
		t.Fatalf("not eqlog: %q", got)
	}
}

func TestOnlineCharactersIdle(t *testing.T) {
	w := New(t.TempDir())
	w.idle = time.Minute
	now := time.Now()
	w.online["Hero"] = now
	w.online["Idle"] = now.Add(-2 * time.Minute)
	got := w.OnlineCharacters()
	if len(got) != 1 || got[0] != "Hero" {
		t.Fatalf("%#v", got)
	}
	if _, ok := w.online["Idle"]; ok {
		t.Fatal("Idle should be pruned")
	}
}

func TestScanWelcome(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Tank_server.txt")
	if err := os.WriteFile(path, []byte("Welcome to EverQuest!\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	w.scan(offsets)
	online := w.OnlineCharacters()
	if len(online) != 1 || online[0] != "Tank" {
		t.Fatalf("%#v", online)
	}
	// no growth → still online, offset unchanged size
	prev := offsets[path]
	w.scan(offsets)
	if offsets[path] != prev {
		t.Fatalf("offset changed without growth")
	}
}

func TestScanYouHaveEntered(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Zone_server.txt")
	if err := os.WriteFile(path, []byte("You have entered West Freeport.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.scan(map[string]int64{})
	online := w.OnlineCharacters()
	if len(online) != 1 || online[0] != "Zone" {
		t.Fatalf("%#v", online)
	}
}

func TestScanTruncateResetsOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Hero_server.txt")
	if err := os.WriteFile(path, []byte("Welcome to EverQuest!\nextra\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	w.scan(offsets)
	if offsets[path] == 0 {
		t.Fatal("expected offset")
	}
	if err := os.WriteFile(path, []byte("You have entered East Commonlands.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.scan(offsets)
	st, _ := os.Stat(path)
	if offsets[path] != st.Size() {
		t.Fatalf("offset=%d size=%d", offsets[path], st.Size())
	}
	online := w.OnlineCharacters()
	if len(online) != 1 || online[0] != "Hero" {
		t.Fatalf("%#v", online)
	}
}
