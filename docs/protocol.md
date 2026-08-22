# Protocol notes (GUI)

DES-CBC, key/IV eight zero bytes, plaintext `username\0password\0` zero-padded to 8.

Golden: `user`/`pass` → `575ab3e46810e874f75cb31595902052`

Combined Login splice updates the Packet sublen byte and replaces ciphertext. See `internal/protocol`.

SOE sequence rewrite / server-list filter are optional follow-ons once the login path is solid.
