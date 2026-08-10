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
	"golang.org/x/mod/semver"
)

type Asset struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Release struct {
	TagName      string  `json:"tag_name"`
	HTMLURL      string  `json:"html_url"`
	Body         string  `json:"body"`
	Assets       []Asset `json:"assets"`
	AssetURL     string  `json:"-"`
	DownloadSize int64   `json:"-"`
}

// Check returns the latest release if it is newer than current, or (nil, nil)
// if current is up-to-date, a dev build, or no update is needed.
// Returns (nil, error) if a transient failure occurs (network error, rate limit, etc.).
// token is optional; when non-empty it is sent as a Bearer header so that
// private repositories can be queried.
func Check(current, token string) (*Release, error) {
	if current == "dev" || !semver.IsValid(current) {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := config.GetReleasesBaseURL() + "/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

// FetchRelease returns the named release (adding a leading "v" if the tag
// omits one) with its platform asset resolved, regardless of whether it is
// newer than any particular current version. Used when the caller already
// knows which version to install.
func FetchRelease(tag, token string) (*Release, error) {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	url := fmt.Sprintf("%s/tags/%s", config.GetReleasesBaseURL(), tag)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
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
