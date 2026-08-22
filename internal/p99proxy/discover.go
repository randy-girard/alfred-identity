package p99proxy

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const ConfigFileName = "proxyconfig.ini"

// Install describes a discovered P99 Login Proxy data directory.
type Install struct {
	ConfigPath    string
	ConfigDir     string
	AccountsCSV   string
	CharactersCSV string
	EQDirectory   string
	HasAccounts   bool
}

// ParseConfig reads proxyconfig.ini and resolves local account file paths.
func ParseConfig(configPath string) (Install, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return Install{}, os.ErrInvalid
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return Install{}, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return Install{}, err
	}
	if st.IsDir() {
		abs = filepath.Join(abs, ConfigFileName)
		st, err = os.Stat(abs)
		if err != nil {
			return Install{}, err
		}
	}
	dir := filepath.Dir(abs)
	vals, err := parseINI(abs)
	if err != nil {
		return Install{}, err
	}
	accRel := strings.TrimSpace(vals["local_accounts_file"])
	if accRel == "" {
		accRel = "local_accounts.csv"
	}
	charRel := strings.TrimSpace(vals["local_characters_file"])
	if charRel == "" {
		charRel = "local_characters.csv"
	}
	accPath := accRel
	if !filepath.IsAbs(accRel) {
		accPath = filepath.Join(dir, accRel)
	}
	charPath := charRel
	if !filepath.IsAbs(charRel) {
		charPath = filepath.Join(dir, charRel)
	}
	_, accErr := os.Stat(accPath)
	return Install{
		ConfigPath:    abs,
		ConfigDir:     dir,
		AccountsCSV:   accPath,
		CharactersCSV: charPath,
		EQDirectory:   strings.TrimSpace(vals["eq_directory"]),
		HasAccounts:   accErr == nil,
	}, nil
}

// Discover searches common locations for proxyconfig.ini.
// extraDirs are checked as-is and with one-level children (e.g. EverQuest install).
func Discover(extraDirs ...string) []Install {
	seen := map[string]bool{}
	var out []Install
	add := func(configPath string) {
		configPath = strings.TrimSpace(configPath)
		if configPath == "" {
			return
		}
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return
		}
		if seen[abs] {
			return
		}
		inst, err := ParseConfig(abs)
		if err != nil {
			return
		}
		seen[abs] = true
		out = append(out, inst)
	}
	scan := func(root string, depth int) {
		scanTreeForConfig(root, depth, add)
	}
	for _, dir := range discoverRunningDataDirs() {
		add(filepath.Join(dir, ConfigFileName))
	}
	for _, dir := range wellKnownDirs() {
		scan(dir, 2)
	}
	for _, dir := range candidateDirs(extraDirs...) {
		scan(dir, 1)
	}
	return out
}

func hasConfig(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, ConfigFileName))
	return err == nil && st != nil && !st.IsDir()
}

func wellKnownDirs() []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, "Documents", "P99LoginProxy"),
		filepath.Join(home, "Documents", "p99-login-proxy"),
		filepath.Join(home, "Documents", "P99 Login Proxy"),
		filepath.Join(home, "Downloads", "p99-login-proxy"),
		filepath.Join(home, "Downloads", "P99 Login Proxy"),
		filepath.Join(home, "Desktop", "P99LoginProxy"),
		filepath.Join(home, "Desktop", "p99-login-proxy"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
	}
}

func scanTreeForConfig(root string, depth int, add func(string)) {
	root = strings.TrimSpace(root)
	if root == "" || depth < 0 {
		return
	}
	add(filepath.Join(root, ConfigFileName))
	if depth == 0 {
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		scanTreeForConfig(filepath.Join(root, name), depth-1, add)
	}
}

func candidateDirs(extra ...string) []string {
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		for _, existing := range out {
			if existing == abs {
				return
			}
		}
		out = append(out, abs)
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Dir(exe))
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(home)
		add(filepath.Join(home, "p99-login-proxy"))
		add(filepath.Join(home, "P99 Login Proxy"))
	}
	if cfg, err := os.UserConfigDir(); err == nil {
		add(cfg)
		add(filepath.Join(cfg, "p99-login-proxy"))
		add(filepath.Join(cfg, "P99 Login Proxy"))
	}
	switch runtime.GOOS {
	case "windows":
		if lad := os.Getenv("LOCALAPPDATA"); lad != "" {
			add(filepath.Join(lad, "P99 Login Proxy"))
			add(filepath.Join(lad, "p99-login-proxy"))
		}
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			add(filepath.Join(appdata, "P99 Login Proxy"))
			add(filepath.Join(appdata, "p99-login-proxy"))
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			add(filepath.Join(home, "Applications", "P99 Login Proxy"))
			add(filepath.Join(home, "Library", "Application Support", "P99 Login Proxy"))
		}
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			add(filepath.Join(xdg, "p99-login-proxy"))
		}
	}
	for _, e := range extra {
		add(e)
		add(filepath.Dir(e))
	}
	return out
}

func parseINI(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	vals := map[string]string{}
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		if section != "" && section != "default" {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		vals[key] = val
	}
	return vals, sc.Err()
}
