package proxy

import (
	"context"
	"net"
	"testing"

	"github.com/alfred-identity/app/internal/protocol"
)

func TestEngineHandleServerPacketRewritesSeq(t *testing.T) {
	e := &Engine{}
	pkt := []byte{0x00, protocol.OpPacket, 0x00, 0x04, 0x01, 0x02, 0x03}
	actions := e.handleServer(context.Background(), append([]byte{}, pkt...))
	if len(actions.SendClient) != 1 {
		t.Fatalf("client packets=%d", len(actions.SendClient))
	}
	if protocol.GetSequence(actions.SendClient[0], 0) != 0 {
		t.Fatalf("seq=%d", protocol.GetSequence(actions.SendClient[0], 0))
	}
	if e.Session.SeqToClient != 1 {
		t.Fatalf("seq_to_client=%d", e.Session.SeqToClient)
	}
}

func TestEngineHandleServerCombinedForwardsRewritten(t *testing.T) {
	e := &Engine{}
	pkt := []byte{
		0x00, protocol.OpCombined,
		0x04, 0x00, protocol.OpAck, 0x00, 0x01,
		0x04, 0x00, protocol.OpPacket, 0x00, 0x02,
	}
	actions := e.handleServer(context.Background(), pkt)
	if len(actions.SendClient) != 1 {
		t.Fatal("expected combined forward")
	}
	if protocol.GetSequence(actions.SendClient[0], 8) != 0 {
		t.Fatalf("packet seq=%d", protocol.GetSequence(actions.SendClient[0], 8))
	}
}

func TestEngineHandleServerFragmentForwardsFilteredList(t *testing.T) {
	e := &Engine{maxPacket: 512}
	// Minimal valid server-list app payload (one P99 server).
	rawEntry := append([]byte("127.0.0.1\x00"), 0, 0, 0, 1, 0, 0, 0, 1)
	rawEntry = append(rawEntry, []byte("Project 1999: Blue (Velious, PvE)\x00EN\x00US\x00")...)
	rawEntry = append(rawEntry, 0, 0, 0, 0, 0, 0, 0, 1)
	header := [16]byte{}
	app := append([]byte{0x18, 0x00}, header[:]...)
	app = append(app, 0, 0, 0, 1)
	app = append(app, rawEntry...)
	frags := protocol.BuildFragments(app, 2, 512)
	var actions Actions
	for _, frag := range frags {
		actions = e.handleServer(context.Background(), frag)
	}
	if len(actions.SendClient) != 1 {
		t.Fatalf("expected one client packet, got %d", len(actions.SendClient))
	}
	if protocol.TransportOpcode(actions.SendClient[0]) != protocol.OpPacket {
		t.Fatalf("opcode=%x", protocol.TransportOpcode(actions.SendClient[0]))
	}
}

func TestEngineOnDatagramUpstreamUsesNormalizeAddr(t *testing.T) {
	e := &Engine{}
	upstream := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998}
	mapped := &net.UDPAddr{IP: net.ParseIP("::ffff:1.2.3.4"), Port: 5998}
	resp := make([]byte, 17)
	resp[0], resp[1] = 0x00, 0x02
	actions := e.OnDatagram(context.Background(), resp, mapped, upstream)
	if len(actions.SendClient) != 1 {
		t.Fatal("expected upstream packet handling")
	}
}
