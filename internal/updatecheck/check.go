package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Result struct {
	UpdateAvailable bool   `json:"update_available"`
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	ReleaseURL      string `json:"release_url"`
	AssetName       string `json:"asset_name,omitempty"`
	AssetURL        string `json:"asset_url,omitempty"`
	CanApply        bool   `json:"can_apply"`
}

// Overridable for tests.
var githubAPIBase = "https://api.github.com"

// SetAPIBaseForTest overrides the GitHub API base URL for tests. Returns a restore func.
func SetAPIBaseForTest(base string) func() {
	old := githubAPIBase
	githubAPIBase = base
	return func() { githubAPIBase = old }
}

// Check compares current version to the latest GitHub Release tag for owner/repo
// and selects a downloadable asset for this platform when available.
func Check(ctx context.Context, ownerRepo, current string) (Result, error) {
	parts := strings.Split(ownerRepo, "/")
	if len(parts) != 2 {
		return Result{}, fmt.Errorf("github_repo must be owner/repo")
	}
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBase, parts[0], parts[1])
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "alfred-identity-updatecheck")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return Result{}, fmt.Errorf("no GitHub release found for %s", ownerRepo)
	}
	if res.StatusCode != 200 {
		return Result{}, fmt.Errorf("github api %d", res.StatusCode)
	}
	var body struct {
		TagName string  `json:"tag_name"`
		HTMLURL string  `json:"html_url"`
		Assets  []Asset `json:"assets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Result{}, err
	}
	latest := strings.TrimPrefix(body.TagName, "v")
	cur := strings.TrimPrefix(current, "v")
	out := Result{
		UpdateAvailable: latest != "" && latest != cur,
		Current:         current,
		Latest:          body.TagName,
		ReleaseURL:      body.HTMLURL,
	}
	if asset, err := PickAsset(body.Assets); err == nil {
		out.AssetName = asset.Name
		out.AssetURL = asset.BrowserDownloadURL
		out.CanApply = out.UpdateAvailable && asset.BrowserDownloadURL != ""
	}
	return out, nil
}
