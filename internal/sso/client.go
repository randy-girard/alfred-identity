package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const ProtocolVersion = 1

type AccountMeta struct {
	ID              int64    `json:"id"`
	Username        string   `json:"username"`
	Disabled        bool     `json:"disabled"`
	Elevated        bool     `json:"elevated"`
	RequiredRoleID  string   `json:"required_role_id"`
	RequiredRoleIDs []string `json:"required_role_ids"`
	RequiredUserID  int64    `json:"required_user_id"`
	GroupIDs        []int64  `json:"group_ids"`
	Restricted      bool     `json:"restricted"`
	OwnerUserID     int64    `json:"owner_user_id"`
	SharedUserIDs   []int64  `json:"shared_user_ids"`
	Aliases         []string `json:"aliases"`
	Tags            []string `json:"tags"`
	Characters      []string `json:"characters"`
}

type OnlineEntry struct {
	AccountID     int64  `json:"account_id"`
	CharacterName string `json:"character_name"`
}

type ShareLoginEntry struct {
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UserID          int64     `json:"user_id"`
	ActorName       string    `json:"actor_name"`
	ActorDiscordID  string    `json:"actor_discord_id"`
	AccountID       int64     `json:"account_id"`
	AccountUsername string    `json:"account_username"`
	Detail          string    `json:"detail"`
	ActorIsOwner    bool      `json:"actor_is_owner"`
}

type ShareOnlineEntry struct {
	AccountID       int64     `json:"account_id"`
	AccountUsername string    `json:"account_username"`
	CharacterName   string    `json:"character_name"`
	UserID          int64     `json:"user_id"`
	UserDisplayName string    `json:"user_display_name"`
	UserDiscordID   string    `json:"user_discord_id"`
	ActorIsOwner    bool      `json:"actor_is_owner"`
	LastSeen        time.Time `json:"last_seen"`
}

type ShareActivity struct {
	Logins []ShareLoginEntry  `json:"logins"`
	Online []ShareOnlineEntry `json:"online"`
}

type FullState struct {
	Accounts []AccountMeta `json:"accounts"`
	Online   []OnlineEntry `json:"online"`
}

type DirectoryUser struct {
	ID          int64  `json:"id"`
	DiscordID   string `json:"discord_id"`
	DisplayName string `json:"display_name"`
}

type GroupUser struct {
	ID          int64  `json:"id"`
	DiscordID   string `json:"discord_id"`
	DisplayName string `json:"display_name"`
}

type GroupDetail struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	WebRole     string      `json:"web_role"`
	Users       []GroupUser `json:"users"`
	UserIDs     []int64     `json:"user_ids"`
	RoleIDs     []string    `json:"role_ids"`
	AccountIDs  []int64     `json:"account_ids"`
}

type AdminUser struct {
	ID             int64    `json:"id"`
	DiscordID      string   `json:"discord_id"`
	DisplayName    string   `json:"display_name"`
	RoleIDs        []string `json:"role_ids"`
	AccessRevoked  bool     `json:"access_revoked"`
	HasActiveToken bool     `json:"has_active_token"`
}

type DiscordRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AdminState struct {
	Users []AdminUser   `json:"users"`
	Roles []DiscordRole `json:"roles"`
}

type LoginAuthResult struct {
	RealUser  string
	CipherB64 string
	AccountID int64
	Error     string
}

type AdminResult struct {
	OK        bool
	AccountID int64
	Error     string
}

type Client struct {
	mu          sync.Mutex
	writeMu     sync.Mutex
	conn        *websocket.Conn
	state       FullState
	admin       AdminState
	directory   []DirectoryUser
	groups      []GroupDetail
	roles       []DiscordRole
	shareAct    ShareActivity
	userID      int64
	discordID   string
	displayName string
	isAdmin     bool
	cancel      context.CancelFunc
	waiters     map[string]chan LoginAuthResult
	adminWait   map[string]chan AdminResult
	stateWait   []chan struct{}
	connected   bool
	// keepaliveEvery overrides the default ping interval when > 0 (tests).
	keepaliveEvery time.Duration
}

func NewClient() *Client {
	return &Client{
		waiters:   make(map[string]chan LoginAuthResult),
		adminWait: make(map[string]chan AdminResult),
	}
}

// TestClientState seeds getter state for unit tests without a live websocket.
type TestClientState struct {
	Connected     bool
	Admin         bool
	UserID        int64
	Accounts      []AccountMeta
	ShareActivity ShareActivity
	Directory     []DirectoryUser
	Groups        []GroupDetail
	Roles         []DiscordRole
	AdminUsers    []AdminUser
	AdminRoles    []DiscordRole
}

// SetStateForTest replaces cached SSO client state (tests only).
func (c *Client) SetStateForTest(st TestClientState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = st.Connected
	c.isAdmin = st.Admin
	c.userID = st.UserID
	c.state = FullState{
		Accounts: append([]AccountMeta(nil), st.Accounts...),
		Online:   []OnlineEntry{},
	}
	c.shareAct = st.ShareActivity
	if c.shareAct.Logins == nil {
		c.shareAct.Logins = []ShareLoginEntry{}
	}
	if c.shareAct.Online == nil {
		c.shareAct.Online = []ShareOnlineEntry{}
	}
	if st.Directory != nil {
		c.directory = append([]DirectoryUser(nil), st.Directory...)
	} else {
		c.directory = []DirectoryUser{}
	}
	if st.Groups != nil {
		c.groups = append([]GroupDetail(nil), st.Groups...)
	} else {
		c.groups = []GroupDetail{}
	}
	if st.Roles != nil {
		c.roles = append([]DiscordRole(nil), st.Roles...)
	} else {
		c.roles = []DiscordRole{}
	}
	if st.Admin {
		c.admin = AdminState{
			Users: append([]AdminUser(nil), st.AdminUsers...),
			Roles: append([]DiscordRole(nil), st.AdminRoles...),
		}
	} else {
		c.admin = AdminState{}
	}
}

func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *Client) IsAdmin() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isAdmin
}

func (c *Client) Admin() AdminState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.admin
}

func (c *Client) Directory() []DirectoryUser {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]DirectoryUser, len(c.directory))
	copy(out, c.directory)
	return out
}

func (c *Client) Groups() []GroupDetail {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]GroupDetail, len(c.groups))
	copy(out, c.groups)
	return out
}

func (c *Client) Roles() []DiscordRole {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]DiscordRole, len(c.roles))
	copy(out, c.roles)
	return out
}

func (c *Client) ShareActivity() ShareActivity {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := ShareActivity{
		Logins: append([]ShareLoginEntry(nil), c.shareAct.Logins...),
		Online: append([]ShareOnlineEntry(nil), c.shareAct.Online...),
	}
	if out.Logins == nil {
		out.Logins = []ShareLoginEntry{}
	}
	if out.Online == nil {
		out.Online = []ShareOnlineEntry{}
	}
	return out
}

func (c *Client) UserID() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.userID
}

func (c *Client) State() FullState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Client) NameInMetadata(name string) bool {
	st := c.State()
	for _, a := range st.Accounts {
		for _, al := range a.Aliases {
			if strings.EqualFold(al, name) {
				return true
			}
		}
		for _, tag := range a.Tags {
			if strings.EqualFold(tag, name) {
				return true
			}
		}
		for _, ch := range a.Characters {
			if strings.EqualFold(ch, name) {
				return true
			}
		}
		if strings.EqualFold(a.Username, name) {
			return true
		}
	}
	return false
}

func (c *Client) OnlineAccountIDs() map[int64]bool {
	st := c.State()
	m := make(map[int64]bool)
	for _, o := range st.Online {
		m[o.AccountID] = true
	}
	return m
}

// normalizeDialURL remaps http(s)→ws(s) and rejects remote plain ws://.
func normalizeDialURL(wsURL string) (*url.URL, error) {
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	host := u.Hostname()
	if u.Scheme == "ws" && host != "127.0.0.1" && host != "localhost" && !strings.EqualFold(host, "localhost") {
		if host != "::1" {
			return nil, fmt.Errorf("remote sources require wss://")
		}
	}
	return u, nil
}

func (c *Client) Connect(parent context.Context, wsURL, token, clientVersion string) error {
	c.Disconnect()
	u, err := normalizeDialURL(wsURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(parent)
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	})
	if err != nil {
		cancel()
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.cancel = cancel
	c.connected = true
	c.isAdmin = false
	c.state = FullState{}
	c.admin = AdminState{}
	c.directory = nil
	c.groups = nil
	c.roles = nil
	c.userID = 0
	c.discordID = ""
	c.displayName = ""
	c.waiters = make(map[string]chan LoginAuthResult)
	c.adminWait = make(map[string]chan AdminResult)
	c.mu.Unlock()

	if strings.TrimSpace(clientVersion) == "" {
		clientVersion = "gui/unknown"
	}
	auth, _ := json.Marshal(map[string]any{
		"type": "auth", "token": token, "protocol_version": ProtocolVersion, "client_version": clientVersion,
	})
	c.writeMu.Lock()
	err = conn.Write(ctx, websocket.MessageText, auth)
	c.writeMu.Unlock()
	if err != nil {
		c.Disconnect()
		return err
	}
	go c.readLoop(ctx)
	go c.keepaliveLoop(ctx)
	return nil
}

// defaultKeepaliveInterval is how often the client sends a ping while connected.
// Character heartbeats only fire when someone is online; without this, idle
// sockets are often closed by reverse proxies around ~60s of silence.
const defaultKeepaliveInterval = 25 * time.Second

// SetKeepaliveIntervalForTest sets this client's ping interval (call before Connect).
func (c *Client) SetKeepaliveIntervalForTest(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keepaliveEvery = d
}

func (c *Client) keepaliveLoop(ctx context.Context) {
	c.mu.Lock()
	interval := c.keepaliveEvery
	c.mu.Unlock()
	if interval <= 0 {
		interval = defaultKeepaliveInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.mu.Lock()
			conn := c.conn
			ok := c.connected && conn != nil
			c.mu.Unlock()
			if !ok {
				return
			}
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.writeJSON(pingCtx, conn, map[string]any{"type": "ping"})
			cancel()
			if err != nil {
				// readLoop will clear connected when the socket dies
				return
			}
		}
	}
}

func (c *Client) readLoop(ctx context.Context) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			c.mu.Lock()
			c.connected = false
			c.isAdmin = false
			c.mu.Unlock()
			return
		}
		var tip struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &tip) != nil {
			continue
		}
		switch tip.Type {
		case "full_state":
			var msg struct {
				State         FullState       `json:"state"`
				IsAdmin       bool            `json:"is_admin"`
				Admin         AdminState      `json:"admin"`
				Directory     []DirectoryUser `json:"directory"`
				Groups        []GroupDetail   `json:"groups"`
				Roles         []DiscordRole   `json:"roles"`
				ShareActivity ShareActivity   `json:"share_activity"`
				UserID        int64           `json:"user_id"`
				DiscordID     string          `json:"discord_id"`
				DisplayName   string          `json:"display_name"`
			}
			if json.Unmarshal(data, &msg) == nil {
				c.mu.Lock()
				c.state = msg.State
				c.isAdmin = msg.IsAdmin
				c.userID = msg.UserID
				c.discordID = msg.DiscordID
				c.displayName = msg.DisplayName
				if msg.Directory != nil {
					c.directory = msg.Directory
				} else {
					c.directory = []DirectoryUser{}
				}
				if msg.Groups != nil {
					c.groups = msg.Groups
				} else {
					c.groups = []GroupDetail{}
				}
				if msg.Roles != nil {
					c.roles = msg.Roles
				} else {
					c.roles = []DiscordRole{}
				}
				c.shareAct = msg.ShareActivity
				if c.shareAct.Logins == nil {
					c.shareAct.Logins = []ShareLoginEntry{}
				}
				if c.shareAct.Online == nil {
					c.shareAct.Online = []ShareOnlineEntry{}
				}
				if msg.IsAdmin {
					c.admin = msg.Admin
					if c.admin.Users == nil {
						c.admin.Users = []AdminUser{}
					}
					if c.admin.Roles == nil {
						c.admin.Roles = []DiscordRole{}
					}
				} else {
					c.admin = AdminState{}
				}
				waiters := c.stateWait
				c.stateWait = nil
				c.mu.Unlock()
				for _, ch := range waiters {
					select {
					case ch <- struct{}{}:
					default:
					}
				}
			}
		case "login_auth_response":
			var resp struct {
				RequestID            string `json:"request_id"`
				RealUser             string `json:"real_user"`
				EncryptedCredentials string `json:"encrypted_credentials"`
				AccountID            int64  `json:"account_id"`
				Error                string `json:"error"`
			}
			if json.Unmarshal(data, &resp) != nil {
				continue
			}
			c.mu.Lock()
			ch := c.waiters[resp.RequestID]
			c.mu.Unlock()
			if ch != nil {
				select {
				case ch <- LoginAuthResult{
					RealUser: resp.RealUser, CipherB64: resp.EncryptedCredentials,
					AccountID: resp.AccountID, Error: resp.Error,
				}:
				default:
				}
			}
		case "admin_result", "share_result":
			var resp struct {
				RequestID string `json:"request_id"`
				OK        bool   `json:"ok"`
				AccountID int64  `json:"account_id"`
				Error     string `json:"error"`
			}
			if json.Unmarshal(data, &resp) != nil {
				continue
			}
			c.mu.Lock()
			ch := c.adminWait[resp.RequestID]
			c.mu.Unlock()
			if ch != nil {
				select {
				case ch <- AdminResult{OK: resp.OK, AccountID: resp.AccountID, Error: resp.Error}:
				default:
				}
			}
		case "ping":
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()
			if conn != nil {
				_ = c.writeJSON(ctx, conn, map[string]any{"type": "pong"})
			}
		case "pong":
			// keepalive ack from server
		}
	}
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.conn != nil {
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
		c.conn = nil
	}
	c.connected = false
	c.isAdmin = false
	c.state = FullState{}
	c.admin = AdminState{}
	c.shareAct = ShareActivity{}
	c.directory = nil
	c.groups = nil
	c.roles = nil
	c.mu.Unlock()
}

func (c *Client) writeJSON(ctx context.Context, conn *websocket.Conn, v any) error {
	msg, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return conn.Write(ctx, websocket.MessageText, msg)
}

func (c *Client) LoginAuth(ctx context.Context, requestID, username string) (LoginAuthResult, error) {
	c.mu.Lock()
	conn := c.conn
	if conn == nil {
		c.mu.Unlock()
		return LoginAuthResult{}, fmt.Errorf("not connected")
	}
	ch := make(chan LoginAuthResult, 1)
	c.waiters[requestID] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.waiters, requestID)
		c.mu.Unlock()
	}()

	if err := c.writeJSON(ctx, conn, map[string]any{
		"type": "login_auth", "request_id": requestID, "username": username,
	}); err != nil {
		return LoginAuthResult{}, err
	}
	select {
	case res := <-ch:
		return res, nil
	case <-time.After(3 * time.Second):
		return LoginAuthResult{}, fmt.Errorf("login_auth timeout")
	case <-ctx.Done():
		return LoginAuthResult{}, ctx.Err()
	}
}

func (c *Client) LoginAuthWithRetry(ctx context.Context, requestID, username string) (LoginAuthResult, error) {
	res, err := c.LoginAuth(ctx, requestID, username)
	if err == nil && res.Error == "" {
		return res, nil
	}
	if err == nil && res.Error != "" {
		return res, nil
	}
	return c.LoginAuth(ctx, requestID+"-r", username)
}

func (c *Client) adminRPC(ctx context.Context, payload map[string]any) (AdminResult, error) {
	c.mu.Lock()
	isAdmin := c.isAdmin
	c.mu.Unlock()
	if !isAdmin {
		return AdminResult{}, fmt.Errorf("admin access required")
	}
	return c.requestRPC(ctx, payload)
}

func (c *Client) requestRPC(ctx context.Context, payload map[string]any) (AdminResult, error) {
	reqID, _ := payload["request_id"].(string)
	if reqID == "" {
		reqID = uuid.NewString()
		payload["request_id"] = reqID
	}
	c.mu.Lock()
	conn := c.conn
	if conn == nil {
		c.mu.Unlock()
		return AdminResult{}, fmt.Errorf("not connected to SSO")
	}
	ch := make(chan AdminResult, 1)
	c.adminWait[reqID] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.adminWait, reqID)
		c.mu.Unlock()
	}()

	msg, err := json.Marshal(payload)
	if err != nil {
		return AdminResult{}, err
	}
	c.writeMu.Lock()
	err = conn.Write(ctx, websocket.MessageText, msg)
	c.writeMu.Unlock()
	if err != nil {
		return AdminResult{}, err
	}
	select {
	case res := <-ch:
		if !res.OK {
			if res.Error == "" {
				res.Error = "failed"
			}
			return res, fmt.Errorf("%s", res.Error)
		}
		return res, nil
	case <-time.After(5 * time.Second):
		return AdminResult{}, fmt.Errorf("request timeout")
	case <-ctx.Done():
		return AdminResult{}, ctx.Err()
	}
}

func (c *Client) AdminAddAccount(ctx context.Context, username, password, requiredRoleID string) (AdminResult, error) {
	msg := map[string]any{
		"type": "admin_add_account", "request_id": uuid.NewString(),
		"username": username, "password": password,
	}
	if requiredRoleID != "" {
		msg["required_role_id"] = requiredRoleID
	}
	return c.adminRPC(ctx, msg)
}

func (c *Client) AdminUpdateAccount(ctx context.Context, accountID int64, password *string, disabled *bool, requiredRoleID *string) (AdminResult, error) {
	msg := map[string]any{
		"type": "admin_update_account", "request_id": uuid.NewString(),
		"account_id": accountID,
	}
	if password != nil {
		msg["password"] = *password
	}
	if disabled != nil {
		msg["disabled"] = *disabled
	}
	if requiredRoleID != nil {
		msg["required_role_id"] = *requiredRoleID
	}
	return c.adminRPC(ctx, msg)
}

func (c *Client) AdminAddAlias(ctx context.Context, alias string, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_add_alias", "request_id": uuid.NewString(),
		"alias": alias, "account_id": accountID,
	})
}

func (c *Client) AdminRemoveAlias(ctx context.Context, alias string, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_remove_alias", "request_id": uuid.NewString(),
		"alias": alias, "account_id": accountID,
	})
}

func (c *Client) AdminAddTag(ctx context.Context, tag string, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_add_tag", "request_id": uuid.NewString(),
		"tag": tag, "account_id": accountID,
	})
}

func (c *Client) AdminRemoveTag(ctx context.Context, tag string, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_remove_tag", "request_id": uuid.NewString(),
		"tag": tag, "account_id": accountID,
	})
}

func (c *Client) AdminAddCharacter(ctx context.Context, name string, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_add_character", "request_id": uuid.NewString(),
		"name": name, "account_id": accountID,
	})
}

func (c *Client) AdminRemoveCharacter(ctx context.Context, name string, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_remove_character", "request_id": uuid.NewString(),
		"name": name, "account_id": accountID,
	})
}

func (c *Client) AdminRemoveAccount(ctx context.Context, accountID int64) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_remove_account", "request_id": uuid.NewString(),
		"account_id": accountID,
	})
}

func (c *Client) AdminSetUserAccess(ctx context.Context, userID int64, revoked bool) (AdminResult, error) {
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_set_user_access", "request_id": uuid.NewString(),
		"user_id": userID, "revoked": revoked,
	})
}

func (c *Client) AdminSetUserRoles(ctx context.Context, userID int64, roleIDs []string) (AdminResult, error) {
	if roleIDs == nil {
		roleIDs = []string{}
	}
	return c.adminRPC(ctx, map[string]any{
		"type": "admin_set_user_roles", "request_id": uuid.NewString(),
		"user_id": userID, "role_ids": roleIDs,
	})
}

func (c *Client) RefreshState(ctx context.Context) error {
	c.mu.Lock()
	conn := c.conn
	if conn == nil {
		c.mu.Unlock()
		return nil
	}
	ch := make(chan struct{}, 1)
	c.stateWait = append(c.stateWait, ch)
	c.mu.Unlock()

	msg, _ := json.Marshal(map[string]any{"type": "get_state"})
	c.writeMu.Lock()
	err := conn.Write(ctx, websocket.MessageText, msg)
	c.writeMu.Unlock()
	if err != nil {
		return err
	}
	select {
	case <-ch:
		return nil
	case <-time.After(2 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) Heartbeat(ctx context.Context, character string, offline bool) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil
	}
	msg, _ := json.Marshal(map[string]any{
		"type": "heartbeat", "character_name": character, "offline": offline,
	})
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return conn.Write(ctx, websocket.MessageText, msg)
}

func (c *Client) ShareAccount(ctx context.Context, username, password string, aliases []string, userIDs []int64, roleIDs []string, groupIDs []int64) (AdminResult, error) {
	if userIDs == nil {
		userIDs = []int64{}
	}
	if roleIDs == nil {
		roleIDs = []string{}
	}
	if groupIDs == nil {
		groupIDs = []int64{}
	}
	if aliases == nil {
		aliases = []string{}
	}
	return c.requestRPC(ctx, map[string]any{
		"type": "share_account", "request_id": uuid.NewString(),
		"username": username, "password": password, "aliases": aliases,
		"user_ids": userIDs, "role_ids": roleIDs, "group_ids": groupIDs,
	})
}

func (c *Client) UnshareAccount(ctx context.Context, username string) (AdminResult, error) {
	return c.requestRPC(ctx, map[string]any{
		"type": "unshare_account", "request_id": uuid.NewString(),
		"username": username,
	})
}
