package proxy

import (
	"net"
	"testing"
)

func TestEffectiveBindAddr(t *testing.T) {
	upExt, err := net.ResolveUDPAddr("udp", "70.35.159.39:5998")
	if err != nil {
		t.Fatal(err)
	}
	upLocal, err := net.ResolveUDPAddr("udp", "127.0.0.1:5998")
	if err != nil {
		t.Fatal(err)
	}

	got, changed := EffectiveBindAddr("127.0.0.1:6998", upExt)
	if !changed || got != "0.0.0.0:6998" {
		t.Fatalf("external upstream: got %q changed=%v", got, changed)
	}

	got, changed = EffectiveBindAddr("127.0.0.1:6998", upLocal)
	if changed || got != "127.0.0.1:6998" {
		t.Fatalf("local upstream: got %q changed=%v", got, changed)
	}

	got, changed = EffectiveBindAddr("0.0.0.0:6998", upExt)
	if changed || got != "0.0.0.0:6998" {
		t.Fatalf("already all interfaces: got %q changed=%v", got, changed)
	}

	got, changed = EffectiveBindAddr("localhost:6999", upExt)
	if !changed || got != "0.0.0.0:6999" {
		t.Fatalf("localhost alias: got %q changed=%v", got, changed)
	}
}
