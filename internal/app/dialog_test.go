package app

import "testing"

func TestDialogConfirmed(t *testing.T) {
	cases := []struct {
		sel    string
		accept string
		want   bool
	}{
		{"Delete", "Delete", true},
		{"delete", "Delete", true},
		{"Remove", "Remove", true},
		{"Yes", "Delete", true},
		{"Ok", "Delete", true},
		{"OK", "Remove", true},
		{"Cancel", "Delete", false},
		{"No", "Delete", false},
		{"", "Delete", false},
		{"Revoke", "Revoke", true},
		{"Install", "Install", true},
		{"Open release", "Open release", true},
		{"Later", "Install", false},
	}
	for _, tc := range cases {
		if got := dialogConfirmed(tc.sel, tc.accept); got != tc.want {
			t.Fatalf("dialogConfirmed(%q, %q)=%v want %v", tc.sel, tc.accept, got, tc.want)
		}
	}
}

func TestDialogHelpersNilContext(t *testing.T) {
	a := &App{}
	a.infoDialog("t", "m")
	a.errorDialog("t", "m")
	ok, err := a.confirmDialog("t", "m", "OK")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("confirmDialog without ctx should auto-confirm")
	}
}
