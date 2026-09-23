package router

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/sso"
)

func TestHandleLoginPacketNil(t *testing.T) {
	r := &Router{Local: &localdata.Store{}}
	res := r.HandleLoginPacket(context.Background(), nil)
	if res.Decision != DecisionFail || res.Message == "" {
		t.Fatalf("%+v", res)
	}
}

func TestHandleLoginPacketSSODoesNotSplice(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	fake := &fakeSSO{
		connected: true,
		names:     map[string]bool{"user": true},
		result:    sso.LoginAuthResult{AccountID: 9},
	}
	r := &Router{Local: store, SSO: fake}
	login := testLoginPacket(t)
	orig := append([]byte{}, login.Buf...)
	res := r.HandleLoginPacket(context.Background(), login)
	if res.Decision != DecisionSSO {
		t.Fatalf("%+v", res)
	}
	if string(res.Packet) != string(orig) {
		t.Fatal("SSO path must forward the original alias packet")
	}
}

func TestWipeHelpers(t *testing.T) {
	res := sso.LoginAuthResult{RealUser: "u", CipherB64: "x"}
	wipeLoginAuthResult(&res)
	if res.RealUser != "" || res.CipherB64 != "" {
		t.Fatalf("%+v", res)
	}
}
