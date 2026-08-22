//go:build windows || linux

package main

import (
	"context"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) applicationMenu() *menu.Menu {
	// Show / Exit live on the system tray only.
	m := menu.NewMenu()
	m.Append(menu.EditMenu())
	return m
}

func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

func (a *App) hideWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowHide(a.ctx)
}

func (a *App) onBeforeClose(_ context.Context) bool {
	if a.quitting.Load() {
		return false
	}
	a.hideWindow()
	return true
}

func (a *App) onDomReady(_ context.Context) {
	a.showWindow()
}

func (a *App) quitApp() {
	a.quitting.Store(true)
	systray.Quit()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) onSecondInstanceLaunch(_ options.SecondInstanceData) {
	showAlreadyRunningError()
	a.showWindow()
}

func (a *App) startTray() {
	go systray.Run(a.onTrayReady, func() {})
}

func (a *App) onTrayReady() {
	if len(trayIcon) > 0 {
		systray.SetIcon(trayIcon)
	}
	systray.SetTitle("Alfred Identity")
	systray.SetTooltip("Alfred Identity")
	mShow := systray.AddMenuItem("Show Window", "Show Alfred Identity")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Quit Alfred Identity")
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				a.showWindow()
			case <-mQuit.ClickedCh:
				a.quitApp()
				return
			}
		}
	}()
}

func (a *App) stopTray() {
	systray.Quit()
}
