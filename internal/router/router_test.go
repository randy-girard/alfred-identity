package router

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/protocol"
)

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

func TestHandleLoginPacketLocal(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := store.UpsertAccount("eqbox", "secret", nil); err != nil {
		t.Fatal(err)
	}
	r := &Router{Local: store}
	res := r.HandleLoginPacket(context.Background(), loginPacket(), "eqbox")
	if res.Decision != DecisionLocal || len(res.Packet) == 0 {
		t.Fatalf("%+v", res)
	}
}

func TestHandleLoginPacketBusy(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	_ = store.UpsertAccount("eqbox", "secret", nil)
	r := &Router{
		Local: store,
		BusyFn: func() map[string]bool {
			return map[string]bool{"eqbox": true}
		},
	}
	res := r.HandleLoginPacket(context.Background(), loginPacket(), "eqbox")
	if res.Decision != DecisionFail || res.Message == "" {
		t.Fatalf("%+v", res)
	}
}

func TestHandleLoginPacketPassthrough(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	r := &Router{Local: store}
	pkt := loginPacket()
	res := r.HandleLoginPacket(context.Background(), pkt, "unknown")
	if res.Decision != DecisionPassthrough {
		t.Fatalf("%+v", res)
	}
}
