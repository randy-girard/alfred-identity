package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/logwatch"
	"github.com/alfred-identity/app/internal/p99proxy"
)

func TestGetVersion(t *testing.T) {
	a := &App{}
	if a.GetVersion() != Version {
		t.Fatalf("got %q", a.GetVersion())
	}
}

func TestBusyLocalMapsOnlineCharacters(t *testing.T) {
	a, _ := testAppWithConfig(t)
	if len(a.busyLocal()) != 0 {
		t.Fatal("expected empty without watcher")
	}

	a.local.Characters = []localdata.Character{{Account: "BoxOne", Name: "Hero"}}
	w := logwatch.New(t.TempDir())
	w.SetOnlineForTest("Hero")
	a.watcher = w

	busy := a.busyLocal()
	if !busy["boxone"] {
		t.Fatalf("busy=%v", busy)
	}
}

func TestLocalAccountCRUD(t *testing.T) {
	a, _ := testAppWithConfig(t)

	if err := a.SaveLocalAccount("Tank", "secret", []string{"box"}); err != nil {
		t.Fatal(err)
	}
	accts := a.GetLocalAccounts()
	if len(accts) != 1 || accts[0].Name != "Tank" || !accts[0].HasPass {
		t.Fatalf("%+v", accts)
	}
	if accts[0].Password != "" {
		t.Fatal("list DTO must omit password")
	}
	pw, err := a.GetLocalAccountPassword("Tank")
	if err != nil || pw != "secret" {
		t.Fatalf("password=%q err=%v", pw, err)
	}

	chars := a.GetLocalCharacters()
	if len(chars) != 0 {
		t.Fatalf("expected no characters yet: %+v", chars)
	}
	if err := a.SaveLocalCharacter("Tank", "Main"); err != nil {
		t.Fatal(err)
	}
	chars = a.GetLocalCharacters()
	if len(chars) != 1 || chars[0].Name != "Main" {
		t.Fatalf("%+v", chars)
	}

	a.ctx = nil // confirmDialog auto-confirms without Wails runtime
	if err := a.DeleteLocalCharacter("Main"); err != nil {
		t.Fatal(err)
	}
	if len(a.GetLocalCharacters()) != 0 {
		t.Fatal("character should be deleted")
	}
	if err := a.DeleteLocalAccount("Tank"); err != nil {
		t.Fatal(err)
	}
	if len(a.GetLocalAccounts()) != 0 {
		t.Fatal("account should be deleted")
	}
}

func TestGetLogsLimitClamping(t *testing.T) {
	a, _ := testAppWithConfig(t)
	for i := 0; i < 5; i++ {
		a.log.Info("line", "n", i)
	}
	if len(a.GetLogs(0)) < 5 {
		t.Fatal("expected default limit")
	}
	if len(a.GetLogs(5000)) > 2000 {
		t.Fatal("expected max 2000")
	}
}

func TestDeleteLocalAccountValidation(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.ctx = nil
	if err := a.DeleteLocalAccount(""); err == nil {
		t.Fatal("expected name required")
	}
}

func TestSaveLocalAccountNotReady(t *testing.T) {
	a := &App{}
	if err := a.SaveLocalAccount("x", "y", nil); err == nil {
		t.Fatal("expected not ready")
	}
}

func TestImportLocalAccountsFromProxyConfig(t *testing.T) {
	a, dir := testAppWithConfig(t)
	proxyDir := filepath.Join(dir, "P99 Login Proxy")
	if err := os.MkdirAll(proxyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(proxyDir, p99proxy.ConfigFileName)
	if err := os.WriteFile(cfg, []byte("[DEFAULT]\nlocal_accounts_file = accounts.csv\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proxyDir, "accounts.csv"), []byte("imported,pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := a.ImportLocalAccountsFromPath(cfg)
	if err != nil || res.Added != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	res, err = a.ImportLocalAccountsFromPath(proxyDir)
	if err != nil || res.Updated != 1 {
		t.Fatalf("dir import res=%+v err=%v", res, err)
	}
	if len(a.GetLocalAccounts()) != 1 || a.GetLocalAccounts()[0].Name != "imported" {
		t.Fatalf("%+v", a.GetLocalAccounts())
	}
}
