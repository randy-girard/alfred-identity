# alfred-identity-app

Native desktop app (Wails v2) — local UDP login proxy and SSO client for the alfred-identity-web daemon.

## Requirements

- Go (see `go.mod`)
- [Wails v2](https://wails.io) CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- Node.js (for the frontend)

## Develop

```bash
cd alfred-identity-app
wails dev
```

## Build

```bash
cd alfred-identity-app
wails build
# → build/bin/Alfred Identity.app (macOS) or equivalent
```

Or:

```bash
./scripts/build.sh
```

## Tests and coverage

```bash
make test          # unit tests under ./internal (race detector)
make coverage      # writes coverage/index.html (+ source.html, func.txt)
```

Open `coverage/index.html` in a browser. Or from the repo root: `make test` / `make coverage`.

## Window / tray

The app lives in the **menu bar** (macOS: **AI** status item) or **system tray** (Windows/Linux). Closing the window hides it; proxy and SSO keep running.

- **macOS:** Dock icon only while the window is open; menu-bar **AI** item stays available
- **Windows/Linux:** Taskbar entry while the window is open; tray icon always available
- Reopen: status/tray item → **Show Window** (or click the Dock icon / reopen the app)
- Quit: status/tray item → **Exit** (macOS app menu Quit also works)

## First-time setup (with local daemon)

1. Start the daemon (from `../alfred-identity-web`): `docker compose up --build`
2. Create an SSO token in Discord (`/alfred-identity-sso create`) or with `go run ./cmd/seedtoken …` when Discord is disabled
3. Open **Alfred Identity** → **Connections** → paste `http://127.0.0.1:8181` (or `…/sso-source.json`) → **Add from URL**, paste your token, then set mode to **Login w/ SSO**
4. **EverQuest**: pick the install directory (log presence + `eqhost.txt`)
5. Point EQ login at the proxy (or let the app rewrite `eqhost.txt`), then restart EverQuest

## Config (SSO sources)

Stored under the OS user config dir: `…/alfred-identity-gui/config.json`.

Add sources in the UI (**Connections** → **Add from URL** using the guild’s `{origin}/sso-source.json`), or edit `sources` in the config file. Use **host** only (`host:port`); the app builds `ws(s)://{host}/ws/sso` when connecting (plain `ws` for localhost/LAN, `wss` for public hosts):

```json
{
  "active_source_id": "default",
  "sources": [
    {
      "id": "default",
      "name": "Local daemon",
      "host": "127.0.0.1:8181",
      "token": "paste-token-here"
    }
  ],
  "listen_addr": "127.0.0.1:6998",
  "connection_mode": "disabled"
}
```

`connection_mode`: `login_sso` (proxy + SSO), `login_only` (proxy without SSO), or `disabled`.

Only the **active** source is connected. Use **Activate** on the Connections tab to switch.

## Tabs

| Tab | Purpose |
|-----|---------|
| Connections | Listen port, enable/disable proxy, SSO source list (activate) |
| Accounts | Sub-tabs: **SSO** roster from the active daemon; **Local** personal accounts and aliases |
| EverQuest | Install path (Browse…), online characters from logs |

A status bar at the bottom of every tab shows proxy, SSO, EQ path, and online count (updates automatically).

Local personal accounts CSV paths are also under that config directory by default.

## Docs

- [docs/ws-api.md](docs/ws-api.md) — WebSocket contract (same as daemon)
- [docs/protocol.md](docs/protocol.md) — login packet / DES notes
- [../alfred-identity-web/README.md](../alfred-identity-web/README.md) — daemon Compose + Discord
