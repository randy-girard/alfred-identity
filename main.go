package main

import (
	"embed"
	"os"

	"github.com/alfred-identity/app/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if !app.EnsureSingleInstance() {
		os.Exit(1)
	}
	defer app.ReleaseSingleInstance()

	a := app.New()

	err := wails.Run(&options.App{
		Title:     app.AppName + " v" + app.Version,
		Width:     920,
		Height:    720,
		MinWidth:  720,
		MinHeight: 560,
		// Close button hides to tray; dock/taskbar only while the window is open.
		HideWindowOnClose: false,
		StartHidden:       false,
		OnBeforeClose:     a.OnBeforeClose,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 245, B: 245, A: 1},
		OnStartup:        a.Startup,
		OnDomReady:       a.OnDomReady,
		OnShutdown:       a.Shutdown,
		Menu:             a.ApplicationMenu(),
		Bind: []interface{}{
			a,
		},
		// Defense in depth: if a second process gets past our lock, focus the first.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "com.alfred-identity.app",
			OnSecondInstanceLaunch: a.OnSecondInstanceLaunch,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   app.AppName,
				Message: "Local login proxy + SSO\nVersion " + app.Version,
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
