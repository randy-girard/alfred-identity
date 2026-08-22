# Protocol notes (GUI)

DES-CBC, key/IV eight zero bytes, plaintext `username\0password\0` zero-padded to 8.

Golden: `user`/`pass` → `575ab3e46810e874f75cb31595902052`

Login splice follows **p99-login-proxy** `LoginPacket` layout (Ack + Login subpacket). See `internal/protocol/login_packet.go`.

The UDP proxy strips/restores SOE CRC after session negotiation and rewrites transport sequences like p99-login-proxy (`internal/protocol/session.go`, `internal/proxy/engine.go`).
