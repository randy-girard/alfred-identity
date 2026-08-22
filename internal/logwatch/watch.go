package logwatch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Watcher tails eqlog_<Character>_*.txt and tracks online characters.
type Watcher struct {
	LogsDir string
	mu      sync.Mutex
	online  map[string]time.Time // character -> last activity
	idle    time.Duration
}

func New(logsDir string) *Watcher {
	return &Watcher{
		LogsDir: logsDir,
		online:  make(map[string]time.Time),
		idle:    3 * time.Minute,
	}
}

func (w *Watcher) OnlineCharacters() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	var out []string
	for name, t := range w.online {
		if now.Sub(t) <= w.idle {
			out = append(out, name)
		} else {
			delete(w.online, name)
		}
	}
	return out
}

// SetOnlineForTest marks a character as recently active (for tests).
func (w *Watcher) SetOnlineForTest(character string) {
	if w == nil {
		return
	}
	w.mu.Lock()
	if w.online == nil {
		w.online = make(map[string]time.Time)
	}
	w.online[character] = time.Now()
	w.mu.Unlock()
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	offsets := map[string]int64{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(offsets)
		}
	}
}

func (w *Watcher) scan(offsets map[string]int64) {
	entries, err := os.ReadDir(w.LogsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "eqlog_") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		char := characterFromFilename(name)
		if char == "" {
			continue
		}
		path := filepath.Join(w.LogsDir, name)
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		prev := offsets[path]
		if st.Size() < prev {
			prev = 0
		}
		if st.Size() == prev {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		_, _ = f.Seek(prev, 0)
		buf := make([]byte, st.Size()-prev)
		_, _ = f.Read(buf)
		_ = f.Close()
		offsets[path] = st.Size()
		text := string(buf)
		if strings.Contains(text, "Welcome to EverQuest!") || strings.Contains(text, "You have entered") {
			w.mu.Lock()
			w.online[char] = time.Now()
			w.mu.Unlock()
		}
		// Any new log activity refreshes presence
		if len(buf) > 0 {
			w.mu.Lock()
			if _, ok := w.online[char]; ok {
				w.online[char] = time.Now()
			}
			w.mu.Unlock()
		}
	}
}

func characterFromFilename(name string) string {
	// eqlog_<CharacterName>_<rest>.txt
	name = strings.TrimSuffix(name, ".txt")
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
