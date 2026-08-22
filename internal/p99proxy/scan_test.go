package p99proxy

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverRecursiveFindsNestedConfig(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	nested := filepath.Join(home, "deep", "nested", "P99LoginProxy")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(nested, ConfigFileName)
	if err := os.WriteFile(cfg, []byte("[DEFAULT]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	found := DiscoverRecursive(home)
	if !installPathsContain(found, cfg) {
		t.Fatalf("expected %q in found=%+v", cfg, found)
	}
}

func TestShouldSkipScanDirSkipsHeavyPaths(t *testing.T) {
	if !shouldSkipScanDir("/tmp/proj/node_modules") {
		t.Fatal("expected node_modules skip")
	}
	if shouldSkipScanDir("/tmp/Documents/P99LoginProxy") {
		t.Fatal("should not skip normal install path")
	}
	if runtime.GOOS == "darwin" {
		if !shouldSkipScanDir("/Users/me/Library/Caches/Foo") {
			t.Fatal("expected Library/Caches skip on macOS")
		}
		if shouldSkipScanDir("/Users/me/Documents/P99LoginProxy") {
			t.Fatal("should not skip documents install path on macOS")
		}
	}
}
