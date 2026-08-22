package protocol

import (
	"encoding/binary"
	"testing"
)

func TestParseSessionResponse(t *testing.T) {
	resp := make([]byte, 17)
	resp[0], resp[1] = 0x00, 0x02
	binary.BigEndian.PutUint32(resp[2:6], 0x11111111)
	binary.BigEndian.PutUint32(resp[6:10], 0x12345678)
	resp[10] = 2
	resp[11], resp[12] = 0x01, 0x02
	binary.LittleEndian.PutUint32(resp[13:17], 512)

	got, ok := ParseSessionResponse(resp)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.ConnectCode != 0x11111111 || got.EncodeKey != 0x12345678 || got.CRCBytes != 2 || got.MaxPacketSize != 512 {
		t.Fatalf("got=%+v", got)
	}
	if _, ok := ParseSessionResponse(resp[:16]); ok {
		t.Fatal("short packet should fail")
	}
}

func TestPacketUsesCRCAndTransportOpcode(t *testing.T) {
	if TransportOpcode([]byte{0x00}) != 0 {
		t.Fatal("short opcode")
	}
	if !PacketUsesCRC([]byte{0x00, OpKeepAlive}) {
		t.Fatal("keepalive uses crc")
	}
	if PacketUsesCRC([]byte{0x00, OpSessionRequest}) {
		t.Fatal("session request exempt")
	}
	if PacketUsesCRC([]byte{0x00, OpSessionResponse}) {
		t.Fatal("session response exempt")
	}
}

func TestSetAndGetSequence(t *testing.T) {
	buf := []byte{0x00, OpPacket, 0x00, 0x00}
	SetSequence(buf, 0, 0xABCD)
	if GetSequence(buf, 0) != 0xABCD {
		t.Fatalf("seq=%x", GetSequence(buf, 0))
	}
	SetSequence(buf, 0, 0) // too short offset safety
	if len(buf) != 4 {
		t.Fatal("buffer unchanged length")
	}
}
