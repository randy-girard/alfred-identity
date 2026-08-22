package protocol

import (
	"bufio"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReassembleCapturedServerListFragments(t *testing.T) {
	fragPath := filepath.Join("testdata", "server_list_fragments.hexlist")
	assembledPath := filepath.Join("testdata", "server_list_assembled.hex")

	wantHex, err := os.ReadFile(assembledPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString(strings.TrimSpace(string(wantHex)))
	if err != nil {
		t.Fatal(err)
	}
	servers, header, ok := ParseServerList(want)
	if !ok {
		t.Fatal("parse assembled fixture")
	}
	filtered := FilterP99Servers(servers)
	if len(filtered) != 4 {
		t.Fatalf("filtered count=%d", len(filtered))
	}
	expectApp := BuildServerListResponse(filtered, header)

	f, err := os.Open(fragPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var sess ProxySessionState
	var out [][]byte
	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		raw, err := hex.DecodeString(line)
		if err != nil {
			t.Fatalf("line %d decode: %v", lines+1, err)
		}
		if pkt := sess.RecvFragment(raw, 512); len(pkt) > 0 {
			out = pkt
		}
		lines++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one filtered client packet, got %d from %d fragments", len(out), lines)
	}
	if TransportOpcode(out[0]) != OpPacket {
		t.Fatalf("opcode=%x", TransportOpcode(out[0]))
	}
	app := out[0][4:]
	if string(app) != string(expectApp) {
		t.Fatalf("filtered payload mismatch len=%d expect=%d", len(app), len(expectApp))
	}
}

func TestServerListForwardIgnoresDuplicateCompletions(t *testing.T) {
	fragPath := filepath.Join("testdata", "server_list_fragments.hexlist")
	f, err := os.Open(fragPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var sess ProxySessionState
	forwards := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		raw, err := hex.DecodeString(line)
		if err != nil {
			t.Fatal(err)
		}
		if out := sess.RecvFragment(raw, 512); len(out) > 0 {
			forwards++
		}
	}
	if forwards != 1 {
		t.Fatalf("forwards=%d want 1", forwards)
	}
}

func TestParseAndFilterServerListFixture(t *testing.T) {
	wantHex, err := os.ReadFile(filepath.Join("testdata", "server_list_assembled.hex"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString(strings.TrimSpace(string(wantHex)))
	if err != nil {
		t.Fatal(err)
	}
	servers, _, ok := ParseServerList(want)
	if !ok || len(servers) != 110 {
		t.Fatalf("parse ok=%v count=%d", ok, len(servers))
	}
	filtered := FilterP99Servers(servers)
	if len(filtered) != 4 {
		t.Fatalf("filtered=%d", len(filtered))
	}
}
