package main

import (
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
	dir, err := appConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, instanceLockName), nil
}
