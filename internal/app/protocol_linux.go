//go:build linux

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alfred-identity/app/internal/sources"
)

func registerAppProtocol() {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil && resolved != "" {
		exe = resolved
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	desktop := filepath.Join(dir, "alfred-identity.desktop")
	body := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nExec=%q %%u\nStartupNotify=false\nMimeType=x-scheme-handler/%s;\nNoDisplay=true\n",
		AppName, exe, sources.AppURLScheme)
	if err := os.WriteFile(desktop, []byte(body), 0o644); err != nil {
		return
	}
	_ = exec.Command("xdg-mime", "default", "alfred-identity.desktop", "x-scheme-handler/"+sources.AppURLScheme).Run()
	_ = exec.Command("update-desktop-database", dir).Run()
}
