package protocol

import "encoding/binary"

type SubPacket struct {
	Offset      int
	Length      int
	TransportOp uint16
}

type CombinedPacket struct {
	Subs []SubPacket
}

func ParseCombined(buf []byte, start int, length int) (CombinedPacket, bool) {
	end := len(buf)
	if length >= 0 {
		end = start + length
	}
	if end > len(buf) || start+2 > end {
		return CombinedPacket{}, false
	}
	pos := start + 2
	var subs []SubPacket
	for pos < end {
		if pos >= len(buf) {
			break
		}
		sublen := int(buf[pos])
		pos++
		if sublen == 0xFF {
			if pos+2 > end {
				break
			}
			sublen = int(binary.BigEndian.Uint16(buf[pos : pos+2]))
			pos += 2
		}
		if sublen == 0 || pos+sublen > end {
			break
		}
		op := uint16(0)
		if pos+2 <= len(buf) {
			op = binary.BigEndian.Uint16(buf[pos : pos+2])
		}
		subs = append(subs, SubPacket{Offset: pos, Length: sublen, TransportOp: op})
		pos += sublen
	}
	return CombinedPacket{Subs: subs}, true
}
