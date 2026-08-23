package logwatch

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultIdle is the backup offline timeout: after the last fresh EQ log activity
// a character stays "online" this long. Covers /quit, crashes, and force-close
// where there is no camp line (same idea as p99-login-proxy's 30s mtime gate).
const DefaultIdle = 30 * time.Second

// CampGrace is a faster path when "/camp" is seen in the log. Classic EQ camps
// take ~30s and usually write no final "logged out" line; we expire ~35s after
// the prepare-camp message and ignore combat spam during that window.
const CampGrace = 35 * time.Second

const (
	lineWelcome     = "Welcome to EverQuest!"
	lineEntered     = "You have entered"
	lineCampPrepare = "It will take you about 30 seconds to prepare your camp."
	lineCampAbandon = "You abandon your preparations to camp."
	tailInspectMax  = 4096
)

// Watcher tails eqlog_<Character>_*.txt and tracks in-world characters.
// presence drives SSO heartbeats; busy drives local login blocking (stricter).
type Watcher struct {
	LogsDir  string
	mu       sync.Mutex
	presence map[string]time.Time // character -> SSO heartbeat presence expiry
	busy     map[string]time.Time // character -> local account busy expiry
	camping  map[string]bool      // character currently camping out
	logPath  map[string]string    // character -> eqlog path (for mtime gate)
	idle     time.Duration
	prev     map[string]struct{} // last Poll snapshot (for gone detection)
}

func New(logsDir string) *Watcher {
	return &Watcher{
		LogsDir:  logsDir,
		presence: make(map[string]time.Time),
		busy:     make(map[string]time.Time),
		camping:  make(map[string]bool),
		logPath:  make(map[string]string),
		idle:     DefaultIdle,
		prev:     make(map[string]struct{}),
	}
}

// OnlineNames returns characters with fresh log presence (for SSO heartbeats).
func (w *Watcher) OnlineNames() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.namesLocked(w.presence, time.Now())
}

// BusyNames returns characters that should block local login on their account.
func (w *Watcher) BusyNames() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.namesLocked(w.busy, time.Now())
}

// Poll returns current presence names and any that went offline since the last Poll.
func (w *Watcher) Poll() (online []string, gone []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	online = w.namesLocked(w.presence, now)
	cur := make(map[string]struct{}, len(online))
	for _, name := range online {
		cur[name] = struct{}{}
	}
	for name := range w.prev {
		if _, ok := cur[name]; !ok {
			gone = append(gone, name)
		}
	}
	w.prev = cur
	return online, gone
}

func (w *Watcher) namesLocked(m map[string]time.Time, now time.Time) []string {
	var out []string
	for name, exp := range m {
		if now.After(exp) {
			w.clearCharacterLocked(name)
			continue
		}
		if path := w.logPath[name]; path != "" {
			if st, err := os.Stat(path); err == nil {
				if now.Sub(st.ModTime()) > w.idle {
					w.clearCharacterLocked(name)
					continue
				}
			}
		}
		out = append(out, name)
	}
	return out
}

func (w *Watcher) clearCharacterLocked(name string) {
	delete(w.presence, name)
	delete(w.busy, name)
	delete(w.camping, name)
}

func (w *Watcher) noteLogPathLocked(char, path string) {
	if w.logPath == nil {
		w.logPath = make(map[string]string)
	}
	w.logPath[char] = path
}

// OnlineCharacters is an alias for OnlineNames (tests / older call sites).
func (w *Watcher) OnlineCharacters() []string {
	return w.OnlineNames()
}

// SetOnlineForTest marks a character as recently active (for tests).
func (w *Watcher) SetOnlineForTest(character string) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.presence == nil {
		w.presence = make(map[string]time.Time)
	}
	if w.busy == nil {
		w.busy = make(map[string]time.Time)
	}
	if w.camping == nil {
		w.camping = make(map[string]bool)
	}
	delete(w.camping, character)
	exp := time.Now().Add(w.idle)
	w.presence[character] = exp
	w.busy[character] = exp
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	offsets := map[string]int64{}
	seeded := map[string]bool{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(offsets, seeded)
		}
	}
}

func (w *Watcher) scan(offsets map[string]int64, seeded map[string]bool) {
	entries, err := os.ReadDir(w.LogsDir)
	if err != nil {
		return
	}
	now := time.Now()
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

		w.mu.Lock()
		w.noteLogPathLocked(char, path)
		w.mu.Unlock()

		if !seeded[path] {
			seeded[path] = true
			offsets[path] = st.Size()
			// If the log is actively being written, inspect the recent tail once so
			// characters already in-game before the GUI started still get heartbeats.
			if st.Size() > 0 && now.Sub(st.ModTime()) <= w.idle {
				w.applyRecentLogTail(char, path, st.Size(), now)
			}
			continue
		}

		prev := offsets[path]
		if st.Size() < prev {
			w.mu.Lock()
			w.clearCharacterLocked(char)
			w.mu.Unlock()
			prev = 0
		}
		if st.Size() == prev {
			w.mu.Lock()
			w.expireStaleLocked(char, now, st.ModTime())
			w.mu.Unlock()
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
		w.applyLogChunk(char, string(buf), now)
	}
}

func (w *Watcher) expireStaleLocked(char string, now, mod time.Time) {
	stale := func(m map[string]time.Time) bool {
		exp, ok := m[char]
		return ok && (now.After(exp) || now.Sub(mod) > w.idle)
	}
	if stale(w.presence) || stale(w.busy) {
		w.clearCharacterLocked(char)
	}
}

func (w *Watcher) applyRecentLogTail(char, path string, size int64, now time.Time) {
	start := int64(0)
	if size > tailInspectMax {
		start = size - tailInspectMax
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return
	}
	buf := make([]byte, size-start)
	if _, err := io.ReadFull(f, buf); err != nil && err != io.ErrUnexpectedEOF {
		return
	}
	w.applyLogChunk(char, string(buf), now)
}

func (w *Watcher) applyLogChunk(char, text string, now time.Time) {
	if text == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.presence == nil {
		w.presence = make(map[string]time.Time)
	}
	if w.busy == nil {
		w.busy = make(map[string]time.Time)
	}
	if w.camping == nil {
		w.camping = make(map[string]bool)
	}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ts, hasTS := parseLogLineTime(line)
		recent := hasTS && logTimeRecent(ts, now, w.idle)

		switch {
		case strings.Contains(line, lineCampPrepare):
			if !recent {
				continue
			}
			w.camping[char] = true
			exp := now.Add(CampGrace)
			w.presence[char] = exp
			w.busy[char] = exp
			return

		case strings.Contains(line, lineCampAbandon):
			if !recent {
				continue
			}
			delete(w.camping, char)
			exp := now.Add(w.idle)
			w.presence[char] = exp
			w.busy[char] = exp

		case strings.Contains(line, lineWelcome), strings.Contains(line, lineEntered):
			if !recent {
				continue
			}
			delete(w.camping, char)
			exp := now.Add(w.idle)
			w.presence[char] = exp
			w.busy[char] = exp

		default:
			if w.camping[char] || !recent {
				continue
			}
			// Fresh combat/chat marks presence for heartbeats but does not block login.
			w.presence[char] = now.Add(w.idle)
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
