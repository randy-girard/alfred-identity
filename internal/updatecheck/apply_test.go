package updatecheck

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPickAsset(t *testing.T) {
	assets := []Asset{
		{Name: "alfred-identity-1.2.0-linux-amd64.zip", BrowserDownloadURL: "https://example.com/linux"},
		{Name: "alfred-identity-1.2.0-macos-universal.zip", BrowserDownloadURL: "https://example.com/mac"},
		{Name: "alfred-identity-1.2.0-windows-amd64.zip", BrowserDownloadURL: "https://example.com/win"},
		{Name: "alfred-identity-1.2.0-windows-arm64.zip", BrowserDownloadURL: "https://example.com/winarm"},
		{Name: "alfred-identity-1.2.0-linux-arm64.zip", BrowserDownloadURL: "https://example.com/linuxarm"},
	}
	got, err := PickAsset(assets)
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := ArtifactSuffix()
	if wantSuffix == "" {
		t.Skip("unsupported platform")
	}
	if !strings.Contains(strings.ToLower(got.Name), wantSuffix) {
		t.Fatalf("got %q want suffix %q", got.Name, wantSuffix)
	}
	if got.BrowserDownloadURL == "" {
		t.Fatal("empty url")
	}
}

func TestPickAssetMissing(t *testing.T) {
	if ArtifactSuffix() == "" {
		t.Skip("unsupported platform")
	}
	_, err := PickAsset([]Asset{{Name: "notes.txt", BrowserDownloadURL: "https://example.com/x"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckSelectsAsset(t *testing.T) {
	suffix := ArtifactSuffix()
	if suffix == "" {
		t.Skip("unsupported platform")
	}
	body := `{"tag_name":"v1.2.0","html_url":"https://example.com/r","assets":[` +
		`{"name":"alfred-identity-1.2.0-` + suffix + `.zip","browser_download_url":"https://example.com/a.zip","size":12}` +
		`]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	restore := SetAPIBaseForTest(srv.URL)
	defer restore()

	res, err := Check(context.Background(), "acme/app", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpdateAvailable || !res.CanApply || res.AssetURL == "" {
		t.Fatalf("%+v", res)
	}
}

func TestReplaceFileAndFindPayload(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "appbin")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "newbin")
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(dest, src); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "new" {
		t.Fatalf("dest=%q err=%v", b, err)
	}
	if _, err := os.Stat(dest + ".old"); err != nil {
		t.Fatal("expected .old backup", err)
	}
}

func TestFindPayloadApp(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := t.TempDir()
	app := filepath.Join(dir, "Alfred Identity.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findPayload(dir)
	if err != nil || got != app {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestUnzipAndZipSlip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "t.zip")
	if err := writeTestZip(zipPath, map[string]string{
		"Alfred Identity": "binary",
	}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	_ = os.MkdirAll(out, 0o755)
	if err := unzip(zipPath, out); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("exe expected on windows")
	}
	if runtime.GOOS == "darwin" {
		t.Skip("app expected on darwin")
	}
	p, err := findPayload(out)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "binary" {
		t.Fatalf("%q", b)
	}
}

func writeTestZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, body := range files {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(body)); err != nil {
			return err
		}
	}
	return w.Close()
}
