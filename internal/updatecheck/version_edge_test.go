package updatecheck

import "testing"

func TestCompareVersionsPreRelease(t *testing.T) {
	if compareVersions("1.0.0-rc.1", "1.0.0-rc.2") >= 0 {
		t.Fatal("rc.1 should be older than rc.2")
	}
	if compareVersions("1.0.0", "1.0.0-rc.99") <= 0 {
		t.Fatal("release should beat rc")
	}
	if compareVersions("1.0.0-alpha", "1.0.0-beta") >= 0 {
		t.Fatal("alpha < beta")
	}
	if compareVersions("1.0.0-beta.11", "1.0.0-beta.2") <= 0 {
		t.Fatal("numeric pre-release segment")
	}
}

func TestParseVersionInvalid(t *testing.T) {
	bad := []string{"", "v", "1.2.3.4", "1.2-", "1.x.0", "-beta"}
	for _, s := range bad {
		if _, ok := parseVersion(s); ok {
			t.Fatalf("expected invalid: %q", s)
		}
	}
	if _, ok := parseVersion("1.2.3"); !ok {
		t.Fatal("expected valid 1.2.3")
	}
}

func TestCleanupStaleUpdateFilesNoPanic(t *testing.T) {
	CleanupStaleUpdateFiles()
}
