# WebSocket SSO API (v1)

Protocol version: **1** (hard-reject mismatch on `auth`).

Endpoint: `WS_PATH` (default `/ws/sso`). Production: terminate TLS at an external reverse proxy (`wss://`).

## Client → server

### `auth`
```json
{ "type": "auth", "token": "<raw api token>", "protocol_version": 1, "client_version": "gui/0.1.0" }
```

### `get_state`
```json
{ "type": "get_state" }
```
Re-sends `full_state` for the authenticated user (after Discord group/account changes).

### `login_auth`
```json
{ "type": "login_auth", "request_id": "uuid", "username": "alias-or-character" }
```
Daemon is authoritative: resolve alias → allowed non-disabled accounts → skip busy accounts → return first free.

### `heartbeat`
```json
{ "type": "heartbeat", "character_name": "Hero", "offline": false }
```
Server looks up character → EQ account; marks **account** online. Unknown character ignored. Only accounts the token may access are accepted.

### `pong`
```json
{ "type": "pong" }
```

## Server → client

### `full_state` (after auth / reconnect — no `delta` in v1)
```json
{
  "type": "full_state",
  "state": {
    "accounts": [
      { "id": 1, "disabled": false, "aliases": ["tank"], "characters": ["Hero"] }
    ],
    "online": [{ "account_id": 1, "character_name": "Hero" }]
  }
}
```
**Never** includes passwords, hashes, or DES blobs.

### `login_auth_response`
Success:
```json
{
  "type": "login_auth_response",
  "request_id": "uuid",
  "real_user": "equser",
  "encrypted_credentials": "<base64 DES-CBC blob>",
  "account_id": 1
}
```
Error:
```json
{ "type": "login_auth_response", "request_id": "uuid", "error": "not_found|all_busy|rate_limited|internal" }
```

DES wire: username`\0`password`\0`, zero-pad to 8, DES-CBC with key/IV eight zero bytes. Treat blob as a password — ephemeral, TLS only, never log or persist.

### `error` / `ping`
```json
{ "type": "error", "message": "..." }
{ "type": "ping" }
```

## Golden DES vector

Plain: `user\0pass\0` (+ zero pad) → cipher hex `575ab3e46810e874f75cb31595902052`
