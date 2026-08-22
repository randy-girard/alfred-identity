package protocol

import "testing"

func TestAppendCRCTwoAndFourByte(t *testing.T) {
	pkt := []byte{0x00, OpPacket, 0x00, 0x01, 0x01, 0x02}
	key := uint32(0x12345678)

	out2 := AppendCRC(pkt, key, 2)
	if len(out2) != len(pkt)+2 {
		t.Fatalf("2-byte crc len=%d", len(out2))
	}
	if got := StripCRC(out2, 2); string(got) != string(pkt) {
		t.Fatal("2-byte strip mismatch")
	}

	out4 := AppendCRC(pkt, key, 4)
	if len(out4) != len(pkt)+4 {
		t.Fatalf("4-byte crc len=%d", len(out4))
	}
	if got := StripCRC(out4, 4); string(got) != string(pkt) {
		t.Fatal("4-byte strip mismatch")
	}

	unchanged := AppendCRC(pkt, key, 0)
	if string(unchanged) != string(pkt) {
		t.Fatal("crcBytes=0 should copy only")
	}
}

func TestStripCRCEdges(t *testing.T) {
	pkt := []byte{0x00, OpPacket, 0x00, 0x01}
	if got := StripCRC(pkt, 0); string(got) != string(pkt) {
		t.Fatal("crcBytes=0 should return original")
	}
	if got := StripCRC([]byte{1}, 2); len(got) != 1 {
		t.Fatal("short packet should not strip")
	}
	withCRC := AppendCRC(pkt, 0x11111111, 2)
	if got := StripCRC(withCRC, 2); string(got) != string(pkt) {
		t.Fatal("strip should recover payload")
	}
}

func TestSOECRC32Deterministic(t *testing.T) {
	pkt := []byte{0x00, OpPacket, 0x00, 0x01}
	a := SOECRC32(pkt, 0x11111111)
	b := SOECRC32(pkt, 0x11111111)
	if a != b {
		t.Fatalf("crc=%x vs %x", a, b)
	}
	if SOECRC32(pkt, 0x22222222) == a {
		t.Fatal("expected different key → different crc")
	}
}
