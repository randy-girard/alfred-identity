package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alfred-identity/app/internal/eqhost"
	"github.com/alfred-identity/app/internal/eqpath"
	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/logbuf"
	"github.com/alfred-identity/app/internal/logwatch"
	"github.com/alfred-identity/app/internal/proxy"
	"github.com/alfred-identity/app/internal/router"
	"github.com/alfred-identity/app/internal/sources"
	"github.com/alfred-identity/app/internal/sso"
	"github.com/alfred-identity/app/internal/updatecheck"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const Version = "0.1.0"

// dialogConfirmed reports whether the user chose the affirmative action.
// Windows QuestionDialog ignores custom Buttons and returns Yes/No/Ok/Cancel.
func dialogConfirmed(selection string, accept ...string) bool {
	sel := strings.TrimSpace(selection)
	if sel == "" {
		return false
	}
	for _, a := range accept {
		if strings.EqualFold(sel, a) {
			return true
		}
	}
	switch strings.ToLower(sel) {
	case "ok", "yes", "delete", "remove", "revoke":
		return true
	default:
		return false
	}
}

// confirmDialog shows a native question dialog. accept is the primary button label
// (also matched case-insensitively along with Ok/Yes on Windows).
func (a *App) confirmDialog(title, message, accept string) (bool, error) {
	if a.ctx == nil {
		return true, nil
	}
	cancel := "Cancel"
	sel, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         title,
		Message:       message,
		Buttons:       []string{accept, cancel},
		DefaultButton: cancel,
		CancelButton:  cancel,
	})
	if err != nil {
		return false, err
	}
	return dialogConfirmed(sel, accept), nil
}

// App is the native Wails backend.
type App struct {
	ctx       context.Context
	log       *slog.Logger
	logBuf    *logbuf.Buffer
	cfg       *sources.Manager
	local     *localdata.Store
	sso       *sso.Client
	proxy     *proxy.Server
	watcher   *logwatch.Watcher
	hbCancel  context.CancelFunc
	quitting  atomic.Bool
}

// globalApp lets the macOS status-item C callbacks reach the running App.
var globalApp *App

func NewApp() *App {
	buf := logbuf.New(3000)
	a := &App{
		logBuf: buf,
		log:    slog.New(logbuf.NewHandler(buf, os.Stdout)),
		sso:    sso.NewClient(),
	}
	globalApp = a
	return a
}

// GetLogs returns recent GUI log lines for the Logs tab.
func (a *App) GetLogs(limit int) []logbuf.Entry {
	if a.logBuf == nil {
		return nil
	}
	if limit <= 0 {
		limit = 500
	}
	if limit > 2000 {
		limit = 2000
	}
	return a.logBuf.Recent(limit)
}

// ClearLogs removes buffered GUI log lines.
func (a *App) ClearLogs() {
	if a.logBuf != nil {
		a.logBuf.Clear()
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Menu-bar / tray icon as early as possible (safe if DomReady calls again).
	a.startTray()

	home, _ := os.UserConfigDir()
	dir := filepath.Join(home, "alfred-identity-gui")
	// Migrate legacy config directory from the old product name.
	if legacy := filepath.Join(home, "p99-identity-gui"); legacy != dir {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if _, err := os.Stat(legacy); err == nil {
				_ = os.Rename(legacy, dir)
			}
		}
	}
	_ = os.MkdirAll(dir, 0o700)

	cfgPath := filepath.Join(dir, "config.json")
	mgr, err := sources.Load(cfgPath)
	if err != nil {
		a.log.Error("config", "err", err)
		return
	}
	a.cfg = mgr
	cfg := mgr.Get()

	accPath, charPath := cfg.AccountsCSV, cfg.CharactersCSV
	if accPath == "" {
		accPath, charPath = localdata.DefaultPaths(dir)
	}
	a.local = &localdata.Store{AccountsPath: accPath, CharactersPath: charPath}
	_ = a.local.Load()

	if cfg.EQDirectory != "" {
		a.startWatcher(cfg.EQDirectory)
	}

	if err := a.applyConnectionMode(cfg.ConnectionMode, false); err != nil {
		a.log.Warn("connection mode auto-start", "err", err, "mode", cfg.ConnectionMode)
	}

	hbCtx, cancel := context.WithCancel(ctx)
	a.hbCancel = cancel
	go a.heartbeatLoop(hbCtx)
	go a.ssoReconnectLoop(hbCtx)

	go a.checkUpdateOnStartup()
}

func (a *App) shutdown(ctx context.Context) {
	if a.hbCancel != nil {
		a.hbCancel()
	}
	a.stopTray()
	// Stop listening and restore eqhost; connection_mode is kept for next launch.
	a.stopProxyRuntime(true)
	if a.sso != nil {
		a.sso.Disconnect()
	}
}

func (a *App) startWatcher(eqDir string) {
	logs, err := eqpath.LogsDir(eqDir)
	if err != nil {
		return
	}
	a.watcher = logwatch.New(logs)
	go a.watcher.Run(a.ctx)
}

func (a *App) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if a.watcher == nil || a.sso == nil || !a.sso.Connected() {
				continue
			}
			for _, ch := range a.watcher.OnlineCharacters() {
				_ = a.sso.Heartbeat(ctx, ch, false)
			}
		}
	}
}

// ssoReconnectLoop keeps the last active SSO source connected when mode is login_sso.
func (a *App) ssoReconnectLoop(ctx context.Context) {
	t := time.NewTicker(8 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if a.cfg == nil || !a.cfg.Mode().WantsSSO() {
				continue
			}
			a.ensureActiveSSO()
		}
	}
}

// ensureActiveSSO connects to the configured active source (or first usable source).
func (a *App) ensureActiveSSO() {
	if a.cfg == nil || a.sso == nil || a.ctx == nil {
		return
	}
	if !a.cfg.Mode().WantsSSO() {
		return
	}
	if a.sso.Connected() {
		return
	}
	a.ensureActiveSSOLocked()
}

func (a *App) ensureActiveSSOLocked() {
	if a.cfg == nil || a.sso == nil || a.ctx == nil {
		return
	}
	if !a.cfg.Mode().WantsSSO() {
		return
	}
	if a.sso.Connected() {
		return
	}
	src, ok := a.cfg.Active()
	if !ok || !src.CanConnect() {
		// Fall back to the first source that can connect; remember it as active.
		cfg := a.cfg.Get()
		found := false
		for _, s := range cfg.Sources {
			if s.CanConnect() {
				src = s
				ok = true
				found = true
				_ = a.cfg.Update(func(c *sources.Config) { c.ActiveSourceID = s.ID })
				break
			}
		}
		if !found {
			return
		}
	}
	wsURL, err := src.DialURL()
	if err != nil {
		a.log.Debug("sso reconnect", "err", err, "source", src.ID)
		return
	}
	if err := a.sso.Connect(a.ctx, wsURL, src.Token, "gui/"+Version); err != nil {
		a.log.Debug("sso reconnect", "err", err, "source", src.ID)
	} else {
		a.log.Info("sso connected", "source", src.ID, "name", src.Name)
	}
}

func (a *App) busyLocal() map[string]bool {
	busy := map[string]bool{}
	if a.watcher == nil || a.local == nil {
		return busy
	}
	for _, ch := range a.watcher.OnlineCharacters() {
		for _, c := range a.local.Characters {
			if strings.EqualFold(c.Name, ch) {
				busy[strings.ToLower(c.Account)] = true
			}
		}
	}
	return busy
}

func (a *App) checkUpdateOnStartup() {
	time.Sleep(2 * time.Second)
	info, err := a.CheckUpdate()
	if err != nil || !info.UpdateAvailable {
		return
	}
	if a.ctx == nil {
		return
	}
	sel, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Update available",
		Message:       "Version " + info.Latest + " is available (you have " + info.Current + ").\n\nOpen the release page?",
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err == nil && sel == "Yes" && info.ReleaseURL != "" {
		runtime.BrowserOpenURL(a.ctx, info.ReleaseURL)
	}
}

// --- Wails-bound API ---

type StatusDTO struct {
	Version         string               `json:"version"`
	ConnectionMode  string               `json:"connection_mode"`
	ProxyEnabled    bool                 `json:"proxy_enabled"`
	SSOConnected    bool                 `json:"sso_connected"`
	SSOIsAdmin      bool                 `json:"sso_is_admin"`
	SSOUserID       int64                `json:"sso_user_id"`
	ActiveSource    string               `json:"active_source"`
	Online          []string             `json:"online"`
	EQDirectory     string               `json:"eq_directory"`
	Listen          string               `json:"listen"`
	SSOAccounts     []sso.AccountMeta    `json:"sso_accounts"`
	SSOOnline       []sso.OnlineEntry    `json:"sso_online"`
	SSODirectory    []sso.DirectoryUser  `json:"sso_directory"`
	SSOGroups       []sso.GroupDetail    `json:"sso_groups"`
	SSOAdminUsers   []sso.AdminUser      `json:"sso_admin_users"`
	SSOAdminRoles   []sso.DiscordRole    `json:"sso_admin_roles"`
	ShareActivity   sso.ShareActivity    `json:"share_activity"`
	Sources         []SourceDTO          `json:"sources"`
}

// SourceDTO is a token-safe view of an SSO source for the UI.
type SourceDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Notes    string `json:"notes,omitempty"`
	HasToken bool   `json:"has_token"`
}

func sourceDTO(s sources.Source) SourceDTO {
	return SourceDTO{
		ID: s.ID, Name: s.Name, Host: s.Host, Notes: s.Notes, HasToken: s.Token != "",
	}
}

func (a *App) GetStatus() StatusDTO {
	if a.cfg == nil {
		return StatusDTO{Version: Version}
	}
	// Pull latest SSO roster while connected (Discord grants may have changed).
	if a.sso != nil && a.sso.Connected() && a.ctx != nil {
		_ = a.sso.RefreshState(a.ctx)
	}
	cfg := a.cfg.Get()
	online := []string{}
	if a.watcher != nil {
		online = a.watcher.OnlineCharacters()
	}
	connected := a.sso != nil && a.sso.Connected()
	isAdmin := connected && a.sso.IsAdmin()
	st := a.sso.State()
	accounts := append([]sso.AccountMeta(nil), st.Accounts...)
	for i := range accounts {
		sort.Slice(accounts[i].Aliases, func(a, b int) bool {
			return strings.ToLower(accounts[i].Aliases[a]) < strings.ToLower(accounts[i].Aliases[b])
		})
		sort.Slice(accounts[i].Tags, func(a, b int) bool {
			return strings.ToLower(accounts[i].Tags[a]) < strings.ToLower(accounts[i].Tags[b])
		})
	}
	sort.Slice(accounts, func(i, j int) bool {
		ni := strings.ToLower(accounts[i].Username)
		nj := strings.ToLower(accounts[j].Username)
		if ni == nj {
			return accounts[i].ID < accounts[j].ID
		}
		if ni == "" {
			return false
		}
		if nj == "" {
			return true
		}
		return ni < nj
	})
	safeSources := make([]SourceDTO, len(cfg.Sources))
	for i, s := range cfg.Sources {
		safeSources[i] = sourceDTO(s)
	}
	mode := sources.NormalizeConnectionMode(cfg.ConnectionMode)
	var adminUsers []sso.AdminUser
	var adminRoles []sso.DiscordRole
	var directory []sso.DirectoryUser
	var groups []sso.GroupDetail
	var shareAct sso.ShareActivity
	var userID int64
	if connected {
		directory = a.sso.Directory()
		groups = a.sso.Groups()
		shareAct = a.sso.ShareActivity()
		userID = a.sso.UserID()
	}
	if shareAct.Logins == nil {
		shareAct.Logins = []sso.ShareLoginEntry{}
	}
	if shareAct.Online == nil {
		shareAct.Online = []sso.ShareOnlineEntry{}
	}
	if isAdmin {
		admin := a.sso.Admin()
		adminUsers = admin.Users
		adminRoles = admin.Roles
	}
	return StatusDTO{
		Version:        Version,
		ConnectionMode: string(mode),
		ProxyEnabled:   a.proxy != nil,
		SSOConnected:   connected,
		SSOIsAdmin:     isAdmin,
		SSOUserID:      userID,
		ActiveSource:   cfg.ActiveSourceID,
		Online:         online,
		EQDirectory:    cfg.EQDirectory,
		Listen:         cfg.ListenAddr,
		SSOAccounts:    accounts,
		SSOOnline:      st.Online,
		SSODirectory:   directory,
		SSOGroups:      groups,
		SSOAdminUsers:  adminUsers,
		SSOAdminRoles:  adminRoles,
		ShareActivity:  shareAct,
		Sources:        safeSources,
	}
}

type LocalAccountDTO struct {
	Name           string   `json:"name"`
	Password       string   `json:"password"`
	Aliases        []string `json:"aliases"`
	HasPass        bool     `json:"has_password"`
	Shared         bool     `json:"shared"`
	SharedUserIDs  []int64  `json:"shared_user_ids"`
	SharedSSOAcct  int64    `json:"shared_sso_account_id"`
	InUse          bool     `json:"in_use"`
	InUseBy        string   `json:"in_use_by,omitempty"`
	InUseOther     bool     `json:"in_use_other"`
	LastLoginAt    string   `json:"last_login_at,omitempty"`
	LastLoginBy    string   `json:"last_login_by,omitempty"`
	LastLoginOther bool     `json:"last_login_other"`
}

type LocalCharacterDTO struct {
	Account string `json:"account"`
	Name    string `json:"name"`
}

func (a *App) GetLocalAccounts() []LocalAccountDTO {
	if a.local == nil {
		return nil
	}
	_ = a.local.Load()
	sharedByName := map[string]sso.AccountMeta{}
	myID := int64(0)
	var shareAct sso.ShareActivity
	if a.sso != nil && a.sso.Connected() {
		if a.ctx != nil {
			_ = a.sso.RefreshState(a.ctx)
		}
		myID = a.sso.UserID()
		shareAct = a.sso.ShareActivity()
		for _, acc := range a.sso.State().Accounts {
			if acc.Restricted && acc.OwnerUserID == myID && myID > 0 {
				sharedByName[strings.ToLower(acc.Username)] = acc
			}
		}
	}
	onlineByAcct := map[int64]sso.ShareOnlineEntry{}
	for _, o := range shareAct.Online {
		onlineByAcct[o.AccountID] = o
	}
	// Prefer most recent login by someone else; fall back to any login.
	lastOtherByAcct := map[int64]sso.ShareLoginEntry{}
	lastAnyByAcct := map[int64]sso.ShareLoginEntry{}
	for _, e := range shareAct.Logins {
		if _, ok := lastAnyByAcct[e.AccountID]; !ok {
			lastAnyByAcct[e.AccountID] = e
		}
		if !e.ActorIsOwner {
			if _, ok := lastOtherByAcct[e.AccountID]; !ok {
				lastOtherByAcct[e.AccountID] = e
			}
		}
	}
	out := make([]LocalAccountDTO, 0, len(a.local.Accounts))
	for _, acc := range a.local.Accounts {
		aliases := make([]string, 0)
		for _, al := range acc.Aliases {
			if !strings.EqualFold(al, acc.Name) {
				aliases = append(aliases, al)
			}
		}
		sort.Slice(aliases, func(i, j int) bool {
			return strings.ToLower(aliases[i]) < strings.ToLower(aliases[j])
		})
		dto := LocalAccountDTO{
			Name: acc.Name, Password: acc.Password, Aliases: aliases, HasPass: acc.Password != "",
			SharedUserIDs: []int64{},
		}
		if sh, ok := sharedByName[strings.ToLower(acc.Name)]; ok {
			dto.Shared = true
			dto.SharedSSOAcct = sh.ID
			dto.SharedUserIDs = append([]int64(nil), sh.SharedUserIDs...)
			if on, ok := onlineByAcct[sh.ID]; ok {
				dto.InUse = true
				dto.InUseOther = !on.ActorIsOwner
				name := strings.TrimSpace(on.UserDisplayName)
				if name == "" {
					name = on.UserDiscordID
				}
				if name == "" {
					name = "someone"
				}
				if on.ActorIsOwner {
					dto.InUseBy = name + " (you)"
				} else {
					dto.InUseBy = name
				}
			}
			if e, ok := lastOtherByAcct[sh.ID]; ok {
				dto.LastLoginOther = true
				dto.LastLoginAt = e.CreatedAt.UTC().Format(time.RFC3339)
				name := strings.TrimSpace(e.ActorName)
				if name == "" {
					name = e.ActorDiscordID
				}
				dto.LastLoginBy = name
			} else if e, ok := lastAnyByAcct[sh.ID]; ok {
				dto.LastLoginAt = e.CreatedAt.UTC().Format(time.RFC3339)
				name := strings.TrimSpace(e.ActorName)
				if name == "" {
					name = e.ActorDiscordID
				}
				if e.ActorIsOwner {
					dto.LastLoginBy = name + " (you)"
				} else {
					dto.LastLoginBy = name
				}
			}
		}
		out = append(out, dto)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (a *App) GetLocalCharacters() []LocalCharacterDTO {
	if a.local == nil {
		return nil
	}
	_ = a.local.Load()
	out := make([]LocalCharacterDTO, 0, len(a.local.Characters))
	for _, c := range a.local.Characters {
		out = append(out, LocalCharacterDTO{Account: c.Account, Name: c.Name})
	}
	return out
}

func (a *App) SaveLocalAccount(name, password string, aliases []string) error {
	if a.local == nil {
		return fmt.Errorf("not ready")
	}
	_ = a.local.Load()
	return a.local.UpsertAccount(name, password, aliases)
}

// ShareLocalAccount publishes a local account to SSO as a restricted share for the given user IDs.
// Empty userIDs keeps the account on SSO for the owner only (clears recipients).
func (a *App) ShareLocalAccount(name string, userIDs []int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if a.local == nil {
		return fmt.Errorf("not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("account required")
	}
	_ = a.local.Load()
	var acc *localdata.Account
	for i := range a.local.Accounts {
		if strings.EqualFold(a.local.Accounts[i].Name, name) {
			acc = &a.local.Accounts[i]
			break
		}
	}
	if acc == nil {
		return fmt.Errorf("local account not found")
	}
	if acc.Password == "" {
		return fmt.Errorf("local account has no password")
	}
	aliases := make([]string, 0)
	for _, al := range acc.Aliases {
		if !strings.EqualFold(al, acc.Name) {
			aliases = append(aliases, al)
		}
	}
	if userIDs == nil {
		userIDs = []int64{}
	}
	_, err := a.sso.ShareAccount(a.ctx, acc.Name, acc.Password, aliases, userIDs)
	return err
}

// UnshareLocalAccount removes the owner's restricted SSO copy of a local account.
func (a *App) UnshareLocalAccount(name string) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.UnshareAccount(a.ctx, name)
	return err
}

// ImportLocalAccountsCSV opens a file picker and merges accounts from a CSV
// (name,password[,aliases] with | -separated aliases).
func (a *App) ImportLocalAccountsCSV() (string, error) {
	if a.local == nil || a.ctx == nil {
		return "", fmt.Errorf("not ready")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import local accounts CSV",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	_ = a.local.Load()
	added, updated, err := a.local.ImportAccountsCSV(path)
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Imported %d new, updated %d existing.", added, updated)
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Import complete",
		Message: msg,
	})
	return msg, nil
}

// ExportLocalAccountsCSV opens a save dialog and writes accounts in import-compatible CSV
// (name,password[,aliases] with | -separated aliases).
func (a *App) ExportLocalAccountsCSV() (string, error) {
	if a.local == nil || a.ctx == nil {
		return "", fmt.Errorf("not ready")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export local accounts CSV",
		DefaultFilename: "local_accounts.csv",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".csv") {
		path += ".csv"
	}
	_ = a.local.Load()
	n, err := a.local.ExportAccountsCSV(path)
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Exported %d account(s) to %s", n, path)
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Export complete",
		Message: msg,
	})
	return msg, nil
}

func (a *App) DeleteLocalAccount(name string) error {
	if a.local == nil {
		return fmt.Errorf("not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("account name required")
	}
	ok, err := a.confirmDialog(
		"Delete local account",
		fmt.Sprintf("Delete local account “%s”? This cannot be undone.", name),
		"Delete",
	)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	_ = a.local.Load()
	return a.local.DeleteAccount(name)
}

func (a *App) SaveLocalCharacter(account, name string) error {
	if a.local == nil {
		return fmt.Errorf("not ready")
	}
	_ = a.local.Load()
	return a.local.UpsertCharacter(account, name)
}

func (a *App) DeleteLocalCharacter(name string) error {
	if a.local == nil {
		return fmt.Errorf("not ready")
	}
	_ = a.local.Load()
	return a.local.DeleteCharacter(name)
}

func (a *App) GetSources() []SourceDTO {
	if a.cfg == nil {
		return nil
	}
	cfg := a.cfg.Get()
	out := make([]SourceDTO, len(cfg.Sources))
	for i, s := range cfg.Sources {
		out[i] = sourceDTO(s)
	}
	return out
}

func (a *App) SSOAdminAddAccount(username, password, requiredRoleID string) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username required")
	}
	if password == "" {
		return fmt.Errorf("password required")
	}
	_, err := a.sso.AdminAddAccount(a.ctx, username, password, strings.TrimSpace(requiredRoleID))
	return err
}

func (a *App) SSOAdminUpdateAccount(accountID int64, password string, disabled bool, setDisabled bool, requiredRoleID string, setRequiredRole bool) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	var pw *string
	if strings.TrimSpace(password) != "" {
		p := password
		pw = &p
	}
	var dis *bool
	if setDisabled {
		dis = &disabled
	}
	var role *string
	if setRequiredRole {
		r := strings.TrimSpace(requiredRoleID)
		role = &r
	}
	if pw == nil && dis == nil && role == nil {
		return fmt.Errorf("nothing to update")
	}
	_, err := a.sso.AdminUpdateAccount(a.ctx, accountID, pw, dis, role)
	return err
}

func (a *App) SSOAdminAddAlias(alias string, accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("alias required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.AdminAddAlias(a.ctx, alias, accountID)
	return err
}

func (a *App) SSOAdminRemoveAlias(alias string, accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("alias required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.AdminRemoveAlias(a.ctx, alias, accountID)
	return err
}

func (a *App) SSOAdminAddTag(tag string, accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.AdminAddTag(a.ctx, tag, accountID)
	return err
}

func (a *App) SSOAdminRemoveTag(tag string, accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.AdminRemoveTag(a.ctx, tag, accountID)
	return err
}

func (a *App) SSOAdminAddCharacter(name string, accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("character name required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.AdminAddCharacter(a.ctx, name, accountID)
	return err
}

func (a *App) SSOAdminRemoveCharacter(name string, accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("character name required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	_, err := a.sso.AdminRemoveCharacter(a.ctx, name, accountID)
	return err
}

func (a *App) SSOAdminRemoveAccount(accountID int64) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	if accountID <= 0 {
		return fmt.Errorf("account required")
	}
	label := fmt.Sprintf("#%d", accountID)
	for _, acc := range a.sso.State().Accounts {
		if acc.ID == accountID {
			if acc.Username != "" {
				label = acc.Username
			}
			break
		}
	}
	ok, err := a.confirmDialog(
		"Remove SSO account",
		fmt.Sprintf("Remove SSO account %s?\n\nAliases, tags, and characters on it are deleted.", label),
		"Remove",
	)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	_, err = a.sso.AdminRemoveAccount(a.ctx, accountID)
	return err
}

func (a *App) SSOAdminSetUserAccess(userID int64, revoked bool) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	if userID <= 0 {
		return fmt.Errorf("user required")
	}
	if revoked && a.ctx != nil {
		label := fmt.Sprintf("user #%d", userID)
		for _, u := range a.sso.Admin().Users {
			if u.ID == userID {
				if u.DisplayName != "" {
					label = u.DisplayName
				} else if u.DiscordID != "" {
					label = u.DiscordID
				}
				break
			}
		}
		sel, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Revoke SSO access",
			Message:       fmt.Sprintf("Revoke SSO access for %s?\n\nThey will be disconnected and cannot create a new token until restored.", label),
			Buttons:       []string{"Revoke", "Cancel"},
			DefaultButton: "Cancel",
			CancelButton:  "Cancel",
		})
		if err != nil {
			return err
		}
		if !dialogConfirmed(sel, "Revoke") {
			return nil
		}
	}
	_, err := a.sso.AdminSetUserAccess(a.ctx, userID, revoked)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "cannot_revoke_self":
			return fmt.Errorf("you cannot revoke your own access")
		case "invalid_user":
			return fmt.Errorf("invalid user")
		case "forbidden":
			return fmt.Errorf("admin access required")
		default:
			return err
		}
	}
	// Pull admin roster so the Users tab reflects access_revoked immediately.
	if a.ctx != nil {
		_ = a.sso.RefreshState(a.ctx)
	}
	return nil
}

func (a *App) SSOAdminSetUserRoles(userID int64, roleIDs []string) error {
	if a.sso == nil || !a.sso.Connected() {
		return fmt.Errorf("not connected to SSO")
	}
	if !a.sso.IsAdmin() {
		return fmt.Errorf("admin access required")
	}
	if userID <= 0 {
		return fmt.Errorf("user required")
	}
	_, err := a.sso.AdminSetUserRoles(a.ctx, userID, roleIDs)
	return err
}

func (a *App) SaveSource(src sources.Source) (SourceDTO, error) {
	if a.cfg == nil {
		return SourceDTO{}, fmt.Errorf("not ready")
	}
	src.Name = strings.TrimSpace(src.Name)
	src.Host = sources.NormalizeHost(src.Host)
	src.Notes = strings.TrimSpace(src.Notes)
	if src.Name == "" {
		return SourceDTO{}, fmt.Errorf("name required")
	}
	if src.Host == "" {
		return SourceDTO{}, fmt.Errorf("host required")
	}
	isNew := src.ID == ""
	if isNew {
		src.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if isNew && strings.TrimSpace(src.Token) == "" {
		return SourceDTO{}, fmt.Errorf("token required for a new source")
	}
	prevHost := ""
	for _, existing := range a.cfg.Get().Sources {
		if existing.ID == src.ID {
			prevHost = existing.Host
			break
		}
	}
	if err := a.cfg.UpsertSource(src, prevHost); err != nil {
		return SourceDTO{}, err
	}

	cfg := a.cfg.Get()
	var saved sources.Source
	for _, s := range cfg.Sources {
		if s.ID == src.ID {
			saved = s
			break
		}
	}

	// Only reconnect when this source is (or becomes) active — activation is chosen on the Connections tab.
	makeActive := cfg.ActiveSourceID == "" || cfg.ActiveSourceID == saved.ID
	if cfg.ActiveSourceID == "" {
		_ = a.cfg.Update(func(c *sources.Config) { c.ActiveSourceID = saved.ID })
		makeActive = true
	}
	if makeActive && a.cfg.Mode().WantsSSO() {
		a.sso.Disconnect()
		if saved.CanConnect() {
			wsURL, err := saved.DialURL()
			if err != nil {
				return sourceDTO(saved), err
			}
			if err := a.sso.Connect(a.ctx, wsURL, saved.Token, "gui/"+Version); err != nil {
				return sourceDTO(saved), err
			}
		}
	}
	return sourceDTO(saved), nil
}

// PreviewSourceJSON parses pasted source JSON for the add-source form. Nothing is saved.
func (a *App) PreviewSourceJSON(raw string) ([]SourceImportPreview, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("json required")
	}
	parsed, err := sources.ParseImportSources([]byte(raw))
	if err != nil {
		return nil, err
	}
	out := make([]SourceImportPreview, 0, len(parsed))
	for _, src := range parsed {
		out = append(out, SourceImportPreview{
			Name:  src.Name,
			Host:  src.Host,
			Notes: src.Notes,
			Token: src.Token,
		})
	}
	return out, nil
}

// SourceImportPreview is a parsed source entry for the add-source form.
type SourceImportPreview struct {
	Name  string `json:"name"`
	Host  string `json:"host"`
	Notes string `json:"notes,omitempty"`
	Token string `json:"token,omitempty"`
}

func (a *App) SetActiveSource(id string) error {
	if a.cfg == nil {
		return nil
	}
	_ = a.cfg.Update(func(c *sources.Config) { c.ActiveSourceID = id })
	a.sso.Disconnect()
	if !a.cfg.Mode().WantsSSO() {
		return nil
	}
	src, ok := a.cfg.Active()
	if !ok || !src.CanConnect() {
		return fmt.Errorf("source has no host or token — save credentials first")
	}
	wsURL, err := src.DialURL()
	if err != nil {
		return err
	}
	return a.sso.Connect(a.ctx, wsURL, src.Token, "gui/"+Version)
}

func (a *App) DeleteSource(id string) error {
	if a.cfg == nil || id == "" {
		return nil
	}
	label := id
	for _, s := range a.cfg.Get().Sources {
		if s.ID == id {
			if s.Name != "" {
				label = s.Name
			}
			break
		}
	}
	ok, err := a.confirmDialog("Remove source", fmt.Sprintf("Remove source “%s”?", label), "Remove")
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	wasActive := a.cfg.Get().ActiveSourceID == id
	if err := a.cfg.DeleteSource(id); err != nil {
		return err
	}
	if !wasActive {
		return nil
	}
	a.sso.Disconnect()
	if !a.cfg.Mode().WantsSSO() {
		return nil
	}
	src, okActive := a.cfg.Active()
	if !okActive || !src.CanConnect() {
		return nil
	}
	wsURL, err := src.DialURL()
	if err != nil {
		return err
	}
	return a.sso.Connect(a.ctx, wsURL, src.Token, "gui/"+Version)
}

func (a *App) SetConnectionMode(mode string) error {
	return a.applyConnectionMode(sources.NormalizeConnectionMode(sources.ConnectionMode(mode)), true)
}

// SetProxyEnabled is kept for older frontends; maps to login_sso / disabled.
func (a *App) SetProxyEnabled(enabled bool) error {
	if enabled {
		return a.SetConnectionMode(string(sources.ConnectionLoginSSO))
	}
	return a.SetConnectionMode(string(sources.ConnectionDisabled))
}

func (a *App) applyConnectionMode(mode sources.ConnectionMode, showEqhostDialog bool) error {
	if a.cfg == nil {
		return nil
	}
	mode = sources.NormalizeConnectionMode(mode)
	prev := a.cfg.Mode()
	if err := a.cfg.Update(func(c *sources.Config) {
		c.ConnectionMode = mode
		c.ProxyEnabled = false
	}); err != nil {
		return err
	}

	switch {
	case !mode.WantsProxy():
		a.stopProxyRuntime(true)
		if a.sso != nil {
			a.sso.Disconnect()
		}
	case mode.WantsProxy() && !prev.WantsProxy():
		if err := a.startProxy(showEqhostDialog); err != nil {
			_ = a.cfg.Update(func(c *sources.Config) { c.ConnectionMode = sources.ConnectionDisabled })
			return err
		}
	case mode.WantsProxy() && a.proxy == nil:
		if err := a.startProxy(showEqhostDialog); err != nil {
			_ = a.cfg.Update(func(c *sources.Config) { c.ConnectionMode = sources.ConnectionDisabled })
			return err
		}
	}

	if mode.WantsSSO() {
		a.ensureActiveSSOLocked()
	} else if a.sso != nil {
		a.sso.Disconnect()
	}
	return nil
}

func (a *App) SetEQDirectory(path string) error {
	if a.cfg == nil {
		return nil
	}
	if err := eqpath.ValidateInstall(path); err != nil {
		return err
	}
	_ = a.cfg.Update(func(c *sources.Config) { c.EQDirectory = path })
	a.startWatcher(path)
	return nil
}

// EqHostState describes eqhost.txt and its backup for the EverQuest tab.
type EqHostState struct {
	Current   string `json:"current"`
	Backup    string `json:"backup"`
	HasBackup bool   `json:"has_backup"`
}

func (a *App) eqInstallDir() (string, error) {
	if a.cfg == nil {
		return "", nil
	}
	dir := strings.TrimSpace(a.cfg.Get().EQDirectory)
	if dir == "" {
		return "", fmt.Errorf("EverQuest directory not set")
	}
	if err := eqpath.ValidateInstall(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// GetEqHostState returns eqhost.txt and backup contents for the configured install.
func (a *App) GetEqHostState() (EqHostState, error) {
	dir, err := a.eqInstallDir()
	if err != nil {
		if strings.Contains(err.Error(), "not set") {
			return EqHostState{}, nil
		}
		return EqHostState{}, err
	}
	cur, err := eqhost.Read(dir)
	if err != nil {
		return EqHostState{}, err
	}
	bak, err := eqhost.ReadBackup(dir)
	if err != nil {
		return EqHostState{}, err
	}
	return EqHostState{
		Current:   cur,
		Backup:    bak,
		HasBackup: eqhost.HasBackup(dir),
	}, nil
}

// SaveEqHostContent writes eqhost.txt after creating a one-time backup when needed.
func (a *App) SaveEqHostContent(content string) error {
	dir, err := a.eqInstallDir()
	if err != nil {
		return err
	}
	if err := eqhost.Write(dir, content); err != nil {
		return err
	}
	if a.ctx != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "eqhost.txt saved",
			Message: "Restart EverQuest for eqhost.txt changes to apply.",
		})
	}
	return nil
}

// RestoreEqHostBackup restores eqhost.txt from eqhost.txt.bak.
func (a *App) RestoreEqHostBackup() error {
	dir, err := a.eqInstallDir()
	if err != nil {
		return err
	}
	if err := eqhost.RestoreBackup(dir); err != nil {
		return err
	}
	if a.ctx != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "eqhost.txt restored",
			Message: "Restart EverQuest for the restored eqhost.txt to apply.",
		})
	}
	return nil
}

// OpenEQDirectory opens the configured EverQuest install folder in the file manager.
func (a *App) OpenEQDirectory() error {
	dir, err := a.eqInstallDir()
	if err != nil {
		return err
	}
	return eqpath.OpenInFileManager(dir)
}

// PickEQDirectory opens a native folder picker and saves the selection.
func (a *App) PickEQDirectory() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app not ready")
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Select EverQuest install directory",
		CanCreateDirectories: false,
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // cancelled
	}
	if err := a.SetEQDirectory(path); err != nil {
		return "", err
	}
	return path, nil
}

// SetListenAddr sets the UDP proxy bind address (e.g. 127.0.0.1:6998).
// If the proxy is running, it is restarted on the new address.
func (a *App) SetListenAddr(addr string) error {
	if a.cfg == nil {
		return nil
	}
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("listen address required")
	}
	if !strings.Contains(addr, ":") {
		return fmt.Errorf("use host:port, e.g. 127.0.0.1:6998")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}
	if host == "" {
		host = "127.0.0.1"
		addr = net.JoinHostPort(host, port)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		// Keep default safety: only loopback unless we later add an opt-in
		return fmt.Errorf("listen host must be 127.0.0.1 or localhost (for safety)")
	}
	wasOn := a.cfg.Mode().WantsProxy()
	if wasOn {
		a.stopProxyRuntime(false)
	}
	if err := a.cfg.Update(func(c *sources.Config) { c.ListenAddr = addr }); err != nil {
		return err
	}
	if wasOn {
		return a.startProxy(false)
	}
	return nil
}

// SetListenPort sets UDP listen port on 127.0.0.1.
func (a *App) SetListenPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be 1–65535")
	}
	return a.SetListenAddr(fmt.Sprintf("127.0.0.1:%d", port))
}

func (a *App) StartProxy() error {
	return a.SetConnectionMode(string(sources.ConnectionLoginSSO))
}

func (a *App) startProxy(showEqhostDialog bool) error {
	if a.cfg == nil || a.local == nil {
		return nil
	}
	if a.proxy != nil {
		a.stopProxyRuntime(false)
	}
	cfg := a.cfg.Get()
	r := &router.Router{Local: a.local, SSO: a.sso, Log: a.log, BusyFn: a.busyLocal}
	a.proxy = &proxy.Server{
		Listen: cfg.ListenAddr, Upstream: cfg.UpstreamAddr, Router: r, Log: a.log,
	}
	if err := a.proxy.Start(a.ctx); err != nil {
		return err
	}
	if cfg.EQDirectory != "" {
		if err := eqhost.Enable(cfg.EQDirectory, cfg.ListenAddr); err != nil {
			a.log.Warn("eqhost", "err", err)
		} else {
			a.log.Info("eqhost updated; restart EQ for changes")
			if showEqhostDialog && a.ctx != nil {
				_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
					Type:    runtime.InfoDialog,
					Title:   "eqhost.txt updated",
					Message: "Restart EverQuest for the proxy host change to apply.",
				})
			}
		}
	}
	return nil
}

// StopProxy stops the proxy and sets connection mode to disabled.
func (a *App) StopProxy() {
	_ = a.SetConnectionMode(string(sources.ConnectionDisabled))
}

// stopProxyRuntime stops the UDP listener. If restoreEqhost is true, eqhost.txt is restored.
func (a *App) stopProxyRuntime(restoreEqhost bool) {
	if a.proxy != nil {
		a.proxy.Stop()
		a.proxy = nil
	}
	if !restoreEqhost || a.cfg == nil {
		return
	}
	cfg := a.cfg.Get()
	if cfg.EQDirectory != "" {
		_ = eqhost.Disable(cfg.EQDirectory)
	}
}

type UpdateInfo struct {
	UpdateAvailable bool   `json:"update_available"`
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	ReleaseURL      string `json:"release_url"`
	Error           string `json:"error,omitempty"`
}

func (a *App) CheckUpdate() (UpdateInfo, error) {
	repo := "alfred-identity/app"
	if a.cfg != nil && a.cfg.Get().GitHubRepo != "" {
		repo = a.cfg.Get().GitHubRepo
	}
	res, err := updatecheck.Check(a.ctx, repo, Version)
	if err != nil {
		return UpdateInfo{Current: Version, Error: err.Error()}, nil
	}
	return UpdateInfo{
		UpdateAvailable: res.UpdateAvailable,
		Current:         res.Current,
		Latest:          res.Latest,
		ReleaseURL:      res.ReleaseURL,
	}, nil
}

func (a *App) OpenReleaseURL(url string) {
	if url != "" && a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}

func (a *App) GetVersion() string {
	return Version
}
