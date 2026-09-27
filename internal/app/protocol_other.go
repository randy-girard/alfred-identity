//go:build !windows && !linux

package app

// macOS registers alfred-identity:// from Info.plist (wails.json info.protocols).
func registerAppProtocol() {}
