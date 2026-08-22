//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#include <stdlib.h>

void AIStatusItemStart(const char *tooltip, const unsigned char *png, int pngLen);
void AIStatusItemStop(void);
void AISetDockVisible(int visible);
void AIForceShowMainWindow(void);
*/
import "C"
import (
	"context"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ApplicationMenu() *menu.Menu {
	// Show / Exit live on the menu-bar status item only — not in File.
	m := menu.NewMenu()
	m.Append(menu.AppMenu())
	m.Append(menu.EditMenu())
	m.Append(menu.WindowMenu())
	return m
}

func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	C.AISetDockVisible(1)
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	C.AIForceShowMainWindow()
}

func (a *App) hideWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowHide(a.ctx)
	C.AISetDockVisible(0)
}

// activateForNativeDialog makes the app a regular foreground process so Wails native dialogs work
// when only the menu-bar status item is visible.
func (a *App) activateForNativeDialog() {
	C.AISetDockVisible(1)
}

func (a *App) OnBeforeClose(_ context.Context) bool {
	// runtime.Quit invokes OnBeforeClose; true cancels quit. Only hide when not exiting.
	if a.quitting.Load() {
		return false
	}
	a.hideWindow()
	return true // keep process running (menu bar status item)
}

func (a *App) OnDomReady(_ context.Context) {
	// Ensure the status item is created after AppKit's run loop is up.
	a.startTray()
	a.showWindow()
}

func (a *App) quitApp() {
	a.quitting.Store(true)
	a.stopTray()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) OnSecondInstanceLaunch(_ options.SecondInstanceData) {
	// Wails exits the second process silently; tell the user and focus this one.
	showAlreadyRunningError()
	a.showWindow()
}

//export aiGoShow
func aiGoShow() {
	if globalApp != nil {
		globalApp.showWindow()
	}
}

//export aiGoQuit
func aiGoQuit() {
	if globalApp != nil {
		globalApp.quitApp()
	}
}

//export aiGoCheckUpdates
func aiGoCheckUpdates() {
	if globalApp != nil {
		globalApp.checkUpdateInteractive()
	}
}

func (a *App) startTray() {
	tip := C.CString(AppName)
	defer C.free(unsafe.Pointer(tip))
	var pngPtr *C.uchar
	pngLen := C.int(0)
	if len(trayIcon) > 0 {
		pngPtr = (*C.uchar)(unsafe.Pointer(&trayIcon[0]))
		pngLen = C.int(len(trayIcon))
	}
	// ObjC copies tip/png into NSString/NSData before returning.
	C.AIStatusItemStart(tip, pngPtr, pngLen)
}

func (a *App) stopTray() {
	C.AIStatusItemStop()
}
