//go:build windows

package app

import (
	"os"

	"github.com/alfred-identity/app/internal/sources"
	"golang.org/x/sys/windows/registry"
)

func registerAppProtocol() {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return
	}
	scheme := sources.AppURLScheme
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\`+scheme, registry.SET_VALUE)
	if err != nil {
		return
	}
	_ = k.SetStringValue("", "URL:"+AppName)
	_ = k.SetStringValue("URL Protocol", "")
	_ = k.Close()

	cmd, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\`+scheme+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return
	}
	_ = cmd.SetStringValue("", `"`+exe+`" "%1"`)
	_ = cmd.Close()
}
