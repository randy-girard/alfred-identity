package protocol

import "testing"

func TestDESGolden(t *testing.T) {
	ct, err := EncryptCredentials("user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if hexEncode(ct) != GoldenUserPassCipher {
		t.Fatalf("got %s want %s", hexEncode(ct), GoldenUserPassCipher)
	}
}

func TestSpliceRoundTrip(t *testing.T) {
	// Minimal Combined: Ack + Packet Login with golden cipher
	// 00 03 | 04 | 00 15 00 00 | 20 | 00 09 00 01 | 02 00 | 03 00 00 00 | 00 | 02 | 00 00 00 00 | <16 DES>
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
	out, err := RewriteLoginPacket(base, "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	start, _, ok := FindLoginCipherOffset(out)
	if !ok {
		t.Fatal("not found after splice")
	}
	if hexEncode(out[start:start+16]) != GoldenUserPassCipher {
		t.Fatalf("cipher mismatch after splice")
	}
}

func TestDecryptDESAndSpliceCipherBlob(t *testing.T) {
	ct, err := EncryptCredentials("user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := DecryptDES(ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt[:10]) != "user\x00pass\x00" {
		t.Fatalf("%q", pt)
	}
	if _, err := DecryptDES([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected bad length")
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
	base = append(base, GoldenBytes()...)
	out, err := SpliceCipherBlob(base, ct)
	if err != nil {
		t.Fatal(err)
	}
	start, _, ok := FindLoginCipherOffset(out)
	if !ok {
		t.Fatal("cipher not found")
	}
	if hexEncode(out[start:start+16]) != GoldenUserPassCipher {
		t.Fatal("cipher mismatch")
	}
	if _, err := SpliceCipherBlob([]byte{1, 2, 3}, ct); err == nil {
		t.Fatal("expected splice failure")
	}
}

func TestFindLoginCipherOffsetEdges(t *testing.T) {
	if _, _, ok := FindLoginCipherOffset([]byte{1, 2, 3}); ok {
		t.Fatal("too short")
	}
	if _, _, ok := FindLoginCipherOffset([]byte{0x00, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}); ok {
		t.Fatal("wrong opcode")
	}
	// Combined + truncated 0xFF length
	bad := []byte{0x00, 0x03, 0xFF}
	if _, _, ok := FindLoginCipherOffset(bad); ok {
		t.Fatal("truncated ff")
	}
	// Combined Ack only (no login packet)
	ackOnly := []byte{0x00, 0x03, 0x04, 0x00, 0x15, 0x00, 0x00}
	if _, _, ok := FindLoginCipherOffset(ackOnly); ok {
		t.Fatal("ack only")
	}
	good := []byte{
		0x00, 0x03,
		0x04, 0x00, 0x15, 0x00, 0x00,
		0x20, 0x00, 0x09, 0x00, 0x01,
		0x02, 0x00,
		0x03, 0x00, 0x00, 0x00,
		0x00,
		0x02,
		0x00, 0x00, 0x00, 0x00,
	}
	good = append(good, GoldenBytes()...)
	start, subIdx, ok := FindLoginCipherOffset(good)
	if !ok || start <= 0 || subIdx != 7 {
		t.Fatalf("start=%d subIdx=%d ok=%v", start, subIdx, ok)
	}
}

func hexEncode(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = h[v>>4]
		out[i*2+1] = h[v&0xf]
	}
	return string(out)
}
