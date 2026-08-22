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
}
