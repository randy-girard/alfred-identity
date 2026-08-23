package sources

import "strings"

// legacyGitHubRepos are old github_repo values that should map to DefaultGitHubRepo.
var legacyGitHubRepos = map[string]bool{
	"alfred-identity/app":              true,
	"p99-identity/gui":                 true,
	"randy-girard/alfred-identity-gui": true,
}

// ResolveGitHubRepo maps legacy update-check repos to DefaultGitHubRepo.
func ResolveGitHubRepo(repo string) string {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return DefaultGitHubRepo
	}
	lower := strings.ToLower(repo)
	if legacyGitHubRepos[repo] || legacyGitHubRepos[lower] {
		return DefaultGitHubRepo
	}
	switch {
	case strings.HasPrefix(lower, "p99-identity/"),
		lower == "alfred-identity/app":
		return DefaultGitHubRepo
	}
	return repo
}

// migrateGitHubRepo fixes legacy placeholder repos used before public releases existed.
func migrateGitHubRepo(c *Config) bool {
	resolved := ResolveGitHubRepo(c.GitHubRepo)
	before := strings.TrimSpace(c.GitHubRepo)
	if resolved == before {
		return false
	}
	c.GitHubRepo = resolved
	return true
}

// NormalizeGitHubRepo rewrites legacy github_repo values and persists config when changed.
func (m *Manager) NormalizeGitHubRepo() error {
	m.mu.Lock()
	resolved := ResolveGitHubRepo(m.cfg.GitHubRepo)
	before := strings.TrimSpace(m.cfg.GitHubRepo)
	changed := resolved != before
	if changed {
		m.cfg.GitHubRepo = resolved
	}
	m.mu.Unlock()
	if !changed {
		return nil
	}
	return m.Save()
}
