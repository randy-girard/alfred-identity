# Protocol notes (GUI)

DES-CBC, key/IV eight zero bytes, plaintext `username\0password\0` zero-padded to 8.

Golden: `user`/`pass` → `575ab3e46810e874f75cb31595902052`

Login splice follows **p99-login-proxy** `LoginPacket` layout (Ack + Login subpacket). See `internal/protocol/login_packet.go`.

The UDP proxy strips/restores SOE CRC after session negotiation and rewrites transport sequences like p99-login-proxy (`internal/protocol/session.go`, `internal/proxy/engine.go`).

When Connection mode is **Login w/ SSO**, the desktop does **not** UDP-talk to the EQ login server. It tunnels SOE datagrams over the authenticated WebSocket (`login_relay_up` / `login_relay_down`). The daemon splices vault credentials and is the process that sends UDP to `EQ_LOGIN_UPSTREAM` (default `login.eqemulator.net:5998`). After login, the client still connects to world/zone servers directly.

**Login Only** (no SSO) still uses the local UDP hop to the configured upstream, including local CSV rewrite.

Server lists arrive as SOE **Fragment** datagrams; the proxy reassembles them and forwards the payload to the client as Packet(s), matching p99-login-proxy passthrough behavior.
