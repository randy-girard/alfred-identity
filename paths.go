package main

import (
	"os"
	"path/filepath"
)

func appConfigDir() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil || home == "" {
		return "", err
	}
	dir := filepath.Join(home, ConfigDirName)
	migrateLegacyConfigDir(home, dir)
	return dir, nil
}

func migrateLegacyConfigDir(home, dir string) {
	for _, legacyName := range []string{"alfred-identity-gui", "p99-identity-gui"} {
		legacy := filepath.Join(home, legacyName)
		if legacy == dir {
			continue
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if _, err := os.Stat(legacy); err == nil {
				_ = os.Rename(legacy, dir)
				return
			}
		}
	}
}
