package updatecheck

import (
	"fmt"
	"runtime"
	"strings"
)

// ArtifactSuffix is the release zip suffix for this OS/arch
// (matches .github/workflows/release.yml matrix.artifact).
func ArtifactSuffix() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos-universal"
	case "windows":
		if runtime.GOARCH == "arm64" {
			return "windows-arm64"
		}
		return "windows-amd64"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "linux-arm64"
		}
		return "linux-amd64"
	default:
		return ""
	}
}

// PickAsset chooses the release asset for this platform from a GitHub assets list.
func PickAsset(assets []Asset) (Asset, error) {
	suffix := ArtifactSuffix()
	if suffix == "" {
		return Asset{}, fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	needle := "-" + suffix + ".zip"
	var match Asset
	for _, a := range assets {
		name := strings.ToLower(strings.TrimSpace(a.Name))
		if !strings.HasSuffix(name, ".zip") {
			continue
		}
		if strings.Contains(name, needle) || strings.HasSuffix(name, needle) {
			if a.BrowserDownloadURL == "" {
				continue
			}
			match = a
			break
		}
	}
	if match.BrowserDownloadURL == "" {
		return Asset{}, fmt.Errorf("no release asset for %s", suffix)
	}
	return match, nil
}
