//go:build !darwin && !windows && !linux

package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ApplicationMenu() *menu.Menu {
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

func (a *App) activateForNativeDialog() {}

func (a *App) OnBeforeClose(_ context.Context) bool {
	if a.quitting.Load() {
		return false
	}
	a.hideWindow()
	return true
}

func (a *App) OnDomReady(_ context.Context) {
	a.showWindow()
}

func (a *App) quitApp() {
	a.quitting.Store(true)
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) OnSecondInstanceLaunch(_ options.SecondInstanceData) {
	showAlreadyRunningError()
	a.showWindow()
}

func (a *App) startTray() {}

func (a *App) stopTray() {}
