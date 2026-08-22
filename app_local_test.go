package main

import (
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/logwatch"
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
