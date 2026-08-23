package updatecheck

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Apply downloads the release zip, replaces the running install, clears macOS
// quarantine when needed, and schedules a relaunch after this process exits.
func Apply(ctx context.Context, assetURL string) error {
	assetURL = strings.TrimSpace(assetURL)
	if assetURL == "" {
		return fmt.Errorf("no download URL for this platform")
	}
	target, err := resolveInstallTarget()
	if err != nil {
		return err
	}

	tmpRoot, err := os.MkdirTemp("", "alfred-identity-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRoot)

	zipPath := filepath.Join(tmpRoot, "update.zip")
	if err := downloadFile(ctx, assetURL, zipPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	extractDir := filepath.Join(tmpRoot, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return err
	}
	if err := unzip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	payload, err := findPayload(extractDir)
	if err != nil {
		return err
	}
	if err := replaceInstall(target, payload); err != nil {
		return fmt.Errorf("replace: %w", err)
	}
	_ = clearQuarantine(target.Path)
	// Wait for this process to exit before starting the new one (single-instance lock).
	if err := scheduleRelaunch(os.Getpid(), target); err != nil {
		return fmt.Errorf("relaunch: %w", err)
	}
	return nil
}

type installTarget struct {
	// Path is the .app bundle (darwin) or the executable file (windows/linux).
	Path string
	Kind string // "app" or "file"
}

func resolveInstallTarget() (installTarget, error) {
	exe, err := os.Executable()
	if err != nil {
		return installTarget{}, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return installTarget{}, err
	}
	if runtime.GOOS == "darwin" {
		app, err := darwinAppBundle(exe)
		if err != nil {
			return installTarget{}, err
		}
		return installTarget{Path: app, Kind: "app"}, nil
	}
	return installTarget{Path: exe, Kind: "file"}, nil
}

func darwinAppBundle(exe string) (string, error) {
	// .../Name.app/Contents/MacOS/Name
	macos := filepath.Dir(exe)
	contents := filepath.Dir(macos)
	app := filepath.Dir(contents)
	if !strings.HasSuffix(strings.ToLower(app), ".app") {
		return "", fmt.Errorf("not running from an .app bundle (path %s); install from a release build to use in-app updates", exe)
	}
	if filepath.Base(macos) != "MacOS" || filepath.Base(contents) != "Contents" {
		return "", fmt.Errorf("unexpected .app layout at %s", exe)
	}
	return app, nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "alfred-identity-updater")
	client := &http.Client{
		Timeout: 10 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download HTTP %d", res.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, res.Body); err != nil {
		return err
	}
	return f.Close()
}

func unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if err := extractZipFile(f, destAbs); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, destAbs string) error {
	name := filepath.Clean(f.Name)
	if name == "." || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("invalid zip entry %q", f.Name)
	}
	target := filepath.Join(destAbs, name)
	rel, err := filepath.Rel(destAbs, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("zip slip blocked: %q", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func findPayload(extractDir string) (string, error) {
	var appPath, exePath, binPath string
	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := info.Name()
		lower := strings.ToLower(base)
		if info.IsDir() {
			if strings.HasSuffix(lower, ".app") {
				appPath = path
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(lower, ".exe") {
			exePath = path
			return nil
		}
		if !info.IsDir() && (info.Mode()&0o111) != 0 && binPath == "" {
			binPath = path
		}
		// Linux release zip may not preserve +x; prefer known binary name.
		if !info.IsDir() && (base == "Alfred Identity" || lower == "alfred-identity") {
			binPath = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		if appPath == "" {
			return "", fmt.Errorf("release zip has no .app bundle")
		}
		return appPath, nil
	case "windows":
		if exePath == "" {
			return "", fmt.Errorf("release zip has no .exe")
		}
		return exePath, nil
	default:
		if binPath == "" {
			return "", fmt.Errorf("release zip has no application binary")
		}
		return binPath, nil
	}
}

func replaceInstall(target installTarget, payload string) error {
	switch target.Kind {
	case "app":
		return replaceDir(target.Path, payload)
	case "file":
		return replaceFile(target.Path, payload)
	default:
		return fmt.Errorf("unknown install kind %q", target.Kind)
	}
}

func replaceDir(destApp, srcApp string) error {
	parent := filepath.Dir(destApp)
	base := filepath.Base(destApp)
	backup := filepath.Join(parent, base+".bak")
	staging := filepath.Join(parent, base+".new")
	_ = os.RemoveAll(staging)
	_ = os.RemoveAll(backup)

	if err := copyTree(srcApp, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(destApp, backup); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(staging, destApp); err != nil {
		_ = os.Rename(backup, destApp) // best-effort rollback
		_ = os.RemoveAll(staging)
		return err
	}
	_ = os.RemoveAll(backup)
	return nil
}

func replaceFile(destExe, srcExe string) error {
	backup := destExe + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(destExe, backup); err != nil {
		return fmt.Errorf("could not move running binary aside: %w", err)
	}
	if err := copyFile(srcExe, destExe); err != nil {
		_ = os.Rename(backup, destExe)
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(destExe, 0o755)
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(out, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, out)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return copyFile(path, out)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func clearQuarantine(path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	cmd := exec.Command("xattr", "-cr", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xattr: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CleanupStaleUpdateFiles removes leftover .old binaries from a prior update.
func CleanupStaleUpdateFiles() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return
	}
	_ = os.Remove(exe + ".old")
	if runtime.GOOS == "darwin" {
		if app, err := darwinAppBundle(exe); err == nil {
			_ = os.RemoveAll(app + ".bak")
			_ = os.RemoveAll(app + ".new")
		}
	}
}
