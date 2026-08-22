package main

import "testing"

func TestIsDevVersion(t *testing.T) {
	cases := map[string]bool{
		"dev":           true,
		"dev-abc1234":   true,
		"dev-abc+dirty": true,
		"1.0.0":         false,
		"v1.0.0":        false,
	}
	for v, want := range cases {
		if got := isDevVersion(v); got != want {
			t.Fatalf("isDevVersion(%q) = %v, want %v", v, got, want)
		}
	}
}
