package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	githubReleasesURL = "https://api.github.com/repos/vmkteam/pgdesigner/releases/latest"
	releasesPageURL   = "https://github.com/vmkteam/pgdesigner/releases"
	updateCheckTTL    = 24 * time.Hour
	httpTimeout       = 10 * time.Second
)

// UpdateResult holds the result of an update check.
type UpdateResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
	ShouldNotify    bool
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate checks GitHub Releases for a newer version.
// It caches the result in config for 24 hours.
func CheckForUpdate(cfg *Config, currentVersion string) UpdateResult {
	result := UpdateResult{
		CurrentVersion: currentVersion,
		ReleaseURL:     releasesPageURL,
	}

	// Use cached result if fresh enough.
	if !cfg.LastUpdateCheck.IsZero() && time.Since(cfg.LastUpdateCheck) < updateCheckTTL && cfg.CachedLatestVersion != "" {
		result.LatestVersion = cfg.CachedLatestVersion
		result.UpdateAvailable = isNewer(currentVersion, cfg.CachedLatestVersion)
		result.ShouldNotify = result.UpdateAvailable && cfg.CachedLatestVersion != cfg.DismissedVersion
		return result
	}

	// Fetch from GitHub.
	latest, err := fetchLatestRelease()
	if err != nil || latest == "" {
		return result
	}

	// Update cache.
	cfg.CachedLatestVersion = latest
	cfg.LastUpdateCheck = time.Now()
	_ = cfg.Save()

	result.LatestVersion = latest
	result.UpdateAvailable = isNewer(currentVersion, latest)
	result.ShouldNotify = result.UpdateAvailable && latest != cfg.DismissedVersion
	return result
}

// DismissVersion records that the user has seen the notification for this version.
func DismissVersion(cfg *Config, version string) error {
	cfg.DismissedVersion = version
	return cfg.Save()
}

// fetchLatestRelease calls GitHub API and returns the latest tag name (e.g. "v0.2.0").
func fetchLatestRelease() (string, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest(http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "pgdesigner-update-check")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // no releases yet
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// isNewer returns true if latest > current using semver comparison.
func isNewer(current, latest string) bool {
	c := ensureVPrefix(current)
	l := ensureVPrefix(latest)
	if !semver.IsValid(c) || !semver.IsValid(l) {
		return false
	}
	return semver.Compare(l, c) > 0
}

func ensureVPrefix(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}
