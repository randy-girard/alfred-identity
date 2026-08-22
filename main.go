package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if !ensureSingleInstance() {
		os.Exit(1)
	}
	defer releaseSingleInstance()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     AppName + " v" + Version,
		Width:     920,
		Height:    720,
		MinWidth:  720,
		MinHeight: 560,
		// Close button hides to tray; dock/taskbar only while the window is open.
		HideWindowOnClose: false,
		StartHidden:       false,
		OnBeforeClose:     app.onBeforeClose,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 245, B: 245, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.onDomReady,
		OnShutdown:       app.shutdown,
		Menu:             app.applicationMenu(),
		Bind: []interface{}{
			app,
		},
		// Defense in depth: if a second process gets past our lock, focus the first.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "com.alfred-identity.app",
			OnSecondInstanceLaunch: app.onSecondInstanceLaunch,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   AppName,
				Message: "Local login proxy + SSO\nVersion " + Version,
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
