package protocol

import "testing"

func TestProxySessionRecvCombinedRewritesServerSeq(t *testing.T) {
	pkt := []byte{
		0x00, OpCombined,
		0x04, 0x00, OpAck, 0x00, 0x02,
		0x06, 0x00, OpPacket, 0x00, 0x03, 0x01, 0x02,
	}
	var sess ProxySessionState
	sess.SeqFromServer = 3
	out := sess.RecvCombined(pkt, 0, len(pkt))
	if len(out) != len(pkt) {
		t.Fatalf("len=%d", len(out))
	}
	if GetSequence(out, 8) != 0 {
		t.Fatalf("client packet seq=%d", GetSequence(out, 8))
	}
	if sess.SeqToClient != 1 {
		t.Fatalf("seq_to_client=%d", sess.SeqToClient)
	}
}

func TestProxySessionAdjustCombinedRewritesClientAck(t *testing.T) {
	pkt := []byte{
		0x00, OpCombined,
		0x04, 0x00, OpAck, 0x00, 0x99,
		0x06, 0x00, OpPacket, 0x00, 0x01, 0x01, 0x02,
	}
	var sess ProxySessionState
	sess.SeqFromServer = 4
	sess.AdjustCombined(pkt)
	if GetSequence(pkt, 3) != 3 {
		t.Fatalf("ack seq=%d", GetSequence(pkt, 3))
	}
}

func TestProxySessionCSOffsetShiftsClientPacketSeq(t *testing.T) {
	pkt := []byte{0x00, OpPacket, 0x00, 0x05, 0x01}
	var sess ProxySessionState
	sess.CSOffset = 2
	sess.AdjustClientPacket(pkt, 0)
	if GetSequence(pkt, 0) != 7 {
		t.Fatalf("seq=%d", GetSequence(pkt, 0))
	}
	sess.AdjustServerAck(pkt, 0)
	if GetSequence(pkt, 0) != 5 {
		t.Fatalf("server ack seq=%d", GetSequence(pkt, 0))
	}
}

func TestProxySessionResetClearsServerListFlag(t *testing.T) {
	var sess ProxySessionState
	sess.serverListForwarded = true
	sess.SeqToClient = 9
	sess.Reset()
	if sess.serverListForwarded || sess.SeqToClient != 0 {
		t.Fatal("reset incomplete")
	}
}

func TestProxySessionBuildClientOutboundFragmentsLargePayload(t *testing.T) {
	var sess ProxySessionState
	app := make([]byte, 600)
	out := sess.buildClientOutbound(app, 128)
	if len(out) < 2 {
		t.Fatalf("frags=%d", len(out))
	}
	if TransportOpcode(out[0]) != OpFragment {
		t.Fatalf("first opcode=%x", TransportOpcode(out[0]))
	}
}

func TestProxySessionAdjustAckUsesSeqFromServer(t *testing.T) {
	buf := []byte{0x00, OpAck, 0x00, 0x99}
	var sess ProxySessionState
	sess.SeqFromServer = 5
	sess.AdjustAck(buf, 0)
	if GetSequence(buf, 0) != 4 {
		t.Fatalf("ack seq=%d", GetSequence(buf, 0))
	}
}

func TestProxySessionRecvPacketRewritesSeq(t *testing.T) {
	buf := []byte{0x00, OpPacket, 0x00, 0x08, 0x01}
	var sess ProxySessionState
	sess.SeqFromServer = 8
	sess.RecvPacket(buf, 0)
	if GetSequence(buf, 0) != 0 {
		t.Fatalf("client seq=%d", GetSequence(buf, 0))
	}
	if sess.SeqToClient != 1 || sess.SeqFromServer != 9 {
		t.Fatalf("to_client=%d from_server=%d", sess.SeqToClient, sess.SeqFromServer)
	}
}

func TestProxySessionRecvFragmentIgnoresNonServerListOpcode(t *testing.T) {
	app := append([]byte{0x02, 0x00}, []byte("not-server-list")...)
	frags := BuildFragments(app, 1, 64)
	var sess ProxySessionState
	for _, raw := range frags {
		if out := sess.RecvFragment(raw, 512); len(out) != 0 {
			t.Fatal("expected non-server-list fragment to be dropped")
		}
	}
}
