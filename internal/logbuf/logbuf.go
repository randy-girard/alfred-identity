package logbuf

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

const defaultCapacity = 3000

// Entry is one log line for the GUI.
type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Attrs   string `json:"attrs,omitempty"`
}

// Buffer stores recent log lines in memory for the GUI.
type Buffer struct {
	mu       sync.RWMutex
	entries  []Entry
	capacity int
}

// New creates a ring buffer with the given capacity.
func New(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	return &Buffer{capacity: capacity}
}

// Append stores a log entry, evicting the oldest when full.
func (b *Buffer) Append(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) >= b.capacity {
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, e)
}

// Recent returns up to limit newest entries.
func (b *Buffer) Recent(limit int) []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if limit <= 0 || limit > len(b.entries) {
		limit = len(b.entries)
	}
	start := len(b.entries) - limit
	out := make([]Entry, limit)
	copy(out, b.entries[start:])
	return out
}

// Clear removes all buffered entries.
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = nil
}

// NewHandler returns a slog handler that writes to out and appends to buf.
func NewHandler(buf *Buffer, out io.Writer) slog.Handler {
	base := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug})
	return &handler{buf: buf, base: base}
}

type handler struct {
	buf  *Buffer
	base slog.Handler
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.base.Handle(ctx, r); err != nil {
		return err
	}
	if h.buf == nil {
		return nil
	}
	attrs := make([]string, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
		return true
	})
	entry := Entry{
		Time:    r.Time.Format(time.RFC3339),
		Level:   r.Level.String(),
		Message: r.Message,
	}
	if len(attrs) > 0 {
		entry.Attrs = fmt.Sprintf("%v", attrs)
	}
	h.buf.Append(entry)
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{buf: h.buf, base: h.base.WithAttrs(attrs)}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{buf: h.buf, base: h.base.WithGroup(name)}
}
