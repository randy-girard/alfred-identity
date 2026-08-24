package app

import (
	"context"
	"testing"
)

func TestApplyUpdateRejectsDevVersion(t *testing.T) {
	old := Version
	Version = "dev"
	t.Cleanup(func() { Version = old })

	a := &App{}
	err := a.ApplyUpdate()
	if err == nil || err.Error() == "" {
		t.Fatalf("expected error for dev builds, got %v", err)
	}
}

func TestOnBeforeCloseAllowsQuitWhenExiting(t *testing.T) {
	a := &App{}
	if !a.OnBeforeClose(context.Background()) {
		t.Fatal("expected close to be cancelled when not quitting")
	}
	a.quitting.Store(true)
	if a.OnBeforeClose(context.Background()) {
		t.Fatal("expected quit to proceed when quitting flag is set")
	}
}
