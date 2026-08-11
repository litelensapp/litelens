package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	goruntime "runtime"
	"strings"
	"time"
)

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

// fetchManifest fetches the manifest.json release asset for the given tag,
// providing the single source of truth for asset names, checksums, and sizes
// from the actual build matrix output.
func fetchManifest(tag string) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getManifestUrl(tag), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest: HTTP %d", resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	return &manifest, nil
}

// unauthenticatedRelease builds a Release for the given tag using only
// public github.com URLs (no api.github.com calls), matching the unauthenticated
// path in scripts/install.sh. It queries the manifest.json release asset to
// resolve the correct artifact filename and size for the current platform.
func unauthenticatedRelease(tag string) (*Release, error) {
	manifest, err := fetchManifest(tag)
	if err != nil {
		return nil, err
	}

	artifact := manifest.FindArtifact(goruntime.GOOS, goruntime.GOARCH)
	if artifact == nil {
		return nil, fmt.Errorf("current platform %s/%s not found in release manifest for %s", goruntime.GOOS, goruntime.GOARCH, tag)
	}

	assetURL := getUnauthenticatedReleaseAssetUrl(tag, artifact.Filename)
	return &Release{
		TagName:      tag,
		HTMLURL:      getUnauthenticatedReleaseTagsUrl(tag),
		AssetURL:     assetURL,
		DownloadSize: artifact.Size,
		SHA256:       artifact.SHA256,
	}, nil
}
