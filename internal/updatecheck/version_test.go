package updatecheck

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		candidate, current string
		want               bool
	}{
		{"1.2.0", "1.0.0", true},
		{"v1.2.0", "1.2.0", false},
		{"1.2.0", "v1.2.0", false},
		{"1.1.0", "1.2.0", false}, // downgrade
		{"1.2.0", "1.10.0", false},
		{"1.10.0", "1.2.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.2.0", "1.2.0-beta.1", true},
		{"1.2.0-beta.1", "1.2.0", false},
		{"1.2.0-beta.2", "1.2.0-beta.1", true},
		{"1.2.0-alpha", "1.2.0-beta", false},
		{"1.2.0+dirty", "1.2.0", false},
		{"1.2.1+build", "1.2.0", true},
		{"", "1.0.0", false},
		{"1.0.0", "", false},
		{"dev", "1.0.0", false},
		{"1.0.0", "dev", false},
	}
	for _, tc := range cases {
		if got := IsNewer(tc.candidate, tc.current); got != tc.want {
			t.Fatalf("IsNewer(%q, %q) = %v, want %v", tc.candidate, tc.current, got, tc.want)
		}
	}
}
