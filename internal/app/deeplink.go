package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alfred-identity/app/internal/sources"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const pendingDeepLinkFile = "pending-deeplink"

// HandleOpenURL is called from macOS protocol events and other import URLs.
func (a *App) HandleOpenURL(raw string) {
	raw = strings.Trim(strings.TrimSpace(raw), `"'`)
	if !sources.LooksLikeSourceDeepLink(raw) {
		return
	}
	if a.cfg == nil {
		a.deepLinkMu.Lock()
		a.pendingDeepLink = raw
		a.deepLinkMu.Unlock()
		return
	}
	go a.applySourceDeepLink(raw)
}

func (a *App) handleSecondInstance(data options.SecondInstanceData) {
	a.showWindow()
	if raw := sources.DeepLinkFromArgs(data.Args); raw != "" {
		a.applySourceDeepLink(raw)
		return
	}
	showAlreadyRunningError()
}

func (a *App) pendingDeepLinkLoop(ctxDone <-chan struct{}) {
	a.drainPendingDeepLinks()
	t := time.NewTicker(750 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctxDone:
			return
		case <-t.C:
			a.drainPendingDeepLinks()
		}
	}
}

func (a *App) drainPendingDeepLinks() {
	a.deepLinkMu.Lock()
	queued := a.pendingDeepLink
	a.pendingDeepLink = ""
	a.deepLinkMu.Unlock()
	if queued != "" {
		a.applySourceDeepLink(queued)
	}
	if raw := takePendingDeepLink(); raw != "" {
		a.applySourceDeepLink(raw)
	}
}

func (a *App) applySourceDeepLink(raw string) {
	raw = strings.Trim(strings.TrimSpace(raw), `"'`)
	if raw == "" {
		return
	}
	a.deepLinkMu.Lock()
	if raw == a.lastDeepLink && time.Since(a.lastDeepLinkAt) < 2*time.Second {
		a.deepLinkMu.Unlock()
		return
	}
	a.lastDeepLink = raw
	a.lastDeepLinkAt = time.Now()
	a.deepLinkMu.Unlock()

	if a.cfg == nil {
		a.deepLinkMu.Lock()
		a.pendingDeepLink = raw
		a.deepLinkMu.Unlock()
		return
	}

	a.showWindow()
	src, err := sources.ParseSourceDeepLink(raw)
	if err != nil {
		a.logWarn("sso source deep link parse failed", "err", err)
		a.errorDialog("SSO source link", err.Error())
		return
	}

	existing, found := sources.FindByHost(a.cfg.Get().Sources, src.Host)
	if found {
		src.ID = existing.ID
		if strings.TrimSpace(src.Notes) == "" {
			src.Notes = existing.Notes
		}
		if existing.Token == src.Token && existing.Name == src.Name {
			a.logInfo("sso source already configured", "id", existing.ID, "name", src.Name, "host", src.Host)
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "source-imported")
			}
			return
		}
	} else if strings.TrimSpace(src.Notes) == "" {
		src.Notes = "Imported from Discord"
	}

	accept := "Add"
	title := "Add SSO source"
	msg := fmt.Sprintf("Add SSO source “%s” at %s?\n\nThis stores your token on this computer.", src.Name, src.Host)
	if found {
		accept = "Update"
		title = "Update SSO source"
		msg = fmt.Sprintf("Update SSO source “%s” (%s) with this token?\n\nThis replaces the saved token for that host; it does not add a second source.", src.Name, src.Host)
	}

	ok, err := a.confirmDialog(title, msg, accept)
	if err != nil {
		a.logWarn("sso source deep link confirm failed", "err", err)
		return
	}
	if !ok {
		a.logInfo("sso source deep link cancelled", "name", src.Name, "host", src.Host)
		return
	}

	if _, err := a.SaveSource(src); err != nil {
		a.errorDialog("SSO source link", err.Error())
		return
	}
	a.logInfo("sso source imported from link", "name", src.Name, "host", src.Host, "updated", found)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "source-imported")
	}
}

func pendingDeepLinkPath() (string, error) {
	dir, err := appConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, pendingDeepLinkFile), nil
}

func writePendingDeepLink(raw string) error {
	raw = strings.Trim(strings.TrimSpace(raw), `"'`)
	if raw == "" {
		return nil
	}
	path, err := pendingDeepLinkPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0o600)
}

func takePendingDeepLink() string {
	path, err := pendingDeepLinkPath()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	_ = os.Remove(path)
	return strings.Trim(strings.TrimSpace(string(b)), `"'`)
}
