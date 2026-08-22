package router

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

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

// Router implements local → SSO → passthrough per plan.
type Router struct {
	Local  *localdata.Store
	SSO    SSOAuth
	Log    *slog.Logger
	BusyFn func() map[string]bool // local account names busy
}

func (r *Router) HandleLoginPacket(ctx context.Context, pkt []byte, typedUser string) Result {
	busy := map[string]bool{}
	if r.BusyFn != nil {
		busy = r.BusyFn()
	}

	local := r.Local.ResolveLogin(typedUser, busy)
	if local.Matched && local.Chosen != nil {
		out, err := protocol.RewriteLoginPacket(pkt, local.Chosen.Name, local.Chosen.Password)
		if err != nil {
			return Result{Decision: DecisionFail, Message: err.Error()}
		}
		return Result{Decision: DecisionLocal, Packet: out}
	}
	if local.Matched && local.AllBusy && !local.ViaAlias {
		return Result{Decision: DecisionFail, Message: "local account busy"}
	}
	// alias all busy → fall through to SSO

	if r.SSO != nil && r.SSO.Connected() && r.SSO.NameInMetadata(typedUser) {
		res, err := r.SSO.LoginAuthWithRetry(ctx, uuid.NewString(), typedUser)
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
		out, err := protocol.SpliceCipherBlob(pkt, cipher)
		res.CipherB64 = "" // best-effort wipe
		if err != nil {
			return Result{Decision: DecisionFail, Message: err.Error()}
		}
		return Result{Decision: DecisionSSO, Packet: out}
	}

	if local.Matched && local.AllBusy && local.ViaAlias {
		return Result{Decision: DecisionFail, Message: "local alias busy; not found on SSO"}
	}

	_ = strings.TrimSpace
	return Result{Decision: DecisionPassthrough, Packet: pkt}
}
