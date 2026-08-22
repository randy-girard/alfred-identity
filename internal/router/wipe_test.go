package router

import (
	"testing"

	"github.com/alfred-identity/app/internal/sso"
)

func TestWipeLoginAuthResult(t *testing.T) {
	res := sso.LoginAuthResult{
		RealUser:  "secret-user",
		CipherB64: "Ym9vbQ==",
		AccountID: 1,
	}
	wipeLoginAuthResult(&res)
	if res.RealUser != "" || res.CipherB64 != "" {
		t.Fatalf("expected wiped credentials, got %+v", res)
	}
	if res.AccountID != 1 {
		t.Fatalf("account id should remain: %d", res.AccountID)
	}
}
