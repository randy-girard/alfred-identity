package protocol

import (
	"encoding/binary"
	"strings"
)

const AppServerListResponse = 0x0018

var P99ServerPrefixes = []string{"project 1999", "an interesting"}

type ServerEntry struct {
	IP          string
	ListID      uint32
	RuntimeID   uint32
	Name        string
	Language    string
	Region      string
	Status      uint32
	PlayerCount uint32
	Raw         []byte
}

func AppOpcode(appPayload []byte) uint16 {
	if len(appPayload) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(appPayload[0:2])
}

func ParseServerList(appPayload []byte) ([]ServerEntry, [16]byte, bool) {
	if len(appPayload) < 20 {
		return nil, [16]byte{}, false
	}
	data := appPayload[2:]
	var header [16]byte
	copy(header[:], data[:16])
	count := int(binary.LittleEndian.Uint32(data[16:20]))
	pos := 20
	var servers []ServerEntry
	for pos < len(data) && len(servers) < count {
		start := pos
		ip, ok := readCString(data, &pos)
		if !ok {
			break
		}
		listID, ok := readU32LE(data, &pos)
		if !ok {
			break
		}
		runtimeID, ok := readU32LE(data, &pos)
		if !ok {
			break
		}
		name, ok := readCString(data, &pos)
		if !ok {
			break
		}
		language, ok := readCString(data, &pos)
		if !ok {
			break
		}
		region, ok := readCString(data, &pos)
		if !ok {
			break
		}
		status, ok := readU32LE(data, &pos)
		if !ok {
			break
		}
		playerCount, ok := readU32LE(data, &pos)
		if !ok {
			break
		}
		servers = append(servers, ServerEntry{
			IP: ip, ListID: listID, RuntimeID: runtimeID, Name: name,
			Language: language, Region: region, Status: status, PlayerCount: playerCount,
			Raw: append([]byte{}, data[start:pos]...),
		})
	}
	return servers, header, true
}

func FilterP99Servers(servers []ServerEntry) []ServerEntry {
	var out []ServerEntry
	for _, s := range servers {
		lower := strings.ToLower(s.Name)
		for _, p := range P99ServerPrefixes {
			if strings.HasPrefix(lower, p) {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

func BuildServerListResponse(servers []ServerEntry, header [16]byte) []byte {
	out := make([]byte, 0, 22+len(servers)*64)
	out = append(out, byte(AppServerListResponse), 0)
	out = append(out, header[:]...)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(servers)))
	out = append(out, buf...)
	for _, s := range servers {
		out = append(out, s.Raw...)
	}
	return out
}

func readU32LE(data []byte, pos *int) (uint32, bool) {
	if *pos+4 > len(data) {
		return 0, false
	}
	v := binary.LittleEndian.Uint32(data[*pos : *pos+4])
	*pos += 4
	return v, true
}

func readCString(data []byte, pos *int) (string, bool) {
	if *pos >= len(data) {
		return "", false
	}
	end := *pos
	for end < len(data) && data[end] != 0 {
		end++
	}
	if end >= len(data) {
		return "", false
	}
	s := string(data[*pos:end])
	*pos = end + 1
	return s, true
}
