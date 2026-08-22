package proxy

import (
	"context"
	"net"
	"testing"
)

func TestEngineRestoresCRCOnOutbound(t *testing.T) {
	e := &Engine{crcBytes: 2, crcKey: 0x12345678}
	keepalive := []byte{0x00, 0x06}
	out := e.Finalize([][]byte{keepalive})
	if len(out[0]) != len(keepalive)+2 {
		t.Fatalf("expected crc suffix, len=%d", len(out[0]))
	}
}

func TestEngineSessionResponseSetsCRC(t *testing.T) {
	e := &Engine{}
	resp := make([]byte, 17)
	resp[0], resp[1] = 0x00, 0x02
	resp[6], resp[7], resp[8], resp[9] = 0x12, 0x34, 0x56, 0x78
	resp[10] = 2
	actions := e.handleServer(context.Background(), resp)
	if e.crcBytes != 2 || e.crcKey != 0x12345678 {
		t.Fatalf("crc_bytes=%d crc_key=%x", e.crcBytes, e.crcKey)
	}
	if len(actions.SendClient) != 1 {
		t.Fatal("expected forward to client")
	}
}

func TestEngineClientCombinedForwardsUpstream(t *testing.T) {
	e := &Engine{}
	client := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 50000}
	upstream := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5998}
	ka := []byte{0x00, 0x06}
	actions := e.OnDatagram(context.Background(), ka, client, upstream)
	if len(actions.SendUpstream) != 1 {
		t.Fatalf("upstream=%d", len(actions.SendUpstream))
	}
}
