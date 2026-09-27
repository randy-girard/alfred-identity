# Protocol notes (GUI)

DES-CBC, key/IV eight zero bytes, plaintext `username\0password\0` zero-padded to 8.

Golden: `user`/`pass` → `575ab3e46810e874f75cb31595902052`

Login splice follows **p99-login-proxy** `LoginPacket` layout (Ack + Login subpacket). See `internal/protocol/login_packet.go`.

The UDP proxy strips/restores SOE CRC after session negotiation and rewrites transport sequences like p99-login-proxy (`internal/protocol/session.go`, `internal/proxy/engine.go`).

When Connection mode is **Login w/ SSO**, the daemon splices vault credentials into the Combined login packet (`login_splice` / `login_splice_result`). The desktop then UDP-sends that packet (and the rest of the login session) to `login.eqemulator.net:5998` from the player machine so world/zone transfer sees the player's IP. The vault password is not stored in the GUI; it exists only in the spliced datagram on the way to the login server.

**Login Only** (no SSO) still uses the local UDP hop to the configured upstream, including local CSV rewrite.

Server lists arrive as SOE **Fragment** datagrams; the proxy reassembles them and forwards the payload to the client as Packet(s), matching p99-login-proxy passthrough behavior.
