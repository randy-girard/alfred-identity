package eqpath

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

// OpenInFileManager opens path in the platform file manager.
func OpenInFileManager(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory")
	}
	switch goruntime.GOOS {
	case "darwin":
		return exec.Command("open", abs).Run()
	case "windows":
		return exec.Command("explorer", abs).Run()
	default:
		return exec.Command("xdg-open", abs).Run()
	}
}
