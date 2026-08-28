package proxy

import (
	"context"
	"encoding/binary"
	"net"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/protocol"
	"github.com/alfred-identity/app/internal/router"
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

func combinedLoginPacket(t *testing.T) []byte {
	t.Helper()
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

func TestEngineHandleClientLoginFailDoesNotForward(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := store.UpsertAccount("a1", "secret", []string{"shared"}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertAccount("a2", "secret", []string{"shared"}); err != nil {
		t.Fatal(err)
	}
	e := &Engine{
		Router: &router.Router{
			Local: store,
			BusyFn: func() map[string]bool {
				return map[string]bool{"a1": true, "a2": true}
			},
		},
	}
	pkt := combinedLoginPacket(t)
	login, ok := protocol.ParseLoginPacket(pkt)
	if !ok {
		t.Fatal("parse")
	}
	login.Username = "shared"
	// Rebuild isn't needed — handleClient parses raw bytes; rewrite username in packet via RewriteCredentials then... 
	// Easier: put username "shared" into a packet by rewriting credentials from a parsed login.
	rewritten, err := login.RewriteCredentials("shared", "x")
	if err != nil {
		t.Fatal(err)
	}
	actions := e.handleClient(context.Background(), rewritten)
	if len(actions.SendUpstream) != 0 {
		t.Fatalf("expected drop for busy alias pool, got %d upstream packets", len(actions.SendUpstream))
	}
}

func TestEngineHandleClientLocalLoginForwardsRewritten(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := store.UpsertAccount("user", "secret", nil); err != nil {
		t.Fatal(err)
	}
	e := &Engine{Router: &router.Router{Local: store}}
	actions := e.handleClient(context.Background(), combinedLoginPacket(t))
	if len(actions.SendUpstream) != 1 {
		t.Fatalf("upstream packets=%d", len(actions.SendUpstream))
	}
	if protocol.TransportOpcode(actions.SendUpstream[0]) != protocol.OpCombined {
		t.Fatal("expected combined forward")
	}
}

func TestEngineHandleServerSessionResponseSetsCRC(t *testing.T) {
	e := &Engine{}
	resp := make([]byte, 17)
	resp[0], resp[1] = 0x00, protocol.OpSessionResponse
	resp[10] = 2
	binary.LittleEndian.PutUint32(resp[13:17], 512)
	actions := e.handleServer(context.Background(), resp)
	if len(actions.SendClient) != 1 {
		t.Fatal("expected forward to client")
	}
	if e.crcBytes != 2 || e.maxPacket != 512 {
		t.Fatalf("crcBytes=%d maxPacket=%d", e.crcBytes, e.maxPacket)
	}
}

func TestEngineFinalizeRestoresCRC(t *testing.T) {
	e := &Engine{crcBytes: 2, crcKey: 0x12345678}
	raw := []byte{0x00, protocol.OpPacket, 0x00, 0x01, 0x01}
	out := e.Finalize([][]byte{raw})
	if len(out) != 1 || len(out[0]) != len(raw)+2 {
		t.Fatalf("len=%d", len(out[0]))
	}
	if got := protocol.StripCRC(out[0], 2); string(got) != string(raw) {
		t.Fatal("finalize crc round-trip failed")
	}
}

func TestEngineHandleClientLoginPassthroughForwards(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	e := &Engine{Router: &router.Router{Local: store}}
	actions := e.handleClient(context.Background(), combinedLoginPacket(t))
	if len(actions.SendUpstream) != 1 {
		t.Fatalf("upstream=%d", len(actions.SendUpstream))
	}
}

func TestEngineHandleClientAckForwardsUpstream(t *testing.T) {
	e := &Engine{}
	pkt := []byte{0x00, protocol.OpAck, 0x00, 0x03}
	actions := e.handleClient(context.Background(), pkt)
	if len(actions.SendUpstream) != 1 {
		t.Fatalf("upstream=%d", len(actions.SendUpstream))
	}
}

func TestEngineHandleClientCombinedWithoutLoginForwards(t *testing.T) {
	e := &Engine{}
	pkt := []byte{
		0x00, protocol.OpCombined,
		0x04, 0x00, protocol.OpAck, 0x00, 0x02,
	}
	actions := e.handleClient(context.Background(), pkt)
	if len(actions.SendUpstream) != 1 {
		t.Fatal("expected ack-only combined forward")
	}
}

func TestEngineHandleServerEmptyAndUnknownOpcode(t *testing.T) {
	e := &Engine{}
	if len(e.handleServer(context.Background(), nil).SendClient) != 0 {
		t.Fatal("empty packet")
	}
	unknown := []byte{0x00, 0x99, 0x00, 0x01}
	actions := e.handleServer(context.Background(), unknown)
	if len(actions.SendClient) != 1 {
		t.Fatal("expected default forward")
	}
}

func TestEngineOnDatagramStripsCRCFromClient(t *testing.T) {
	e := &Engine{crcBytes: 2, crcKey: 0x12345678}
	upstream := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5998}
	client := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 50000}
	raw := []byte{0x00, protocol.OpAck, 0x00, 0x01}
	wire := protocol.AppendCRC(raw, e.crcKey, e.crcBytes)
	actions := e.OnDatagram(context.Background(), wire, client, upstream)
	if len(actions.SendUpstream) != 1 {
		t.Fatalf("upstream=%d", len(actions.SendUpstream))
	}
	if len(actions.SendUpstream[0]) != len(raw) {
		t.Fatalf("expected stripped crc len=%d", len(actions.SendUpstream[0]))
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

func TestEngineUpstreamKeepalivePacket(t *testing.T) {
	e := &Engine{}
	if pkt := e.UpstreamKeepalivePacket(); pkt != nil {
		t.Fatal("expected nil before session")
	}
	client := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000}
	resp := make([]byte, 17)
	resp[0], resp[1] = 0x00, protocol.OpSessionResponse
	e.OnDatagram(context.Background(), resp, &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998}, &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998})
	e.OnDatagram(context.Background(), []byte{0x00, protocol.OpKeepAlive}, client, &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998})
	pkt := e.UpstreamKeepalivePacket()
	if pkt == nil {
		t.Fatal("expected keepalive packet")
	}
	if protocol.TransportOpcode(pkt[:len(pkt)-2]) != protocol.OpKeepAlive {
		t.Fatalf("opcode=%x", protocol.TransportOpcode(pkt))
	}
}

func TestEngineForwardsKeepAliveFromServer(t *testing.T) {
	e := &Engine{}
	upstream := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998}
	actions := e.handleServer(context.Background(), []byte{0x00, protocol.OpKeepAlive})
	if len(actions.SendClient) != 1 || protocol.TransportOpcode(actions.SendClient[0]) != protocol.OpKeepAlive {
		t.Fatalf("actions=%#v", actions)
	}
	_ = upstream
}
