package protocol

import "testing"

func TestParseCombinedTruncated(t *testing.T) {
	if _, ok := ParseCombined([]byte{0x00}, 0, -1); ok {
		t.Fatal("too short for opcode")
	}
	full := []byte{0x00, OpCombined, 0x04, 0x00, OpAck, 0x00, 0x01}
	if _, ok := ParseCombined(full, 0, 1); ok {
		t.Fatal("length bound too short")
	}
}

func TestParseCombinedAckAndPacket(t *testing.T) {
	// Combined: op(2) + ack sublen(1)+ack(4) + packet sublen(1)+packet(4+payload)
	pkt := []byte{
		0x00, OpCombined,
		0x04, 0x00, OpAck, 0x00, 0x05,
		0x08, 0x00, OpPacket, 0x00, 0x06, 0xAA, 0xBB, 0xCC, 0xDD,
	}
	combined, ok := ParseCombined(pkt, 0, -1)
	if !ok || len(combined.Subs) != 2 {
		t.Fatalf("ok=%v subs=%d", ok, len(combined.Subs))
	}
	if combined.Subs[0].TransportOp != OpAck || combined.Subs[1].TransportOp != OpPacket {
		t.Fatalf("ops=%x %x", combined.Subs[0].TransportOp, combined.Subs[1].TransportOp)
	}
	if combined.Subs[1].Length != 8 {
		t.Fatalf("packet sub len=%d", combined.Subs[1].Length)
	}
}

func TestParseCombinedExtendedLength(t *testing.T) {
	payload := make([]byte, 300)
	for i := range payload {
		payload[i] = byte(i)
	}
	sub := append([]byte{0x00, OpPacket, 0x00, 0x01}, payload...)
	buf := append([]byte{0x00, OpCombined, 0xFF}, byte(len(sub)>>8), byte(len(sub)))
	buf = append(buf, sub...)
	combined, ok := ParseCombined(buf, 0, -1)
	if !ok || len(combined.Subs) != 1 || combined.Subs[0].Length != len(sub) {
		t.Fatalf("ok=%v subs=%v", ok, combined.Subs)
	}
}
