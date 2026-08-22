package router

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/protocol"
	"github.com/alfred-identity/app/internal/sso"
)

type Decision string

const (
	DecisionLocal       Decision = "local"
	DecisionSSO         Decision = "sso"
	DecisionPassthrough Decision = "passthrough"
	DecisionFail        Decision = "fail"
)

type Result struct {
	Decision Decision
	Packet   []byte
	Message  string
}

// SSOAuth is the subset of SSO client used for login rewriting.
type SSOAuth interface {
	Connected() bool
	NameInMetadata(name string) bool
	LoginAuthWithRetry(ctx context.Context, requestID, username string) (sso.LoginAuthResult, error)
}

// Router implements SSO (on-the-fly) → local CSV → passthrough per plan.
type Router struct {
	Local  *localdata.Store
	SSO    SSOAuth
	Log    *slog.Logger
	BusyFn func() map[string]bool // local account names busy
}

func (r *Router) HandleLoginPacket(ctx context.Context, login *protocol.LoginPacket) Result {
	if login == nil {
		return Result{Decision: DecisionFail, Message: "missing login packet"}
	}
	typedUser := login.Username

	busy := map[string]bool{}
	if r.BusyFn != nil {
		busy = r.BusyFn()
	}

	local := r.Local.ResolveLogin(typedUser, busy)

	if r.SSO != nil && r.SSO.Connected() && r.SSO.NameInMetadata(typedUser) {
		return r.loginViaSSO(ctx, login, typedUser, local)
	}

	if local.Matched && local.Chosen != nil {
		out, err := login.RewriteCredentials(local.Chosen.Name, local.Chosen.Password)
		if err != nil {
			return Result{Decision: DecisionFail, Message: err.Error()}
		}
		return Result{Decision: DecisionLocal, Packet: out}
	}
	if local.Matched && local.AllBusy && !local.ViaAlias {
		return Result{Decision: DecisionFail, Message: "local account busy"}
	}
	if local.Matched && local.AllBusy && local.ViaAlias {
		return Result{Decision: DecisionFail, Message: "local alias busy; not found on SSO"}
	}

	return Result{Decision: DecisionPassthrough, Packet: login.Buf}
}

func (r *Router) loginViaSSO(ctx context.Context, login *protocol.LoginPacket, typedUser string, local localdata.ResolveResult) Result {
	res, err := r.SSO.LoginAuthWithRetry(ctx, uuid.NewString(), typedUser)
	defer wipeLoginAuthResult(&res)
	if err != nil {
		return Result{Decision: DecisionFail, Message: fmt.Sprintf("sso login_auth: %v", err)}
	}
	if res.Error != "" {
		if local.Matched && local.AllBusy && local.ViaAlias && res.Error == "not_found" {
			return Result{Decision: DecisionFail, Message: "local alias busy; not found on SSO"}
		}
		return Result{Decision: DecisionFail, Message: "sso: " + res.Error}
	}
	cipher, err := base64.StdEncoding.DecodeString(res.CipherB64)
	if err != nil {
		return Result{Decision: DecisionFail, Message: "bad cipher"}
	}
	defer wipeBytes(cipher)

	out, err := login.SpliceEncryptedCredentials(cipher)
	if err != nil {
		return Result{Decision: DecisionFail, Message: err.Error()}
	}
	return Result{Decision: DecisionSSO, Packet: out}
}

func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func wipeLoginAuthResult(res *sso.LoginAuthResult) {
	res.RealUser = ""
	res.CipherB64 = ""
}
