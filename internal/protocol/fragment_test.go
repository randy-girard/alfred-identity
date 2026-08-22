package protocol

import "testing"

func testPayload(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

func TestFragmentAssemblerReassembles(t *testing.T) {
	payload := testPayload(40)
	frags := BuildFragments(payload, 10, 16)
	var asm FragmentAssembler
	var got []byte
	var ok bool
	for i, raw := range frags {
		seq := uint16(10 + i)
		got, ok = asm.Add(seq, raw)
		if i < len(frags)-1 && ok {
			t.Fatalf("fragment %d completed early", i)
		}
	}
	if !ok || string(got) != string(payload) {
		t.Fatalf("reassemble failed ok=%v len=%d", ok, len(got))
	}
}

func TestRecvFragmentBuildsClientPacket(t *testing.T) {
	app := append([]byte{0x18, 0x00}, []byte("server-list-bytes")...)
	frags := BuildFragments(app, 5, 64)
	var sess ProxySessionState
	var out [][]byte
	for i, raw := range frags {
		out = sess.RecvFragment(raw, 512)
		if i < len(frags)-1 && len(out) != 0 {
			t.Fatalf("unexpected early output on frag %d", i)
		}
	}
	if len(out) != 1 {
		t.Fatalf("expected one client packet, got %d", len(out))
	}
	if TransportOpcode(out[0]) != OpPacket {
		t.Fatalf("opcode=%x", TransportOpcode(out[0]))
	}
	if string(out[0][4:]) != string(app) {
		t.Fatalf("payload mismatch")
	}
}

func TestBuildFragmentsRoundTrip(t *testing.T) {
	payload := testPayload(100)
	frags := BuildFragments(payload, 1, 32)
	var asm FragmentAssembler
	var got []byte
	var ok bool
	for i, raw := range frags {
		got, ok = asm.Add(uint16(1+i), raw)
	}
	if !ok || string(got) != string(payload) {
		t.Fatal("round trip failed")
	}
}
