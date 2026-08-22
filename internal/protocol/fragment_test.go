package protocol

import "testing"

func testPayload(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

func TestFragmentAddRejectsShortDatagram(t *testing.T) {
	var asm FragmentAssembler
	if out, ok := asm.Add(1, []byte{1, 2}); ok || out != nil {
		t.Fatal("expected reject short fragment")
	}
}

func TestFragmentAssemblerResetClearsState(t *testing.T) {
	var asm FragmentAssembler
	payload := testPayload(20)
	frags := BuildFragments(payload, 1, 12)
	for i, raw := range frags {
		got, ok := asm.Add(uint16(1+i), raw)
		if i < len(frags)-1 {
			if ok || got != nil {
				t.Fatal("should not complete early")
			}
			continue
		}
		if !ok || string(got) != string(payload) {
			t.Fatal("expected completion")
		}
	}
	asm.Reset()
	if asm.IsActive() {
		t.Fatal("reset should clear assembler")
	}
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
