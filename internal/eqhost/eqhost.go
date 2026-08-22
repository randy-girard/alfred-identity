package eqhost

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const section = "[LoginServer]"

// Enable writes Host=listen into eqhost.txt after backing up to .bak.
func Enable(eqDir, hostPort string) error {
	path := filepath.Join(eqDir, "eqhost.txt")
	bak := path + ".bak"
	orig, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			_ = os.WriteFile(bak, orig, 0o644)
		}
	}
	content := fmt.Sprintf("%s\nHost=%s\n", section, hostPort)
	return os.WriteFile(path, []byte(content), 0o644)
}

// Disable restores eqhost.txt from .bak when present.
func Disable(eqDir string) error {
	path := filepath.Join(eqDir, "eqhost.txt")
	bak := path + ".bak"
	b, err := os.ReadFile(bak)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no eqhost.txt.bak to restore; restart EQ after fixing eqhost.txt manually")
		}
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func Describe(eqDir string) string {
	path := filepath.Join(eqDir, "eqhost.txt")
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
