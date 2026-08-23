package router

import (
	"context"
	"encoding/base64"
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

func TestHandleLoginPacketSSOSpliceError(t *testing.T) {
	dir := t.TempDir()
	store := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "a.csv"),
		CharactersPath: filepath.Join(dir, "c.csv"),
	}
	huge := make([]byte, 300)
	for i := range huge {
		huge[i] = 'A'
	}
	fake := &fakeSSO{
		connected: true,
		names:     map[string]bool{"guild": true},
		result:    sso.LoginAuthResult{CipherB64: base64.StdEncoding.EncodeToString(huge)},
	}
	r := &Router{Local: store, SSO: fake}
	login := testLoginPacket(t)
	login.Username = "guild"
	res := r.HandleLoginPacket(context.Background(), login)
	if res.Decision != DecisionFail || res.Message == "" {
		t.Fatalf("expected splice fail: %+v", res)
	}
}

func TestWipeHelpers(t *testing.T) {
	b := []byte("secret")
	wipeBytes(b)
	for _, c := range b {
		if c != 0 {
			t.Fatal("wipeBytes")
		}
	}
	res := sso.LoginAuthResult{RealUser: "u", CipherB64: "x"}
	wipeLoginAuthResult(&res)
	if res.RealUser != "" || res.CipherB64 != "" {
		t.Fatalf("%+v", res)
	}
}
