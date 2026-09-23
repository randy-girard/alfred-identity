# WebSocket SSO API (v1)

Protocol version: **1** (hard-reject mismatch on `auth`).

Endpoint: `WS_PATH` (default `/ws/sso`). Production: terminate TLS at an external reverse proxy (`wss://`).

## Client → server

### `auth`
```json
{ "type": "auth", "token": "<raw api token>", "protocol_version": 1, "client_version": "gui/dev-abc1234" }
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
Daemon authorizes the login (ACL + busy). Success does **not** include credentials.

### `login_relay_up`
```json
{ "type": "login_relay_up", "payload": "<base64 SOE datagram>", "splice": true }
```
GUI tunnels login-server UDP through the daemon. `splice` is true only for the Combined login packet.

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
  "account_id": 1,
  "relay": true
}
```
Error:
```json
{ "type": "login_auth_response", "request_id": "uuid", "error": "not_found|all_busy|rate_limited|internal" }
```

### `login_relay_down`
```json
{ "type": "login_relay_down", "payload": "<base64 SOE datagram from EQ login server>" }
```

### `login_relay_error`
```json
{ "type": "login_relay_error", "message": "bad_payload|upstream_unavailable|internal|…" }
```

### `error` / `ping`
```json
{ "type": "error", "message": "..." }
{ "type": "ping" }
```
