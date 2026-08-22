package p99proxy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	scanMaxDepth = 12
	scanMaxDirs  = 75000
)

// DiscoverRecursive searches scan roots recursively for proxyconfig.ini.
func DiscoverRecursive(extraDirs ...string) []Install {
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

	// Quick hits first (running process, common paths).
	for _, inst := range Discover(extraDirs...) {
		if seen[inst.ConfigPath] {
			continue
		}
		seen[inst.ConfigPath] = true
		out = append(out, inst)
	}

	visited := 0
	var walk func(root string, depth int)
	walk = func(root string, depth int) {
		if depth > scanMaxDepth || visited >= scanMaxDirs {
			return
		}
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		visited++
		add(filepath.Join(root, ConfigFileName))

		entries, err := os.ReadDir(root)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			child := filepath.Join(root, e.Name())
			if shouldSkipScanDir(child) {
				continue
			}
			walk(child, depth+1)
		}
	}

	for _, root := range scanRoots(extraDirs...) {
		walk(root, 0)
	}
	return out
}

func scanRoots(extra ...string) []string {
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
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			return
		}
		out = append(out, abs)
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(home)
		add(filepath.Join(home, "Applications"))
	}
	switch runtime.GOOS {
	case "darwin":
		add("/Applications")
	case "windows":
		for _, env := range []string{"USERPROFILE", "LOCALAPPDATA", "APPDATA"} {
			add(os.Getenv(env))
		}
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			add(xdg)
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			add(xdg)
		}
	}
	for _, e := range extra {
		add(e)
		add(filepath.Dir(e))
	}
	return out
}

func shouldSkipScanDir(path string) bool {
	base := filepath.Base(path)
	if base == "" || base == "." || base == ".." {
		return true
	}
	if strings.HasPrefix(base, ".") {
		return true
	}
	switch base {
	case "node_modules", "Caches", "Cache", "CachedData", "GPUCache",
		"Code Cache", "Service Worker", "ShaderCache", "GrShaderCache",
		"__pycache__", "site-packages", "venv", ".venv",
		"target", "build", "dist", "vendor", "go", "pkg",
		"Windows", "System32", "SysWOW64", "Program Files", "Program Files (x86)",
		"Containers", "Group Containers", "Saved Application State",
		"Developer", "Xcode", "Android", "AndroidStudio",
		"Trash", ".Trash":
		return true
	}
	// Heavy macOS Library subtrees (keep Application Support searchable).
	if runtime.GOOS == "darwin" {
		parts := strings.Split(filepath.Clean(path), string(os.PathSeparator))
		for i, part := range parts {
			if part != "Library" {
				continue
			}
			if i+1 < len(parts) {
				switch parts[i+1] {
				case "Caches", "Developer", "Containers", "Group Containers",
					"Logs", "Mail", "Messages", "Safari", "Biome", "Metadata",
					"Photos", "Calendars", "HomeKit", "IdentityServices":
					return true
				}
			}
		}
	}
	return false
}
