//go:build darwin

package app

// showWindowForDialog brings the main window forward so native dialogs receive clicks.
func (a *App) showWindowForDialog() {
	a.showWindow()
}
