package updater

import (
	"strings"

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
	// SHA256 is populated from manifest.json by both the authenticated and
	// unauthenticated paths, and is empty if the manifest was unavailable or
	// didn't list the current platform. Callers must treat empty as
	// "no integrity verification available" and fail closed.
	SHA256 string `json:"-"`
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

	if token != "" {
		return checkAuthenticated(current, token)
	}

	tag, err := resolveLatestTagUnauthenticated()
	if err != nil {
		return nil, err
	}
	latest := tag
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	if !semver.IsValid(latest) || semver.Compare(latest, current) <= 0 {
		return nil, nil
	}
	return unauthenticatedRelease(tag)
}

// FetchRelease returns the named release (adding a leading "v" if the tag
// omits one) with its platform asset resolved, regardless of whether it is
// newer than any particular current version. Used when the caller already
// knows which version to install.
func FetchRelease(tag, token string) (*Release, error) {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	if token != "" {
		return fetchReleaseAuthenticated(tag, token)
	}

	return unauthenticatedRelease(tag)

}
