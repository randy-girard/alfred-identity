package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const instanceLockName = "instance.lock"

// ensureSingleInstance acquires an exclusive process lock. If another instance
// holds it, shows an error dialog and returns false.
func ensureSingleInstance() bool {
	if err := acquireInstanceLock(); err != nil {
		showAlreadyRunningError()
		return false
	}
	return true
}

func instanceLockPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	if dir == "" {
		return "", fmt.Errorf("no config/temp dir for instance lock")
	}
	lockDir := filepath.Join(dir, "alfred-identity-gui")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(lockDir, instanceLockName), nil
}
