package protocol

import (
	"crypto/cipher"
	"crypto/des"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

var (
	desKey = make([]byte, 8)
	desIV  = make([]byte, 8)
)

const GoldenUserPassCipher = "575ab3e46810e874f75cb31595902052"

func EncryptCredentials(username, password string) ([]byte, error) {
	plain := append([]byte(username), 0)
	plain = append(plain, password...)
	plain = append(plain, 0)
	return EncryptDES(plain)
}

func EncryptDES(plain []byte) ([]byte, error) {
	block, err := des.NewCipher(desKey)
	if err != nil {
		return nil, err
	}
	padded := zeroPad(plain, 8)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, desIV).CryptBlocks(out, padded)
	return out, nil
}

func DecryptDES(ct []byte) ([]byte, error) {
	if len(ct) == 0 || len(ct)%8 != 0 {
		return nil, fmt.Errorf("bad ciphertext length")
	}
	block, err := des.NewCipher(desKey)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, desIV).CryptBlocks(out, ct)
	return out, nil
}

func zeroPad(b []byte, bs int) []byte {
	if len(b)%bs == 0 {
		return b
	}
	n := bs - len(b)%bs
	out := make([]byte, len(b)+n)
	copy(out, b)
	return out
}

func GoldenBytes() []byte {
	b, _ := hex.DecodeString(GoldenUserPassCipher)
	return b
}

// Transport opcodes (big-endian).
const (
	OpSessionRequest  = 0x0001
	OpSessionResponse = 0x0002
	OpCombined        = 0x0003
	OpDisconnect      = 0x0005
	OpKeepAlive       = 0x0006
	OpPacket          = 0x0009
	OpFragment        = 0x000D
	OpAck             = 0x0015
)

// App opcodes (little-endian).
const (
	AppLogin = 0x0002
)

// FindLoginCipherOffset returns the start index of DES ciphertext inside a Combined Login packet,
// or -1 if not found. Also returns the index of the Packet sublen byte to update on splice.
func FindLoginCipherOffset(pkt []byte) (cipherStart, sublenIdx int, ok bool) {
	if len(pkt) < 12 || binary.BigEndian.Uint16(pkt[0:2]) != OpCombined {
		return 0, 0, false
	}
	i := 2
	for i < len(pkt) {
		if i >= len(pkt) {
			break
		}
		sublenIdx = i
		sublen := int(pkt[i])
		i++
		if sublen == 0xFF {
			if i+2 > len(pkt) {
				return 0, 0, false
			}
			sublen = int(binary.BigEndian.Uint16(pkt[i : i+2]))
			i += 2
		}
		if i+sublen > len(pkt) {
			return 0, 0, false
		}
		sub := pkt[i : i+sublen]
		if len(sub) >= 14 && binary.BigEndian.Uint16(sub[0:2]) == OpPacket {
			// Packet: op(2) seq(2) + app op LE(2) + LoginBase(10) + cipher
			appOff := 4
			if binary.LittleEndian.Uint16(sub[appOff:appOff+2]) == AppLogin {
				cipherStart = i + 4 + 2 + 10 // packet hdr + app op + LoginBase
				if cipherStart <= i+sublen {
					return cipherStart, sublenIdx, true
				}
			}
		}
		i += sublen
	}
	return 0, 0, false
}

// SpliceLoginCredentials replaces DES ciphertext in a Combined Login packet.
func SpliceLoginCredentials(pkt []byte, cipher []byte) ([]byte, error) {
	if lp, ok := ParseLoginPacket(pkt); ok {
		return lp.SpliceEncryptedCredentials(cipher)
	}
	return nil, fmt.Errorf("login combined not found")
}

// RewriteLoginPacket packs new user/pass and splices using the p99 login layout.
func RewriteLoginPacket(pkt []byte, username, password string) ([]byte, error) {
	lp, ok := ParseLoginPacket(pkt)
	if !ok {
		return nil, fmt.Errorf("login combined not found")
	}
	return lp.RewriteCredentials(username, password)
}

// SpliceCipherBlob splices a pre-encrypted DES blob from the daemon.
func SpliceCipherBlob(pkt []byte, cipher []byte) ([]byte, error) {
	return SpliceLoginCredentials(pkt, cipher)
}
