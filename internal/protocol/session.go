package protocol

import "encoding/binary"

// ProxySessionState tracks SOE sequence numbers between client and upstream (p99-login-proxy).
type ProxySessionState struct {
	SeqToClient   uint32
	SeqFromServer uint32
	CSOffset      uint32
	fragments     FragmentAssembler
}

func (s *ProxySessionState) Reset() {
	s.SeqToClient = 0
	s.SeqFromServer = 0
	s.CSOffset = 0
	s.fragments.Reset()
}

func (s *ProxySessionState) AdjustCombined(buf []byte) {
	combined, ok := ParseCombined(buf, 0, -1)
	if !ok {
		return
	}
	for _, sub := range combined.Subs {
		switch sub.TransportOp {
		case OpAck:
			s.rewriteAck(buf, sub.Offset)
		case OpPacket:
			if s.CSOffset != 0 {
				s.shiftPacketSeq(buf, sub.Offset, s.CSOffset)
			}
		}
	}
}

func (s *ProxySessionState) AdjustAck(buf []byte, offset int) {
	s.rewriteAck(buf, offset)
}

func (s *ProxySessionState) AdjustClientPacket(buf []byte, offset int) {
	if s.CSOffset != 0 {
		s.shiftPacketSeq(buf, offset, s.CSOffset)
	}
}

func (s *ProxySessionState) AdjustServerAck(buf []byte, offset int) {
	if s.CSOffset == 0 {
		return
	}
	cur := GetSequence(buf, offset)
	newSeq := cur - uint16(s.CSOffset)
	SetSequence(buf, offset, newSeq)
}

func (s *ProxySessionState) RecvCombined(buf []byte, start int, length int) []byte {
	end := len(buf)
	if length >= 0 {
		end = start + length
	}
	combined, ok := ParseCombined(buf, start, length)
	if ok {
		for _, sub := range combined.Subs {
			switch sub.TransportOp {
			case OpAck:
				s.AdjustServerAck(buf, sub.Offset)
			case OpPacket:
				s.rewriteServerPacketSeq(buf, sub.Offset)
			}
		}
	}
	if start == 0 && end == len(buf) {
		out := make([]byte, len(buf))
		copy(out, buf)
		return out
	}
	return append([]byte{}, buf[start:end]...)
}

func (s *ProxySessionState) RecvPacket(buf []byte, offset int) {
	s.rewriteServerPacketSeq(buf, offset)
}

// RecvFragment reassembles server Fragment datagrams and returns client-bound packet(s).
func (s *ProxySessionState) RecvFragment(raw []byte, maxPacket int) [][]byte {
	if len(raw) < 4 {
		return nil
	}
	serverSeq := GetSequence(raw, 0)
	if uint32(serverSeq)+1 > s.SeqFromServer {
		s.SeqFromServer = uint32(serverSeq) + 1
	}
	app, ok := s.fragments.Add(serverSeq, raw)
	if !ok || app == nil {
		return nil
	}
	return s.buildClientOutbound(app, maxPacket)
}

func (s *ProxySessionState) buildClientOutbound(app []byte, maxPacket int) [][]byte {
	if maxPacket <= 0 {
		maxPacket = 512
	}
	if len(app)+4 <= maxPacket {
		pkt := make([]byte, 4+len(app))
		binary.BigEndian.PutUint16(pkt[0:2], OpPacket)
		binary.BigEndian.PutUint16(pkt[2:4], uint16(s.SeqToClient))
		copy(pkt[4:], app)
		s.SeqToClient++
		return [][]byte{pkt}
	}
	frags := BuildFragments(app, uint16(s.SeqToClient), maxPacket)
	s.SeqToClient += uint32(len(frags))
	return frags
}

func (s *ProxySessionState) rewriteAck(buf []byte, offset int) {
	var newSeq uint16
	if s.SeqFromServer > 0 {
		newSeq = uint16(s.SeqFromServer - 1)
	}
	SetSequence(buf, offset, newSeq)
}

func (s *ProxySessionState) shiftPacketSeq(buf []byte, offset int, delta uint32) {
	cur := GetSequence(buf, offset)
	SetSequence(buf, offset, cur+uint16(delta))
}

func (s *ProxySessionState) rewriteServerPacketSeq(buf []byte, offset int) {
	serverSeq := GetSequence(buf, offset)
	SetSequence(buf, offset, uint16(s.SeqToClient))
	s.SeqToClient++
	if uint32(serverSeq) == s.SeqFromServer {
		s.SeqFromServer++
	}
}
