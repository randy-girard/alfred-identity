package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	loginBaseSize   = 10
	ackEnd          = 7
	loginSubHeader  = 6
	loginEncOffset  = loginSubHeader + loginBaseSize
)

// LoginPacket is a parsed Combined login datagram (p99-login-proxy layout).
type LoginPacket struct {
	Buf        []byte
	Username   string
	Password   string
	sub2Offset int
	sub2Len    int
	encOffset  int
}

// ParseLoginPacket parses a Combined login packet after CRC has been stripped.
func ParseLoginPacket(buf []byte) (*LoginPacket, bool) {
	if len(buf) < 30 || !bytes.HasPrefix(buf, []byte{0x00, 0x03, 0x04, 0x00, 0x15}) {
		return nil, false
	}
	sub2Len := int(buf[ackEnd])
	sub2Start := ackEnd + 1
	if sub2Start+sub2Len > len(buf) || sub2Len < loginSubHeader {
		return nil, false
	}
	sub2 := buf[sub2Start : sub2Start+sub2Len]
	if binary.BigEndian.Uint16(sub2[0:2]) != OpPacket {
		return nil, false
	}
	if len(sub2) < 6 || binary.LittleEndian.Uint16(sub2[4:6]) != AppLogin {
		return nil, false
	}
	if len(sub2) <= loginEncOffset {
		return nil, false
	}
	encrypted := sub2[loginEncOffset:]
	user, pass, ok := decryptCredentials(encrypted)
	if !ok {
		return nil, false
	}
	return &LoginPacket{
		Buf:        append([]byte{}, buf...),
		Username:   user,
		Password:   pass,
		sub2Offset: sub2Start,
		sub2Len:    sub2Len,
		encOffset:  loginEncOffset,
	}, true
}

func decryptCredentials(encrypted []byte) (username, password string, ok bool) {
	pt, err := DecryptDES(encrypted)
	if err != nil {
		return "", "", false
	}
	firstNul := bytes.IndexByte(pt, 0)
	var userBytes, rest []byte
	if firstNul >= 0 {
		userBytes = pt[:firstNul]
		rest = pt[firstNul+1:]
	} else {
		userBytes = pt
		rest = nil
	}
	secondNul := bytes.IndexByte(rest, 0)
	var passBytes []byte
	if secondNul >= 0 {
		passBytes = rest[:secondNul]
	} else {
		passBytes = rest
	}
	return string(userBytes), string(passBytes), true
}

// SpliceEncryptedCredentials replaces the DES ciphertext region (p99-login-proxy parity).
func (lp *LoginPacket) SpliceEncryptedCredentials(encrypted []byte) ([]byte, error) {
	newSubLen := lp.encOffset + len(encrypted)
	if newSubLen > 0xFF {
		return nil, fmt.Errorf("login subpacket too large (%d)", newSubLen)
	}
	absStart := lp.sub2Offset + lp.encOffset
	absEnd := lp.sub2Offset + lp.sub2Len
	out := append([]byte{}, lp.Buf[:absStart]...)
	out = append(out, encrypted...)
	out = append(out, lp.Buf[absEnd:]...)
	out[lp.sub2Offset-1] = byte(newSubLen)
	return out, nil
}

// RewriteCredentials encrypts username/password and splices into the packet.
func (lp *LoginPacket) RewriteCredentials(username, password string) ([]byte, error) {
	ct, err := EncryptCredentials(username, password)
	if err != nil {
		return nil, err
	}
	return lp.SpliceEncryptedCredentials(ct)
}
