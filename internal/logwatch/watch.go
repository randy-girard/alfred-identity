package logwatch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultIdle is the backup offline timeout: after the last EQ log activity a
// character stays "online" this long. Covers /quit, crashes, and force-close
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
)

// Watcher tails eqlog_<Character>_*.txt and tracks online characters.
type Watcher struct {
	LogsDir string
	mu      sync.Mutex
	online  map[string]time.Time // character -> last activity (or camp deadline)
	camping map[string]bool      // character currently camping out
	idle    time.Duration
	prev    map[string]struct{} // last OnlineCharacters snapshot (for gone detection)
}

func New(logsDir string) *Watcher {
	return &Watcher{
		LogsDir: logsDir,
		online:  make(map[string]time.Time),
		camping: make(map[string]bool),
		idle:    DefaultIdle,
		prev:    make(map[string]struct{}),
	}
}

// OnlineNames returns characters still considered in-world (idle / camp expiry applied).
func (w *Watcher) OnlineNames() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.onlineNamesLocked(time.Now())
}

// Poll returns currently online names and any that went offline since the last Poll
// (idle expiry or finished camp). Use from the heartbeat loop so SSO can clear presence.
func (w *Watcher) Poll() (online []string, gone []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	online = w.onlineNamesLocked(now)
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

func (w *Watcher) onlineNamesLocked(now time.Time) []string {
	var online []string
	for name, exp := range w.online {
		if now.After(exp) {
			delete(w.online, name)
			delete(w.camping, name)
			continue
		}
		online = append(online, name)
	}
	return online
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
	if w.online == nil {
		w.online = make(map[string]time.Time)
	}
	if w.camping == nil {
		w.camping = make(map[string]bool)
	}
	delete(w.camping, character)
	w.online[character] = time.Now().Add(w.idle)
	w.mu.Unlock()
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
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
		w.applyLogChunk(char, string(buf))
	}
}

func (w *Watcher) applyLogChunk(char, text string) {
	if text == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.online == nil {
		w.online = make(map[string]time.Time)
	}
	if w.camping == nil {
		w.camping = make(map[string]bool)
	}

	now := time.Now()
	markedOnline := false
	if strings.Contains(text, lineWelcome) || strings.Contains(text, lineEntered) {
		delete(w.camping, char)
		w.online[char] = now.Add(w.idle)
		markedOnline = true
	}
	if strings.Contains(text, lineCampAbandon) {
		delete(w.camping, char)
		w.online[char] = now.Add(w.idle)
		markedOnline = true
	}
	if strings.Contains(text, lineCampPrepare) {
		// Still in-world for ~30s; ignore further combat/spam refreshes and
		// hard-expire when the camp should have finished (proxy has no equivalent).
		w.camping[char] = true
		w.online[char] = now.Add(CampGrace)
		return
	}
	if markedOnline {
		return
	}
	// Activity only refreshes presence when already online and not camping.
	// This is the quit/crash backup: when the log stops growing, DefaultIdle
	// expires the entry without needing a logout line.
	if _, ok := w.online[char]; !ok {
		return
	}
	if w.camping[char] {
		return
	}
	w.online[char] = now.Add(w.idle)
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
