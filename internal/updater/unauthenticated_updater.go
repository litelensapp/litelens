package updater

import (
	"context"
	"fmt"
	"net/http"
	goruntime "runtime"
	"strings"
	"time"
)

// assetFileName returns the exact release artifact filename for the given
// GOOS/GOARCH, matching scripts/install.sh's naming convention and the
// release matrix built by .github/workflows/job-build.yml.
func assetFileName(goos, goarch string) (string, error) {
	switch {
	case goos == "darwin" && goarch == "arm64":
		return "litelens-darwin-arm64.zip", nil
	case goos == "darwin" && goarch == "amd64":
		return "litelens-darwin-amd64.zip", nil
	case goos == "linux" && goarch == "amd64":
		return "litelens-linux-amd64.tar.gz", nil
	case goos == "windows" && goarch == "amd64":
		return "litelens-windows-amd64.exe", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
	}
}

// resolveLatestTagUnauthenticated resolves the latest release tag via
// GitHub's unauthenticated releases/latest redirect, which is not subject to
// the api.github.com rate limit. It follows the redirect (the default
// behavior of http.DefaultClient) and extracts the tag from the final URL's
// /releases/tag/{tag} path, mirroring `curl -w '%{url_effective}'` in
// scripts/install.sh.
func resolveLatestTagUnauthenticated() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getReleasesLatestUrl(), nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve latest release: HTTP %d", resp.StatusCode)
	}

	finalURL := resp.Request.URL.String()
	_, tag, found := strings.Cut(finalURL, "/releases/tag/")
	if !found {
		return "", fmt.Errorf("resolve latest release: could not find tag in resolved URL %q", finalURL)
	}
	if tag == "" {
		return "", fmt.Errorf("resolve latest release: empty tag in resolved URL %q", finalURL)
	}
	return tag, nil
}

// downloadSizeUnauthenticated best-effort resolves the Content-Length of a
// direct download URL via a HEAD request. Returns 0 on any failure; the size
// is cosmetic (UpdateModal hides the row when it's 0/empty), never a hard
// requirement.
func downloadSizeUnauthenticated(assetURL string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, assetURL, nil)
	if err != nil {
		return 0
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	return resp.ContentLength
}

// unauthenticatedRelease builds a Release for the given tag using only
// public github.com URLs (no api.github.com calls), matching the unauthenticated
// path in scripts/install.sh.
func unauthenticatedRelease(tag string) (*Release, error) {
	name, err := assetFileName(goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		return nil, err
	}

	assetURL := getUnauthenticatedReleaseAssetUrl(tag, name)
	return &Release{
		TagName:      tag,
		HTMLURL:      getUnauthenticatedReleaseTagsUrl(tag),
		AssetURL:     assetURL,
		DownloadSize: downloadSizeUnauthenticated(assetURL),
	}, nil
}
