package proxy

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alfred-identity/app/internal/protocol"
)

func TestIsUpstreamPeer(t *testing.T) {
	up := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998}
	peer := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 5998}
	if !isUpstreamPeer(peer, up) {
		t.Fatal("expected upstream peer match")
	}
	other := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
	if isUpstreamPeer(other, up) {
		t.Fatal("expected client peer mismatch")
	}
}

func TestRelayBidirectional(t *testing.T) {
	upstream, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()
	upAddr := upstream.LocalAddr().(*net.UDPAddr)

	srv := &Server{
		Listen:   "127.0.0.1:0",
		Upstream: upAddr.String(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	proxyAddr, err := net.ResolveUDPAddr("udp", srv.Listen)
	if err != nil {
		t.Fatal(err)
	}
	// Listen was "127.0.0.1:0" — resolve actual bound port from conn
	srv.mu.Lock()
	proxyAddr = srv.conn.LocalAddr().(*net.UDPAddr)
	srv.mu.Unlock()

	client, err := net.DialUDP("udp", nil, proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	req := []byte("hello-from-client")
	if _, err := client.Write(req); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 65535)
	_ = upstream.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, from, err := upstream.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("upstream read: %v", err)
	}
	if string(buf[:n]) != string(req) {
		t.Fatalf("upstream got %q", buf[:n])
	}

	resp := []byte("hello-from-upstream")
	if _, err := upstream.WriteToUDP(resp, from); err != nil {
		t.Fatal(err)
	}

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	n2, err := client.Read(buf)
	if err != nil {
		t.Fatalf("client read: %v", err)
	}
	if string(buf[:n2]) != string(resp) {
		t.Fatalf("client got %q", buf[:n2])
	}
}

func TestRelayMultipleUpstreamPackets(t *testing.T) {
	upstream, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()

	srv := &Server{
		Listen:   "127.0.0.1:0",
		Upstream: upstream.LocalAddr().String(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	srv.mu.Lock()
	proxyAddr := srv.conn.LocalAddr().(*net.UDPAddr)
	srv.mu.Unlock()

	client, err := net.DialUDP("udp", nil, proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 65535)
	_ = upstream.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, from, err := upstream.ReadFromUDP(buf)
	if err != nil {
		t.Fatal(err)
	}

	for i, msg := range [][]byte{[]byte("one"), []byte("two"), []byte("three")} {
		if _, err := upstream.WriteToUDP(msg, from); err != nil {
			t.Fatal(err)
		}
		_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := client.Read(buf)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if string(buf[:n]) != string(msg) {
			t.Fatalf("packet %d: got %q want %q", i, buf[:n], msg)
		}
	}
}

func loginPacket() []byte {
	base := []byte{
		0x00, 0x03,
		0x04, 0x00, 0x15, 0x00, 0x00,
		0x20, 0x00, 0x09, 0x00, 0x01,
		0x02, 0x00,
		0x03, 0x00, 0x00, 0x00,
		0x00,
		0x02,
		0x00, 0x00, 0x00, 0x00,
	}
	return append(base, protocol.GoldenBytes()...)
}

func TestExtractUsernameHint(t *testing.T) {
	hint := extractUsernameHint(loginPacket())
	if hint != "user" {
		t.Fatalf("got %q", hint)
	}
	if extractUsernameHint([]byte{1, 2, 3}) != "" {
		t.Fatal("expected empty for short packet")
	}
	if extractUsernameHint([]byte{0x00, 0x01}) != "" {
		t.Fatal("expected empty for non-combined")
	}
	badLen := loginPacket()
	badLen = badLen[:len(badLen)-1]
	if extractUsernameHint(badLen) != "" {
		t.Fatal("expected empty for bad cipher len")
	}
	ct, err := protocol.EncryptDES([]byte("ABCDEFGH"))
	if err != nil {
		t.Fatal(err)
	}
	base := []byte{
		0x00, 0x03,
		0x04, 0x00, 0x15, 0x00, 0x00,
		0x20, 0x00, 0x09, 0x00, 0x01,
		0x02, 0x00,
		0x03, 0x00, 0x00, 0x00,
		0x00,
		0x02,
		0x00, 0x00, 0x00, 0x00,
	}
	noNul := append(base, ct...)
	if extractUsernameHint(noNul) != "" {
		t.Fatal("expected empty when plaintext has no NUL")
	}
}
