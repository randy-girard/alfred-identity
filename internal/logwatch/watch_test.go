package logwatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func eqLogLine(msg string) string {
	ts := time.Now().Format(eqTimeLayout)
	return "[" + ts + "] " + msg + "\n"
}

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
	w.presenceIdle = time.Minute
	w.busyIdle = time.Minute
	now := time.Now()
	w.presence["Hero"] = now.Add(time.Minute)
	w.presence["Idle"] = now.Add(-time.Second)
	got := w.OnlineCharacters()
	if len(got) != 1 || got[0] != "Hero" {
		t.Fatalf("%#v", got)
	}
	if _, ok := w.presence["Idle"]; ok {
		t.Fatal("Idle should be pruned")
	}
}

func TestScanSkipsStaleHistoricalContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Tank_server.txt")
	oldTS := time.Now().Add(-10 * time.Minute).Format(eqTimeLayout)
	if err := os.WriteFile(path, []byte("["+oldTS+"] Welcome to EverQuest!\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Minute)
	_ = os.Chtimes(path, old, old)
	w := New(dir)
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)
	if len(w.OnlineCharacters()) != 0 {
		t.Fatalf("stale historical welcome must not mark online: %#v", w.OnlineCharacters())
	}
	if offsets[path] == 0 {
		t.Fatal("expected tail offset seeded")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("Welcome to EverQuest!"))
	_ = f.Close()
	w.scan(offsets, seeded)
	online := w.OnlineCharacters()
	if len(online) != 1 || online[0] != "Tank" {
		t.Fatalf("%#v", online)
	}
}

func TestScanRecoversActiveSessionOnStartup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Veseelbox_server.txt")
	body := eqLogLine("You have entered West Freeport.") +
		eqLogLine("You hit a gnoll for 1 point of damage.")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)
	online := w.OnlineNames()
	if len(online) != 1 || online[0] != "Veseelbox" {
		t.Fatalf("expected restart recovery presence: %#v", online)
	}
	busy := w.BusyNames()
	if len(busy) != 1 || busy[0] != "Veseelbox" {
		t.Fatalf("enter should also mark busy: %#v", busy)
	}
}

func TestScanCombatMarksPresenceNotBusy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Hero_server.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded) // seed empty
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("You hit a gnoll for 1 point of damage."))
	_ = f.Close()
	w.scan(offsets, seeded)
	if len(w.OnlineNames()) != 1 {
		t.Fatalf("combat should mark presence: %#v", w.OnlineNames())
	}
	if len(w.BusyNames()) != 0 {
		t.Fatalf("combat alone must not block login: %#v", w.BusyNames())
	}
}

func TestScanIgnoresUntimestampedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Zone_server.txt")
	if err := os.WriteFile(path, []byte("You have entered West Freeport.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded) // seed
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("You have entered West Freeport.\n")
	_ = f.Close()
	w.scan(offsets, seeded)
	if len(w.OnlineCharacters()) != 0 {
		t.Fatalf("untimestamped enter must not mark online: %#v", w.OnlineCharacters())
	}
}

func TestScanYouHaveEntered(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Zone_server.txt")
	if err := os.WriteFile(path, []byte(eqLogLine("You have entered West Freeport.")), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded) // seed tail
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("You have entered West Freeport."))
	_ = f.Close()
	w.scan(offsets, seeded)
	online := w.OnlineCharacters()
	if len(online) != 1 || online[0] != "Zone" {
		t.Fatalf("%#v", online)
	}
}

func TestScanCampPrepareExpiresWithoutActivityRefresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Camper_server.txt")
	if err := os.WriteFile(path, []byte(eqLogLine("Welcome to EverQuest!")), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.presenceIdle = time.Minute
	w.busyIdle = time.Minute
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
	online, _ := w.Poll()
	if len(online) != 1 {
		t.Fatalf("expected online after welcome")
	}

	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("It will take you about 30 seconds to prepare your camp.") +
		eqLogLine("You hit a gnoll for 1 point of damage."))
	_ = f.Close()
	w.scan(offsets, seeded)
	online, _ = w.Poll()
	if len(online) != 1 {
		t.Fatalf("expected still online while camping: %v", online)
	}

	w.mu.Lock()
	exp := w.presence["Camper"]
	camping := w.camping["Camper"]
	w.mu.Unlock()
	if !camping {
		t.Fatal("expected camping flag")
	}
	if time.Until(exp) > CampGrace || time.Until(exp) < CampGrace-5*time.Second {
		t.Fatalf("camp expiry skew: until=%v", time.Until(exp))
	}

	w.mu.Lock()
	w.presence["Camper"] = time.Now().Add(-time.Second)
	w.mu.Unlock()
	online, gone := w.Poll()
	if len(online) != 0 {
		t.Fatalf("online=%v", online)
	}
	if len(gone) != 1 || gone[0] != "Camper" {
		t.Fatalf("gone=%v", gone)
	}
}

func TestScanCampAbandon(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Stay_server.txt")
	if err := os.WriteFile(path, []byte(
		eqLogLine("Welcome to EverQuest!")+eqLogLine("It will take you about 30 seconds to prepare your camp."),
	), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("Welcome to EverQuest!") +
		eqLogLine("It will take you about 30 seconds to prepare your camp."))
	_ = f.Close()
	w.scan(offsets, seeded)
	if !w.camping["Stay"] {
		t.Fatal("expected camping")
	}
	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("You abandon your preparations to camp."))
	_ = f.Close()
	w.scan(offsets, seeded)
	if w.camping["Stay"] {
		t.Fatal("camping should clear on abandon")
	}
	if len(w.OnlineNames()) != 1 {
		t.Fatal("should remain online after abandon")
	}
}

func TestScanTruncateRequiresRecentTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Hero_server.txt")
	if err := os.WriteFile(path, []byte(eqLogLine("Welcome to EverQuest!")+"extra\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
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

	// Truncate with stale (untimestamped) enter must not mark online.
	if err := os.WriteFile(path, []byte("You have entered East Commonlands.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.scan(offsets, seeded)
	if len(w.OnlineCharacters()) != 0 {
		t.Fatalf("stale truncate should not mark online: %#v", w.OnlineCharacters())
	}

	// Fresh timestamped enter appended after truncate does mark online.
	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("You have entered East Commonlands."))
	_ = f.Close()
	w.scan(offsets, seeded)
	st, _ := os.Stat(path)
	if offsets[path] != st.Size() {
		t.Fatalf("offset=%d size=%d", offsets[path], st.Size())
	}
	online := w.OnlineCharacters()
	if len(online) != 1 || online[0] != "Hero" {
		t.Fatalf("%#v", online)
	}
}

func TestMtimeStaleClearsOnline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_Hero_server.txt")
	if err := os.WriteFile(path, []byte(eqLogLine("Welcome to EverQuest!")), 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.presenceIdle = 30 * time.Second
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
	if len(w.OnlineCharacters()) != 1 {
		t.Fatalf("expected online: %#v", w.OnlineCharacters())
	}

	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	w.scan(offsets, seeded)
	if len(w.OnlineCharacters()) != 0 {
		t.Fatalf("stale mtime should clear online: %#v", w.OnlineCharacters())
	}
}

func TestHardQuitClearsAfterPresenceIdle(t *testing.T) {
	// /q and force-close usually write no camp line; presence must fall off on idle.
	w := New(t.TempDir())
	if w.presenceIdle != DefaultPresenceIdle {
		t.Fatalf("presenceIdle=%v want default %v", w.presenceIdle, DefaultPresenceIdle)
	}
	w.SetOnlineForTest("Quitter")
	if online, _ := w.Poll(); len(online) != 1 {
		t.Fatalf("expected online before idle: %#v", online)
	}
	w.mu.Lock()
	w.presence["Quitter"] = time.Now().Add(-time.Second)
	w.mu.Unlock()
	if online, gone := w.Poll(); len(online) != 0 || len(gone) != 1 || gone[0] != "Quitter" {
		t.Fatalf("expected Quitter gone after idle online=%v gone=%v", online, gone)
	}
}

func TestQuietZoneKeepsPresenceWithoutBusy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eqlog_AFK_server.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	w.presenceIdle = 2 * time.Minute
	w.busyIdle = 30 * time.Second
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	w.scan(offsets, seeded)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(eqLogLine("You have entered East Commonlands."))
	_ = f.Close()
	w.scan(offsets, seeded)
	if len(w.OnlineNames()) != 1 || len(w.BusyNames()) != 1 {
		t.Fatalf("expected presence+busy after enter online=%v busy=%v", w.OnlineNames(), w.BusyNames())
	}

	// Simulate quiet zone: expire busy only; presence remains.
	w.mu.Lock()
	w.busy["AFK"] = time.Now().Add(-time.Second)
	w.mu.Unlock()
	if len(w.BusyNames()) != 0 {
		t.Fatal("busy should clear")
	}
	if len(w.OnlineNames()) != 1 {
		t.Fatalf("presence should survive quiet zone: %#v", w.OnlineNames())
	}
}
