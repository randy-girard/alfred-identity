package protocol

import "testing"

func TestSOECRC32Vectors(t *testing.T) {
	got := SOECRC32([]byte("123456789"), 0)
	if got != 0x22896B0A {
		t.Fatalf("got %08x", got)
	}
	got = SOECRC32([]byte("123456789"), 0x12345678)
	if got != 0xAAD05244 {
		t.Fatalf("got %08x", got)
	}
}

func TestAppendStripCRC(t *testing.T) {
	pkt := []byte{0x00, 0x06}
	with := AppendCRC(pkt, 0x12345678, 2)
	if len(with) != len(pkt)+2 {
		t.Fatalf("len=%d", len(with))
	}
	stripped := StripCRC(with, 2)
	if string(stripped) != string(pkt) {
		t.Fatalf("strip mismatch")
	}
}

func TestParseLoginPacketAndSplice(t *testing.T) {
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
	base = append(base, GoldenBytes()...)
	lp, ok := ParseLoginPacket(base)
	if !ok {
		t.Fatal("parse failed")
	}
	if lp.Username != "user" || lp.Password != "pass" {
		t.Fatalf("user=%q pass=%q", lp.Username, lp.Password)
	}
	out, err := lp.RewriteCredentials("user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if out[ackEnd] != base[ackEnd] {
		t.Fatalf("sub len changed unexpectedly: %d vs %d", out[ackEnd], base[ackEnd])
	}
}

func TestParseLoginPacketRejectsBadPrefix(t *testing.T) {
	if _, ok := ParseLoginPacket([]byte{0x00, 0x03, 0x05, 0x00, 0x15}); ok {
		t.Fatal("expected reject")
	}
	if _, ok := ParseLoginPacket([]byte{0x00, 0x03, 0x04, 0x00, 0x15, 0x00, 0x00, 0x05}); ok {
		t.Fatal("truncated login")
	}
}

func TestSpliceEncryptedCredentialsTooLarge(t *testing.T) {
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
	base = append(base, GoldenBytes()...)
	lp, ok := ParseLoginPacket(base)
	if !ok {
		t.Fatal("parse")
	}
	big := make([]byte, 300)
	if _, err := lp.SpliceEncryptedCredentials(big); err == nil {
		t.Fatal("expected too large")
	}
}
