package protocol

import "encoding/binary"

type SessionResponse struct {
	ConnectCode   uint32
	EncodeKey     uint32
	CRCBytes      byte
	EncodePass1   byte
	EncodePass2   byte
	MaxPacketSize uint32
}

func TransportOpcode(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return binary.BigEndian.Uint16(data[0:2])
}

func GetSequence(data []byte, offset int) uint16 {
	if len(data) < offset+4 {
		return 0
	}
	return binary.BigEndian.Uint16(data[offset+2 : offset+4])
}

func SetSequence(buf []byte, offset int, seq uint16) {
	if len(buf) < offset+4 {
		return
	}
	binary.BigEndian.PutUint16(buf[offset+2:offset+4], seq)
}

func ParseSessionResponse(data []byte) (SessionResponse, bool) {
	if len(data) < 17 {
		return SessionResponse{}, false
	}
	return SessionResponse{
		ConnectCode:   binary.BigEndian.Uint32(data[2:6]),
		EncodeKey:     binary.BigEndian.Uint32(data[6:10]),
		CRCBytes:      data[10],
		EncodePass1:   data[11],
		EncodePass2:   data[12],
		MaxPacketSize: binary.LittleEndian.Uint32(data[13:17]),
	}, true
}

func PacketUsesCRC(data []byte) bool {
	op := TransportOpcode(data)
	return op != OpSessionRequest && op != OpSessionResponse
}
