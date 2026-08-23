package logwatch

import (
	"testing"
	"time"
)

func TestParseLogLineTime(t *testing.T) {
	now := time.Now()
	ts := now.Format(eqTimeLayout)
	got, ok := parseLogLineTime("[" + ts + "] You have entered West Freeport.")
	if !ok || got.IsZero() {
		t.Fatalf("bracketed: ok=%v t=%v", ok, got)
	}
	got, ok = parseLogLineTime(ts + " Welcome to EverQuest!")
	if !ok || got.IsZero() {
		t.Fatalf("plain: ok=%v t=%v", ok, got)
	}
	if _, ok := parseLogLineTime("You have entered with no timestamp."); ok {
		t.Fatal("expected no timestamp")
	}
}

func TestLogTimeRecent(t *testing.T) {
	now := time.Now()
	if !logTimeRecent(now.Add(-5*time.Second), now, DefaultIdle) {
		t.Fatal("recent line should pass")
	}
	if logTimeRecent(now.Add(-2*time.Minute), now, DefaultIdle) {
		t.Fatal("old line should fail")
	}
	if logTimeRecent(time.Time{}, now, DefaultIdle) {
		t.Fatal("zero time should fail")
	}
}
