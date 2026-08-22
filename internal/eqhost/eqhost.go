package eqhost

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const section = "[LoginServer]"

func currentPath(eqDir string) string {
	return filepath.Join(eqDir, "eqhost.txt")
}

func backupPath(eqDir string) string {
	return currentPath(eqDir) + ".bak"
}

// HasBackup reports whether eqhost.txt.bak exists.
func HasBackup(eqDir string) bool {
	_, err := os.Stat(backupPath(eqDir))
	return err == nil
}

// Read returns eqhost.txt contents, or empty string when missing.
func Read(eqDir string) (string, error) {
	b, err := os.ReadFile(currentPath(eqDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadBackup returns eqhost.txt.bak contents, or empty string when missing.
func ReadBackup(eqDir string) (string, error) {
	b, err := os.ReadFile(backupPath(eqDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// EnsureBackup copies eqhost.txt to eqhost.txt.bak when no backup exists yet.
func EnsureBackup(eqDir string) error {
	bak := backupPath(eqDir)
	if _, err := os.Stat(bak); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	cur := currentPath(eqDir)
	orig, err := os.ReadFile(cur)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(bak, orig, 0o644)
}

// Write saves eqhost.txt after ensuring a one-time backup of the previous file.
func Write(eqDir, content string) error {
	if err := EnsureBackup(eqDir); err != nil {
		return err
	}
	return os.WriteFile(currentPath(eqDir), []byte(content), 0o644)
}

// Enable writes Host=listen into eqhost.txt after backing up to .bak.
func Enable(eqDir, hostPort string) error {
	content := fmt.Sprintf("%s\nHost=%s\n", section, hostPort)
	return Write(eqDir, content)
}

// RestoreBackup restores eqhost.txt from .bak when present.
func RestoreBackup(eqDir string) error {
	path := currentPath(eqDir)
	bak := backupPath(eqDir)
	b, err := os.ReadFile(bak)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no eqhost.txt.bak to restore; restart EQ after fixing eqhost.txt manually")
		}
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Disable restores eqhost.txt from .bak when present.
func Disable(eqDir string) error {
	return RestoreBackup(eqDir)
}

func Describe(eqDir string) string {
	b, err := Read(eqDir)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(b)
}
