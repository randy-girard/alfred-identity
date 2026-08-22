package proxy

import (
	"testing"

	"github.com/alfred-identity/app/internal/protocol"
)

func loginPacket() []byte {
	base := []byte{
		0x00, 0x03,
		0x04, 0x00, 0x15, 0x00, 0x00,
		0x20, 0x00, 0x09, 0x00, 0x01,
		0x02, 0x00,
		0x03, 0x00, 0x00, 0x00,
		0x00,
		0x02,
		0x00, 0x00, 0x00, 0x00,
	}
	return append(base, protocol.GoldenBytes()...)
}

func TestExtractUsernameHint(t *testing.T) {
	hint := extractUsernameHint(loginPacket())
	if hint != "user" {
		t.Fatalf("got %q", hint)
	}
	if extractUsernameHint([]byte{1, 2, 3}) != "" {
		t.Fatal("expected empty for short packet")
	}
	if extractUsernameHint([]byte{0x00, 0x01}) != "" {
		t.Fatal("expected empty for non-combined")
	}
	// Valid offset but cipher length not multiple of 8
	badLen := loginPacket()
	badLen = badLen[:len(badLen)-1]
	if extractUsernameHint(badLen) != "" {
		t.Fatal("expected empty for bad cipher len")
	}
	// Cipher with no NUL terminator in plaintext
	ct, err := protocol.EncryptDES([]byte("ABCDEFGH"))
	if err != nil {
		t.Fatal(err)
	}
	base := []byte{
		0x00, 0x03,
		0x04, 0x00, 0x15, 0x00, 0x00,
		0x20, 0x00, 0x09, 0x00, 0x01,
		0x02, 0x00,
		0x03, 0x00, 0x00, 0x00,
		0x00,
		0x02,
		0x00, 0x00, 0x00, 0x00,
	}
	noNul := append(base, ct...)
	if extractUsernameHint(noNul) != "" {
		t.Fatal("expected empty when plaintext has no NUL")
	}
}
