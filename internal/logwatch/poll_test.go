package logwatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPollReportsGoneWhenPresenceExpires(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Gone_server.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.presenceIdle = time.Minute
	w.busyIdle = 30 * time.Second
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("Welcome to EverQuest!"))
	_ = f.Close()
	w.scan(offsets, seeded)

	online, gone := w.Poll()
	if len(online) != 1 || online[0] != "Gone" || len(gone) != 0 {
		t.Fatalf("first poll online=%v gone=%v", online, gone)
	}

	w.mu.Lock()
	w.presence["Gone"] = time.Now().Add(-time.Second)
	w.mu.Unlock()

	online, gone = w.Poll()
	if len(online) != 0 {
		t.Fatalf("expected offline: %v", online)
	}
	if len(gone) != 1 || gone[0] != "Gone" {
		t.Fatalf("expected gone Gone: %v", gone)
	}

	online, gone = w.Poll()
	if len(online) != 0 || len(gone) != 0 {
		t.Fatalf("second poll should be quiet online=%v gone=%v", online, gone)
	}
}

func TestCampAbandonRestoresPresence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Camper_server.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.presenceIdle = 2 * time.Minute
	w.busyIdle = 30 * time.Second
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)

	appendLine := func(s string) {
		t.Helper()
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString(eqLogLine(s))
		_ = f.Close()
		w.scan(offsets, seeded)
	}

	appendLine("You have entered East Commonlands.")
	if len(w.OnlineNames()) != 1 {
		t.Fatalf("enter: %v", w.OnlineNames())
	}
	appendLine("It will take you about 30 seconds to prepare your camp.")
	w.mu.Lock()
	camping := w.camping["Camper"]
	w.mu.Unlock()
	if !camping {
		t.Fatal("expected camping flag")
	}
	appendLine("You abandon your preparations to camp.")
	w.mu.Lock()
	camping = w.camping["Camper"]
	w.mu.Unlock()
	if camping {
		t.Fatal("camp abandon should clear camping")
	}
	if len(w.OnlineNames()) != 1 {
		t.Fatalf("should stay online after abandon: %v", w.OnlineNames())
	}
}

func TestWelcomeOnNewCharacterClearsPrevious(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "eqlog_FirstChar_server.txt")
	pathB := filepath.Join(dir, "eqlog_SecondChar_server.txt")
	if err := os.WriteFile(pathA, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.presenceIdle = time.Minute
	w.busyIdle = 30 * time.Second
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)

	appendLine := func(path, char, msg string) {
		t.Helper()
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString(eqLogLine(msg))
		_ = f.Close()
		w.scan(offsets, seeded)
	}

	appendLine(pathA, "FirstChar", "Welcome to EverQuest!")
	online, gone := w.Poll()
	if len(online) != 1 || online[0] != "FirstChar" || len(gone) != 0 {
		t.Fatalf("first poll: online=%v gone=%v", online, gone)
	}

	appendLine(pathB, "SecondChar", "You have entered East Commonlands.")
	names := w.OnlineNames()
	if len(names) != 1 || names[0] != "SecondChar" {
		t.Fatalf("expected only SecondChar online, got %v", names)
	}

	online, gone = w.Poll()
	if len(online) != 1 || online[0] != "SecondChar" {
		t.Fatalf("poll online=%v", online)
	}
	if len(gone) != 1 || gone[0] != "FirstChar" {
		t.Fatalf("poll gone=%v want FirstChar", gone)
	}
}
