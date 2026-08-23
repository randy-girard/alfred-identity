package app

import (
	"testing"
	"time"

	"github.com/alfred-identity/app/internal/sso"
)

func TestGetLocalAccountsMarksSharedInUse(t *testing.T) {
	a, _ := testAppWithConfig(t)
	if err := a.SaveLocalAccount("tank", "secret", nil); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	c := sso.NewClient()
	c.SetStateForTest(sso.TestClientState{
		Connected: true,
		UserID:    5,
		Accounts: []sso.AccountMeta{{
			ID: 10, Username: "tank", Restricted: true, OwnerUserID: 5,
			SharedUserIDs: []int64{9},
		}},
		ShareActivity: sso.ShareActivity{
			Online: []sso.ShareOnlineEntry{{
				AccountID: 10, AccountUsername: "tank", ActorIsOwner: false,
				UserDisplayName: "Other", UserDiscordID: "d2",
			}},
			Logins: []sso.ShareLoginEntry{{
				AccountID: 10, ActorIsOwner: false, ActorName: "Bob",
				ActorDiscordID: "d3", CreatedAt: now,
			}},
		},
	})
	a.sso = c

	accts := a.GetLocalAccounts()
	if len(accts) != 1 {
		t.Fatalf("%+v", accts)
	}
	got := accts[0]
	if !got.Shared || got.SharedSSOAcct != 10 || len(got.SharedUserIDs) != 1 {
		t.Fatalf("shared: %+v", got)
	}
	if !got.InUse || !got.InUseOther || got.InUseBy != "Other" {
		t.Fatalf("in use: %+v", got)
	}
	if !got.LastLoginOther || got.LastLoginBy != "Bob" {
		t.Fatalf("login: %+v", got)
	}
}

func TestShareLocalAccountValidation(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	if err := a.ShareLocalAccount("tank", nil, nil, nil); err == nil {
		t.Fatal("expected not connected")
	}
	c := sso.NewClient()
	c.SetStateForTest(sso.TestClientState{Connected: true})
	a.sso = c
	if err := a.ShareLocalAccount("", nil, nil, nil); err == nil {
		t.Fatal("expected account required")
	}
	if err := a.ShareLocalAccount("missing", nil, nil, nil); err == nil {
		t.Fatal("expected not found")
	}
}

func TestUnshareLocalAccountValidation(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.sso = sso.NewClient()
	if err := a.UnshareLocalAccount("tank"); err == nil {
		t.Fatal("expected not connected")
	}
	c := sso.NewClient()
	c.SetStateForTest(sso.TestClientState{Connected: true})
	a.sso = c
	if err := a.UnshareLocalAccount(""); err == nil {
		t.Fatal("expected account required")
	}
}

func TestGetLocalAccountsOwnerLoginMetadata(t *testing.T) {
	a, _ := testAppWithConfig(t)
	if err := a.SaveLocalAccount("tank", "secret", nil); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c := sso.NewClient()
	c.SetStateForTest(sso.TestClientState{
		Connected: true,
		UserID:    5,
		Accounts: []sso.AccountMeta{{
			ID: 10, Username: "tank", Restricted: true, OwnerUserID: 5,
		}},
		ShareActivity: sso.ShareActivity{
			Online: []sso.ShareOnlineEntry{{
				AccountID: 10, AccountUsername: "tank", ActorIsOwner: true,
				UserDisplayName: "Me",
			}},
			Logins: []sso.ShareLoginEntry{{
				AccountID: 10, ActorIsOwner: true, ActorName: "Me", CreatedAt: now,
			}},
		},
	})
	a.sso = c

	got := a.GetLocalAccounts()[0]
	if !got.InUse || got.InUseOther || got.InUseBy != "Me (you)" {
		t.Fatalf("in use: %+v", got)
	}
	if got.LastLoginOther || got.LastLoginBy != "Me (you)" {
		t.Fatalf("login: %+v", got)
	}
}
