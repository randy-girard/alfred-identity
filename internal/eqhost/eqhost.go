package eqhost

import (
	"fmt"
	"net"
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

// Enable writes Host=listen into eqhost.txt after backing up to .bak when needed.
// It returns changed=false when eqhost.txt already points at hostPort.
func Enable(eqDir, hostPort string) (changed bool, err error) {
	hostPort = eqhostClientHost(hostPort)
	if hostPort == "" {
		return false, fmt.Errorf("listen address required")
	}
	want := fmt.Sprintf("%s\nHost=%s\n", section, hostPort)
	cur, err := Read(eqDir)
	if err != nil {
		return false, err
	}
	if eqhostMatchesHost(cur, hostPort) {
		return false, nil
	}
	if err := Write(eqDir, want); err != nil {
		return false, err
	}
	return true, nil
}

// eqhostClientHost is the address EQ should connect to (p99-login-proxy parity).
func eqhostClientHost(listenAddr string) string {
	listenAddr = strings.TrimSpace(listenAddr)
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return listenAddr
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || strings.EqualFold(host, "localhost") || host == "::1" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func eqhostMatchesHost(content, hostPort string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	want := eqhostClientHost(hostPort)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 5 || !strings.EqualFold(line[:5], "host=") {
			continue
		}
		if hostPortsEqual(strings.TrimSpace(line[5:]), want) {
			return true
		}
	}
	return false
}

func hostPortsEqual(a, b string) bool {
	ah, ap, ok1 := splitHostPortNormalized(a)
	bh, bp, ok2 := splitHostPortNormalized(b)
	return ok1 && ok2 && ah == bh && ap == bp
}

func splitHostPortNormalized(hostPort string) (host, port string, ok bool) {
	hostPort = strings.TrimSpace(hostPort)
	h, p, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", "", false
	}
	h = strings.ToLower(strings.TrimSpace(h))
	if h == "localhost" || h == "::1" {
		h = "127.0.0.1"
	}
	return h, p, true
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
