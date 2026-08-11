package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/internal/lib/ratelimiter"
	"golang.org/x/mod/semver"
)

// checkAuthenticated queries the api.github.com REST API for the latest
// release using a Bearer token, mirroring Check's contract: returns
// (nil, nil) if current is already up-to-date, and populates AssetURL/
// DownloadSize from the matching platform asset when a newer release exists.
func checkAuthenticated(current, token string) (*Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := getReleasesLatestUrl()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, ratelimiter.BuildError(resp)
		}
		return nil, fmt.Errorf("fetch latest release: HTTP %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	latest := rel.TagName
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	if !semver.IsValid(latest) || semver.Compare(latest, current) <= 0 {
		return nil, nil
	}

	if a := platformAsset(rel.Assets); a != nil {
		if config.IsPrivateRepoAccess() {
			rel.AssetURL = a.URL
		} else {
			rel.AssetURL = a.BrowserDownloadURL
		}
		rel.DownloadSize = a.Size
	}

	return &rel, nil
}

// fetchReleaseAuthenticated returns the named release from the api.github.com
// REST API using a Bearer token, with its platform asset resolved.
func fetchReleaseAuthenticated(tag, token string) (*Release, error) {
	url := getAuthenticatedReleaseTagsUrl(tag)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, ratelimiter.BuildError(resp)
		}
		return nil, fmt.Errorf("github API returned HTTP %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	if a := platformAsset(rel.Assets); a != nil {
		if config.IsPrivateRepoAccess() {
			rel.AssetURL = a.URL
		} else {
			rel.AssetURL = a.BrowserDownloadURL
		}
		rel.DownloadSize = a.Size
	}
	return &rel, nil
}

// platformAsset returns the release asset matching the running GOOS/GOARCH,
// skipping checksum (.sha256) assets so they never shadow the real artifact.
func platformAsset(assets []Asset) *Asset {
	os := goruntime.GOOS
	arch := goruntime.GOARCH

	var osPart, archPart string
	switch os {
	case "darwin":
		osPart = "darwin"
	case "linux":
		osPart = "linux"
	case "windows":
		osPart = "windows"
	default:
		return nil
	}
	switch arch {
	case "amd64":
		archPart = "amd64"
	case "arm64":
		archPart = "arm64"
	default:
		return nil
	}

	// Checksum assets are named after the artifact they cover (e.g.
	// "litelens-darwin-arm64.zip.sha256"), so they also contain the os-arch
	// suffix. Skip them or they can shadow the real artifact and cause both
	// the reported download size and the actual update download to point at
	// the tiny checksum file instead of the app itself.
	suffix := fmt.Sprintf("%s-%s", osPart, archPart)
	for i := range assets {
		if strings.HasSuffix(assets[i].Name, ".sha256") {
			continue
		}
		if strings.Contains(assets[i].Name, suffix) {
			return &assets[i]
		}
	}
	return nil
}
