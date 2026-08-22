package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alfred-identity/app/internal/sources"
)

func TestGetEqHostStateInvalidInstall(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	mgr, err := sources.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eqDir := filepath.Join(dir, "EverQuest")
	if err := os.MkdirAll(eqDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Update(func(c *sources.Config) { c.EQDirectory = eqDir }); err != nil {
		t.Fatal(err)
	}
	badFile := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(badFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Update(func(c *sources.Config) { c.EQDirectory = badFile }); err != nil {
		t.Fatal(err)
	}
	a := &App{cfg: mgr}
	if _, err := a.GetEqHostState(); err == nil {
		t.Fatal("expected invalid install error")
	}
}

func TestGetEqHostStateAndSave(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	mgr, err := sources.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eqDir := filepath.Join(dir, "EverQuest")
	if err := os.MkdirAll(eqDir, 0o755); err != nil {
		t.Fatal(err)
	}
	orig := "[LoginServer]\nHost=login.example:5998\n"
	if err := os.WriteFile(filepath.Join(eqDir, "eqhost.txt"), []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Update(func(c *sources.Config) { c.EQDirectory = eqDir }); err != nil {
		t.Fatal(err)
	}
	a := &App{cfg: mgr}
	st, err := a.GetEqHostState()
	if err != nil {
		t.Fatal(err)
	}
	if st.Current != orig || st.HasBackup {
		t.Fatalf("%+v", st)
	}
	if err := a.SaveEqHostContent("[LoginServer]\nHost=127.0.0.1:6998\n"); err != nil {
		t.Fatal(err)
	}
	st, err = a.GetEqHostState()
	if err != nil {
		t.Fatal(err)
	}
	if st.Current != "[LoginServer]\nHost=127.0.0.1:6998\n" {
		t.Fatalf("%q", st.Current)
	}
	if !st.HasBackup || st.Backup != orig {
		t.Fatalf("backup: %+v", st)
	}
	if err := a.RestoreEqHostBackup(); err != nil {
		t.Fatal(err)
	}
	st, err = a.GetEqHostState()
	if err != nil {
		t.Fatal(err)
	}
	if st.Current != orig {
		t.Fatalf("restored %q", st.Current)
	}
}

func TestGetEqHostStateEmptyDir(t *testing.T) {
	a := &App{}
	st, err := a.GetEqHostState()
	if err != nil || st != (EqHostState{}) {
		t.Fatalf("%+v err=%v", st, err)
	}
}
