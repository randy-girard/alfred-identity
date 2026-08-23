package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Result struct {
	UpdateAvailable bool   `json:"update_available"`
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	ReleaseURL      string `json:"release_url"`
}

// Overridable for tests.
var githubAPIBase = "https://api.github.com"

// SetAPIBaseForTest overrides the GitHub API base URL for tests. Returns a restore func.
func SetAPIBaseForTest(base string) func() {
	old := githubAPIBase
	githubAPIBase = base
	return func() { githubAPIBase = old }
}

// Check compares current version to the latest GitHub Release tag for owner/repo.
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
	client := &http.Client{Timeout: 10 * time.Second}
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
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Result{}, err
	}
	latest := strings.TrimPrefix(body.TagName, "v")
	cur := strings.TrimPrefix(current, "v")
	return Result{
		UpdateAvailable: latest != "" && latest != cur,
		Current:         current,
		Latest:          body.TagName,
		ReleaseURL:      body.HTMLURL,
	}, nil
}
