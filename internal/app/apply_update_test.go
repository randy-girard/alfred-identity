package app

import "testing"

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
