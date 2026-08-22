package router

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/protocol"
	"github.com/alfred-identity/app/internal/sso"
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

type fakeSSO struct {
	connected bool
	names     map[string]bool
	result    sso.LoginAuthResult
	err       error
	calls     int
	lastUser  string
}

func (f *fakeSSO) Connected() bool { return f.connected }
func (f *fakeSSO) NameInMetadata(name string) bool {
	return f.names[name]
}
func (f *fakeSSO) LoginAuthWithRetry(ctx context.Context, requestID, username string) (sso.LoginAuthResult, error) {
	f.calls++
	f.lastUser = username
	return f.result, f.err
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

func TestHandleLoginPacketSSO(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	ct := protocol.GoldenBytes()
	fake := &fakeSSO{
		connected: true,
		names:     map[string]bool{"guildbox": true},
		result: sso.LoginAuthResult{
			RealUser:  "guildbox",
			CipherB64: base64.StdEncoding.EncodeToString(ct),
			AccountID: 9,
		},
	}
	r := &Router{Local: store, SSO: fake}
	res := r.HandleLoginPacket(context.Background(), loginPacket(), "guildbox")
	if res.Decision != DecisionSSO || len(res.Packet) == 0 {
		t.Fatalf("%+v", res)
	}
	if fake.calls != 1 || fake.lastUser != "guildbox" {
		t.Fatalf("login_auth calls=%d user=%q", fake.calls, fake.lastUser)
	}
}

func TestHandleLoginPacketSSOPrefersDaemonOverLocalCSV(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	if err := store.UpsertAccount("guildbox", "local-secret", nil); err != nil {
		t.Fatal(err)
	}
	ct := protocol.GoldenBytes()
	fake := &fakeSSO{
		connected: true,
		names:     map[string]bool{"guildbox": true},
		result: sso.LoginAuthResult{
			CipherB64: base64.StdEncoding.EncodeToString(ct),
		},
	}
	r := &Router{Local: store, SSO: fake}
	res := r.HandleLoginPacket(context.Background(), loginPacket(), "guildbox")
	if res.Decision != DecisionSSO {
		t.Fatalf("expected SSO, got %+v", res)
	}
	if fake.calls != 1 {
		t.Fatalf("expected login_auth, calls=%d", fake.calls)
	}
}

func TestHandleLoginPacketAliasBusySSONotFound(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	_ = store.UpsertAccount("a1", "p", []string{"shared"})
	_ = store.UpsertAccount("a2", "p", []string{"shared"})
	fake := &fakeSSO{
		connected: true,
		names:     map[string]bool{"shared": true},
		result:    sso.LoginAuthResult{Error: "not_found"},
	}
	r := &Router{
		Local: store,
		SSO:   fake,
		BusyFn: func() map[string]bool {
			return map[string]bool{"a1": true, "a2": true}
		},
	}
	res := r.HandleLoginPacket(context.Background(), loginPacket(), "shared")
	if res.Decision != DecisionFail || res.Message != "local alias busy; not found on SSO" {
		t.Fatalf("%+v", res)
	}
}

func TestHandleLoginPacketLocalBusyAndSSOErrors(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	_ = store.UpsertAccount("solo", "p", nil)
	r := &Router{
		Local: store,
		BusyFn: func() map[string]bool {
			return map[string]bool{"solo": true}
		},
	}
	res := r.HandleLoginPacket(context.Background(), loginPacket(), "solo")
	if res.Decision != DecisionFail || res.Message != "local account busy" {
		t.Fatalf("busy: %+v", res)
	}

	fake := &fakeSSO{
		connected: true,
		names:     map[string]bool{"guild": true},
		err:       context.DeadlineExceeded,
	}
	r = &Router{Local: store, SSO: fake}
	res = r.HandleLoginPacket(context.Background(), loginPacket(), "guild")
	if res.Decision != DecisionFail || res.Message == "" {
		t.Fatalf("sso err: %+v", res)
	}

	fake = &fakeSSO{
		connected: true,
		names:     map[string]bool{"guild": true},
		result:    sso.LoginAuthResult{Error: "denied"},
	}
	r = &Router{Local: store, SSO: fake}
	res = r.HandleLoginPacket(context.Background(), loginPacket(), "guild")
	if res.Decision != DecisionFail || res.Message != "sso: denied" {
		t.Fatalf("sso denied: %+v", res)
	}

	fake = &fakeSSO{
		connected: true,
		names:     map[string]bool{"guild": true},
		result:    sso.LoginAuthResult{CipherB64: "%%%"},
	}
	r = &Router{Local: store, SSO: fake}
	res = r.HandleLoginPacket(context.Background(), loginPacket(), "guild")
	if res.Decision != DecisionFail || res.Message != "bad cipher" {
		t.Fatalf("bad cipher: %+v", res)
	}

	_ = store.UpsertAccount("a1", "p", []string{"shared"})
	_ = store.UpsertAccount("a2", "p", []string{"shared"})
	r = &Router{
		Local: store,
		BusyFn: func() map[string]bool {
			return map[string]bool{"a1": true, "a2": true}
		},
	}
	res = r.HandleLoginPacket(context.Background(), loginPacket(), "shared")
	if res.Decision != DecisionFail || res.Message != "local alias busy; not found on SSO" {
		t.Fatalf("alias busy no sso: %+v", res)
	}
}
