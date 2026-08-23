package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const DefaultWSPath = "/ws/sso"

// DefaultGitHubRepo is the owner/repo used for release update checks.
const DefaultGitHubRepo = "randy-girard/alfred-identity"

// ConnectionMode controls the UDP proxy and SSO client.
type ConnectionMode string

const (
	ConnectionDisabled  ConnectionMode = "disabled"    // proxy off, SSO off
	ConnectionLoginOnly ConnectionMode = "login_only"  // proxy on, local + passthrough only
	ConnectionLoginSSO  ConnectionMode = "login_sso"   // proxy on + SSO
)

func NormalizeConnectionMode(m ConnectionMode) ConnectionMode {
	switch m {
	case ConnectionDisabled, ConnectionLoginOnly, ConnectionLoginSSO:
		return m
	default:
		return ConnectionDisabled
	}
}

func (m ConnectionMode) WantsProxy() bool {
	m = NormalizeConnectionMode(m)
	return m == ConnectionLoginOnly || m == ConnectionLoginSSO
}

func (m ConnectionMode) WantsSSO() bool {
	return NormalizeConnectionMode(m) == ConnectionLoginSSO
}

func (m ConnectionMode) Label() string {
	switch NormalizeConnectionMode(m) {
	case ConnectionLoginSSO:
		return "Login w/ SSO"
	case ConnectionLoginOnly:
		return "Login Only"
	default:
		return "Disabled"
	}
}

type Source struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Host  string `json:"host"`
	Token string `json:"token"`
	Notes string `json:"notes,omitempty"`

	// URL is legacy; migrated to Host on load and omitted on save.
	URL string `json:"url,omitempty"`
}

type Config struct {
	Sources         []Source       `json:"sources"`
	ActiveSourceID  string         `json:"active_source_id"`
	EQDirectory     string         `json:"eq_directory"`
	ListenAddr      string         `json:"listen_addr"`
	UpstreamAddr    string         `json:"upstream_addr"`
	ConnectionMode  ConnectionMode `json:"connection_mode"`
	// ProxyEnabled is legacy; migrated into ConnectionMode on load.
	ProxyEnabled  bool   `json:"proxy_enabled,omitempty"`
	AccountsCSV   string `json:"accounts_csv"`
	CharactersCSV string `json:"characters_csv"`
	GitHubRepo    string `json:"github_repo"` // owner/repo for update check
}

type Manager struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

func Load(path string) (*Manager, error) {
	m := &Manager{path: path}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			m.cfg = Config{
				ListenAddr:     "127.0.0.1:6998",
				UpstreamAddr:   "login.eqemulator.net:5998",
				GitHubRepo:     DefaultGitHubRepo,
				ActiveSourceID: "",
				ConnectionMode: ConnectionDisabled,
				Sources:        []Source{},
			}
			return m, m.Save()
		}
		return nil, err
	}
	if err := json.Unmarshal(b, &m.cfg); err != nil {
		return nil, err
	}
	if m.cfg.ListenAddr == "" {
		m.cfg.ListenAddr = "127.0.0.1:6998"
	}
	if m.cfg.UpstreamAddr == "" {
		m.cfg.UpstreamAddr = "login.eqemulator.net:5998"
	}
	migrated := false
	for i := range m.cfg.Sources {
		if migrateSource(&m.cfg.Sources[i]) {
			migrated = true
		}
	}
	if migrateConnectionMode(&m.cfg) {
		migrated = true
	}
	if migrateGitHubRepo(&m.cfg) {
		migrated = true
	}
	if migrated {
		_ = m.Save()
	}
	return m, nil
}

// migrateConnectionMode maps legacy proxy_enabled into connection_mode.
func migrateConnectionMode(c *Config) bool {
	before := c.ConnectionMode
	legacyProxy := c.ProxyEnabled
	if c.ConnectionMode == "" {
		if legacyProxy {
			c.ConnectionMode = ConnectionLoginSSO
		} else {
			c.ConnectionMode = ConnectionDisabled
		}
	}
	c.ConnectionMode = NormalizeConnectionMode(c.ConnectionMode)
	c.ProxyEnabled = false // stop writing the legacy flag
	return c.ConnectionMode != before || legacyProxy
}

// migrateGitHubRepo fixes the legacy placeholder repo used before public releases existed.
func migrateGitHubRepo(c *Config) bool {
	before := strings.TrimSpace(c.GitHubRepo)
	switch before {
	case "", "alfred-identity/app":
		c.GitHubRepo = DefaultGitHubRepo
		return c.GitHubRepo != before
	default:
		return false
	}
}

// migrateSource normalizes Host from Host or legacy URL. Returns true if changed.
func migrateSource(s *Source) bool {
	before := s.Host
	legacy := s.URL
	s.URL = ""
	if s.Host == "" && legacy != "" {
		s.Host = HostFromLegacyURL(legacy)
	} else if s.Host != "" {
		s.Host = NormalizeHost(s.Host)
	}
	return s.Host != before || legacy != ""
}

func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	for i := range m.cfg.Sources {
		m.cfg.Sources[i].URL = ""
		m.cfg.Sources[i].Host = NormalizeHost(m.cfg.Sources[i].Host)
	}
	m.cfg.ConnectionMode = NormalizeConnectionMode(m.cfg.ConnectionMode)
	m.cfg.ProxyEnabled = false
	b, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Update(fn func(*Config)) error {
	m.mu.Lock()
	fn(&m.cfg)
	m.mu.Unlock()
	return m.Save()
}

func (m *Manager) Mode() ConnectionMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return NormalizeConnectionMode(m.cfg.ConnectionMode)
}

func (m *Manager) Active() (Source, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.cfg.Sources {
		if s.ID == m.cfg.ActiveSourceID {
			return s, true
		}
	}
	return Source{}, false
}

// UpsertSource adds or updates a source. Empty Token on update keeps the previous
// token unless the host changed (then the token must be re-pasted).
func (m *Manager) UpsertSource(src Source, previousHost string) error {
	src.Host = NormalizeHost(src.Host)
	src.URL = ""
	return m.Update(func(c *Config) {
		for i := range c.Sources {
			if c.Sources[i].ID != src.ID {
				continue
			}
			hostChanged := previousHost != "" && NormalizeHost(previousHost) != src.Host
			if src.Token == "" && !hostChanged {
				src.Token = c.Sources[i].Token
			}
			c.Sources[i] = src
			return
		}
		c.Sources = append(c.Sources, src)
	})
}

// DeleteSource removes a source by ID. If it was active, ActiveSourceID is cleared
// or moved to another remaining source.
func (m *Manager) DeleteSource(id string) error {
	return m.Update(func(c *Config) {
		out := c.Sources[:0]
		for _, s := range c.Sources {
			if s.ID != id {
				out = append(out, s)
			}
		}
		c.Sources = out
		if c.ActiveSourceID != id {
			return
		}
		c.ActiveSourceID = ""
		if len(c.Sources) > 0 {
			c.ActiveSourceID = c.Sources[0].ID
		}
	})
}

// NormalizeHost strips schemes/paths so only host[:port] remains.
func NormalizeHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	for _, p := range []string{"wss://", "ws://", "https://", "http://"} {
		if strings.HasPrefix(lower, p) {
			raw = raw[len(p):]
			lower = strings.ToLower(raw)
			break
		}
	}
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}

// HostFromLegacyURL extracts host:port from a full ws/http URL.
func HostFromLegacyURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	return NormalizeHost(raw)
}

// WebSocketURL builds ws(s)://{host}/ws/sso from a stored host.
// Loopback and private LAN hosts use ws://; public hosts use wss://.
// An explicit ws:// or wss:// prefix in raw overrides the scheme.
func WebSocketURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("host required")
	}
	scheme := ""
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "wss://"):
		scheme = "wss"
		raw = raw[6:]
	case strings.HasPrefix(lower, "ws://"):
		scheme = "ws"
		raw = raw[5:]
	case strings.HasPrefix(lower, "https://"):
		scheme = "wss"
		raw = raw[8:]
	case strings.HasPrefix(lower, "http://"):
		scheme = "ws"
		raw = raw[7:]
	}
	host := NormalizeHost(raw)
	if host == "" {
		return "", fmt.Errorf("host required")
	}
	if scheme == "" {
		if preferPlainWS(host) {
			scheme = "ws"
		} else {
			scheme = "wss"
		}
	}
	return scheme + "://" + host + DefaultWSPath, nil
}

func preferPlainWS(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	h := strings.ToLower(host)
	return h == "localhost" || strings.HasSuffix(h, ".local")
}

// ParseImportSources reads one or more distributable SSO source definitions from JSON.
// Accepted shapes:
//   {"name","host","token?","notes?"}
//   {"source":{...}}
//   {"sources":[{...}, ...]}
//   [{...}, ...]
// Internal "id" fields are ignored (a new id is assigned on save).
func ParseImportSources(data []byte) ([]Source, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	normalize := func(raw map[string]any) (Source, error) {
		get := func(keys ...string) string {
			for _, k := range keys {
				if v, ok := raw[k]; ok {
					switch t := v.(type) {
					case string:
						return strings.TrimSpace(t)
					}
				}
			}
			return ""
		}
		s := Source{
			Name:  get("name"),
			Host:  NormalizeHost(get("host", "url")),
			Token: get("token"),
			Notes: get("notes"),
		}
		if s.Name == "" {
			return Source{}, fmt.Errorf("name required")
		}
		if s.Host == "" {
			return Source{}, fmt.Errorf("host required")
		}
		return s, nil
	}

	var asMap map[string]any
	if err := json.Unmarshal(data, &asMap); err == nil {
		if nested, ok := asMap["sources"]; ok {
			b, _ := json.Marshal(nested)
			return ParseImportSources(b)
		}
		if nested, ok := asMap["source"]; ok {
			b, _ := json.Marshal(nested)
			return ParseImportSources(b)
		}
		s, err := normalize(asMap)
		if err != nil {
			return nil, err
		}
		return []Source{s}, nil
	}

	var asArr []map[string]any
	if err := json.Unmarshal(data, &asArr); err != nil {
		return nil, fmt.Errorf("expected a source object or array of sources")
	}
	if len(asArr) == 0 {
		return nil, fmt.Errorf("no sources in file")
	}
	out := make([]Source, 0, len(asArr))
	for i, raw := range asArr {
		s, err := normalize(raw)
		if err != nil {
			return nil, fmt.Errorf("sources[%d]: %w", i, err)
		}
		out = append(out, s)
	}
	return out, nil
}

// CanConnect reports whether a source has host + token.
func (s Source) CanConnect() bool {
	return NormalizeHost(s.Host) != "" && strings.TrimSpace(s.Token) != ""
}

// DialURL returns the WebSocket URL for this source.
func (s Source) DialURL() (string, error) {
	if s.Host == "" && s.URL != "" {
		return WebSocketURL(HostFromLegacyURL(s.URL))
	}
	return WebSocketURL(s.Host)
}
