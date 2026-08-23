# Alfred Identity

Native desktop GUI ([Wails v2](https://wails.io)) — local UDP login proxy and SSO client for **[alfred-identity-backend](https://github.com/randy-girard/alfred-identity-backend)**.

The app runs in the **menu bar** (macOS) or **system tray** (Windows/Linux). Closing the window hides it; the proxy and SSO connection keep running until you quit.

## Requirements

- Go 1.26+ (see `go.mod`)
- [Wails v2](https://wails.io) CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Node.js (frontend build)

## Project layout

```
main.go                 # Wails entrypoint
wails.json
frontend/               # React UI
internal/
  app/                  # Wails backend, tray, instance lock
  eqhost/               # eqhost.txt read/write/backup
  eqpath/               # EverQuest install path helpers
  localdata/            # Local accounts CSV store
  logbuf/ logwatch/     # In-app log buffer; EQ log tailing
  p99proxy/             # P99 Login Proxy install discovery/import
  protocol/ proxy/      # SOE login protocol; UDP relay
  router/               # Login packet routing (local / SSO / busy)
  sources/              # config.json + SSO source list
  sso/                  # WebSocket SSO client
  updatecheck/          # GitHub release checks + in-app install
scripts/                # dev.sh, build.sh, test.sh, version helpers
docs/                   # ws-api.md, protocol.md
build/                  # Wails platform packaging assets
```

Unit tests live next to each package under `internal/` (co-located `*_test.go` files).

## Develop

```bash
cd alfred-identity
./scripts/dev.sh
```

## Build

```bash
cd alfred-identity
./scripts/build.sh
# → build/bin/Alfred Identity.app (macOS) or equivalent
```

`scripts/dev.sh` and `scripts/build.sh` stamp `app.Version` at link time from git (`dev-<short>`, plus `+dirty` when needed). Release CI overrides with semver from the release tag.

## Release

1. Push your changes to `main`.
2. GitHub → **Actions** → **Release** → **Run workflow**.
3. Enter a semver tag (e.g. `v1.2.0`). The workflow **creates and pushes the tag** on the current commit if it does not exist yet, then builds platform zip assets and publishes a GitHub Release.

Pre-releases: use a suffix (e.g. `v1.2.0-beta.1`). Re-running the workflow for an **existing** tag rebuilds from that tag’s commit.

## Tests and coverage

```bash
make test          # go test ./... with race detector
make coverage      # writes coverage/index.html (+ source.html, func.txt)
```

Open `coverage/index.html` in a browser.

---

## Help — using the app

### Open, hide, and quit

| Platform | Where the app lives | Reopen window | Quit |
|----------|---------------------|---------------|------|
| **macOS** | Menu bar Alfred icon; Dock icon only while window is open | Click the menu bar icon → **Show Window**, or click Dock icon | Menu bar icon → **Exit**, or app menu **Quit** |
| **Windows / Linux** | System tray icon; taskbar while window is open | Tray → **Show Window** | Tray → **Exit** |

**Check for updates** is also on the menu bar / tray menu (macOS: click the Alfred icon → **Check for Updates**). When a newer release has a matching platform zip, you can **Install & restart** in-app (downloads, replaces the install, clears macOS quarantine on the new build, then relaunches). The first install from a browser download may still need **Right-click → Open** or `xattr -cr` once; later in-app updates handle quarantine automatically.

### First-time setup

1. **Start the backend** (guild daemon), e.g. from `alfred-identity-backend`: `docker compose up --build`
2. **Create an SSO token** in Discord (`/alfred-identity-sso create`) or with `go run ./cmd/seedtoken …` when Discord is disabled
3. **Add an SSO source** in Alfred Identity → **Connections** → paste the JSON from `/alfred-identity-sso get` → **Add from JSON** (or **Manage sources…** → **Add manually**)
4. Set **Connection mode** to **Login w/ SSO**
5. **EverQuest** tab → **Browse…** → pick your install folder → **Save path**
6. When the proxy starts, the app can rewrite `eqhost.txt` to point at the local listener — **restart EverQuest** after eqhost changes

### Connection modes

Set on **Connections** (or the status bar at the bottom):

| Mode | Proxy | SSO | Use when |
|------|-------|-----|----------|
| **Login w/ SSO** | On | On | Guild accounts via daemon; local accounts still work |
| **Login Only** | On | Off | Local CSV accounts only; no guild SSO |
| **Disabled** | Off | Off | Not proxying logins |

**Listen port** (default **6998**, `127.0.0.1` only) is on **Settings** → **UDP proxy**. Changing it while the proxy is running restarts the listener.

### SSO sources

- Stored in `config.json` under your OS user config dir (see [Config files](#config-files))
- **Connections** lists sources; click **Use this source** to activate one
- Only the **active** source is connected when mode is **Login w/ SSO**
- Host is `host:port` only — the app builds `ws://` or `wss://` + `/ws/sso` when connecting (`ws` for localhost/LAN, `wss` for public hosts)

### Accounts tab

| Sub-tab | Purpose |
|---------|---------|
| **SSO** | Read-only roster from the active daemon (aliases, tags, characters, who is logged in). Manage accounts in the [web admin](../alfred-identity-backend/README.md). |
| **Local** | Personal accounts on this machine (CSV). Checked **before** SSO when the typed login name matches. **Add account**, **Import CSV…**, **Export CSV…**, or import from a **P99 Login Proxy** install. |
| **Shared** | (When SSO connected) Outgoing shares you granted and accounts others shared with you |

**Login behavior:** type an account name, alias, tag, or character name at the EverQuest login screen. Local names win over SSO. Tags cycle shared guild accounts; aliases belong to one account.

**Share** (Local tab): with SSO connected, copy a local account to selected Discord users as a private SSO share without giving up your local copy.

### EverQuest tab

- **Install directory** — used for log watching (online character count in the status bar) and `eqhost.txt`
- **eqhost.txt** — view, edit, save; first save creates `eqhost.txt.bak`; **Restore backup** rolls back
- **Online (from logs)** — characters with fresh timestamped enter/welcome/activity in `eqlog_*.txt` (on startup, actively written logs are inspected so an already in-game session can resync SSO presence; stale logs do not block login)

### Logs tab

In-app log of proxy, SSO, eqhost, and backend activity. **Auto-scroll** and **Clear** are available while the tab is open.

### Settings tab

- **Appearance** — light / dark theme (saved in browser local storage on this machine)
- **UDP proxy** — listen port
- **Updates** — current version, **Check for updates**, **Install & restart** when a platform build is available, or open the GitHub release page

### Status bar

Shown at the bottom on every tab: **Mode**, **Proxy**, **SSO**, **EQ path**, **Online** count. Mode can be changed from the bar without opening **Connections**.

---

## Config files

Stored under the OS user config directory in **`alfred-identity/`** (migrated automatically from legacy `alfred-identity-gui` or `p99-identity-gui`):

| File | Purpose |
|------|---------|
| `config.json` | SSO sources, active source, connection mode, listen address, EQ directory, GitHub repo for updates |
| `accounts.csv` | Local account names/passwords/aliases |
| `characters.csv` | Local character → account mappings |

Example `config.json` fragment:

```json
{
  "active_source_id": "default",
  "sources": [
    {
      "id": "default",
      "name": "Local daemon",
      "host": "identity.example.com:443",
      "token": "paste-token-here"
    }
  ],
  "listen_addr": "127.0.0.1:6998",
  "connection_mode": "login_sso"
}
```

`connection_mode`: `login_sso`, `login_only`, or `disabled`.

---

## UI tabs (quick reference)

| Tab | Purpose |
|-----|---------|
| **Connections** | Connection mode, SSO source list, add/import sources |
| **Accounts** | SSO roster, local accounts, sharing |
| **EverQuest** | Install path, `eqhost.txt`, online characters from logs |
| **Logs** | Application log viewer |
| **Settings** | Theme, listen port, update check / install |

---

## Docs

- [docs/ws-api.md](docs/ws-api.md) — WebSocket contract (same as daemon)
- [docs/protocol.md](docs/protocol.md) — login packet / DES notes
- [alfred-identity-backend README](../alfred-identity-backend/README.md) — daemon, Compose, Discord bot, web admin
