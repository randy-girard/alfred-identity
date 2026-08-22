package protocol

import (
	"encoding/binary"
)

const (
	firstFragOverhead      = 8
	subsequentFragOverhead = 4
)

// FragmentAssembler reassembles SOE Fragment datagrams (p99-login-proxy parity).
type FragmentAssembler struct {
	fragments map[uint16][]byte
	totalLen  *uint32
	firstSeq  *uint16
}

func (a *FragmentAssembler) Reset() {
	a.fragments = nil
	a.totalLen = nil
	a.firstSeq = nil
}

func (a *FragmentAssembler) IsActive() bool {
	return a.firstSeq != nil
}

// Add ingests one Fragment datagram. Returns assembled app payload when complete.
func (a *FragmentAssembler) Add(seq uint16, rawFrag []byte) ([]byte, bool) {
	if len(rawFrag) < 4 {
		return nil, false
	}
	fragData := rawFrag[4:]

	if a.fragments == nil {
		a.fragments = map[uint16][]byte{}
	}

	startsNew := false
	if a.firstSeq != nil {
		first := *a.firstSeq
		if seq != first && seq-first > 0x8000 { // wrapping_sub > u16::MAX/2
			startsNew = true
		}
	}

	if a.firstSeq == nil || startsNew {
		if startsNew {
			a.fragments = map[uint16][]byte{}
			a.totalLen = nil
		}
		if len(fragData) < 4 {
			return nil, false
		}
		total := binary.BigEndian.Uint32(fragData[0:4])
		first := seq
		a.firstSeq = &first
		a.totalLen = &total
		a.fragments[seq] = append([]byte{}, fragData[4:]...)
	} else if *a.firstSeq == seq {
		if len(fragData) < 4 {
			return nil, false
		}
		total := binary.BigEndian.Uint32(fragData[0:4])
		if a.totalLen == nil || *a.totalLen != total {
			return nil, false
		}
		a.fragments[seq] = append([]byte{}, fragData[4:]...)
	} else {
		a.fragments[seq] = append([]byte{}, fragData...)
	}

	if a.totalLen == nil || a.firstSeq == nil {
		return nil, false
	}
	totalLen := int(*a.totalLen)
	if a.contiguousLen() < totalLen {
		return nil, false
	}
	out := a.reassemble(totalLen)
	a.Reset()
	return out, true
}

func (a *FragmentAssembler) contiguousLen() int {
	if a.firstSeq == nil {
		return 0
	}
	seq := *a.firstSeq
	total := 0
	for i := 0; i < len(a.fragments); i++ {
		payload, ok := a.fragments[seq]
		if !ok {
			break
		}
		total += len(payload)
		seq++
	}
	return total
}

func (a *FragmentAssembler) reassemble(totalLen int) []byte {
	seq := *a.firstSeq
	out := make([]byte, 0, totalLen)
	for len(out) < totalLen {
		payload, ok := a.fragments[seq]
		if !ok {
			break
		}
		out = append(out, payload...)
		seq++
	}
	if len(out) > totalLen {
		out = out[:totalLen]
	}
	return out
}

// BuildFragments splits an app payload into SOE Fragment datagram bodies (opcode+seq+payload).
func BuildFragments(appPayload []byte, startSeq uint16, maxPacket int) [][]byte {
	if maxPacket <= 0 {
		maxPacket = 512
	}
	totalLen := len(appPayload)
	firstCap := maxPacket - firstFragOverhead
	if firstCap < 1 {
		firstCap = 1
	}
	subCap := maxPacket - subsequentFragOverhead
	if subCap < 1 {
		subCap = 1
	}

	var frags [][]byte
	firstChunkEnd := totalLen
	if firstChunkEnd > firstCap {
		firstChunkEnd = firstCap
	}
	first := make([]byte, firstFragOverhead+firstChunkEnd)
	binary.BigEndian.PutUint16(first[0:2], OpFragment)
	binary.BigEndian.PutUint16(first[2:4], startSeq)
	binary.BigEndian.PutUint32(first[4:8], uint32(totalLen))
	copy(first[8:], appPayload[:firstChunkEnd])
	frags = append(frags, first)

	pos := firstChunkEnd
	seq := startSeq + 1
	for pos < totalLen {
		end := pos + subCap
		if end > totalLen {
			end = totalLen
		}
		frag := make([]byte, subsequentFragOverhead+(end-pos))
		binary.BigEndian.PutUint16(frag[0:2], OpFragment)
		binary.BigEndian.PutUint16(frag[2:4], seq)
		copy(frag[4:], appPayload[pos:end])
		frags = append(frags, frag)
		pos = end
		seq++
	}
	return frags
}
