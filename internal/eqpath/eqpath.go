package eqpath

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LogsDir resolves EQ Logs directory from an install root (Windows / Wine / macOS).
func LogsDir(eqInstall string) (string, error) {
	eqInstall = strings.TrimSpace(eqInstall)
	if eqInstall == "" {
		return "", fmt.Errorf("EQ directory not set")
	}
	candidates := []string{
		filepath.Join(eqInstall, "Logs"),
		filepath.Join(eqInstall, "logs"),
	}
	// Wine: sometimes drive_c/Program Files/.../Logs already in path
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		candidates = append(candidates,
			filepath.Join(eqInstall, "drive_c", "Program Files", "Sony", "EverQuest", "Logs"),
			filepath.Join(eqInstall, "drive_c", "Program Files (x86)", "Sony", "EverQuest", "Logs"),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c, nil
		}
	}
	// Default preferred path even if missing (watcher can wait)
	return filepath.Join(eqInstall, "Logs"), nil
}

func ValidateInstall(eqInstall string) error {
	if strings.TrimSpace(eqInstall) == "" {
		return fmt.Errorf("empty path")
	}
	st, err := os.Stat(eqInstall)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}
