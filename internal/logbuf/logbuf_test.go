package logbuf

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestBufferRecentAndClear(t *testing.T) {
	b := New(3)
	b.Append(Entry{Message: "a"})
	b.Append(Entry{Message: "b"})
	b.Append(Entry{Message: "c"})
	b.Append(Entry{Message: "d"})

	got := b.Recent(10)
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Message != "b" || got[2].Message != "d" {
		t.Fatalf("got %#v", got)
	}

	b.Clear()
	if len(b.Recent(10)) != 0 {
		t.Fatal("expected empty after clear")
	}
}

func TestHandlerCapturesLogs(t *testing.T) {
	buf := New(10)
	var out bytes.Buffer
	log := slog.New(NewHandler(buf, &out))
	log.Info("hello", "key", "val")

	entries := buf.Recent(1)
	if len(entries) != 1 {
		t.Fatalf("entries=%d", len(entries))
	}
	if entries[0].Message != "hello" {
		t.Fatalf("message=%q", entries[0].Message)
	}
	if !strings.Contains(entries[0].Attrs, "key=val") {
		t.Fatalf("attrs=%q", entries[0].Attrs)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestHandlerEnabled(t *testing.T) {
	buf := New(10)
	h := NewHandler(buf, ioDiscard{})
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("expected info enabled")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
