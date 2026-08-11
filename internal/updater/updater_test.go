package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/litelensapp/litelens/internal/lib/ratelimiter"
)

// TestCheck covers the primary happy-path and common-failure behavior of
// Check. Additional semver-validation boundary cases live in
// Test_Check_EdgeCases to keep this table focused.
func TestCheck(t *testing.T) {
	tests := []struct {
		name           string
		current        string
		serverResponse *Release
		serverStatus   int
		wantRelease    bool
	}{
		{
			name:        "dev build returns nil",
			current:     "dev",
			wantRelease: false,
		},
		{
			name:        "invalid semver returns nil",
			current:     "not-a-version",
			wantRelease: false,
		},
		{
			name:        "empty string returns nil",
			current:     "",
			wantRelease: false,
		},
		{
			name:    "server error returns nil",
			current: "v1.0.0",
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusInternalServerError,
			wantRelease:  false,
		},
		{
			name:    "non-200 status returns nil",
			current: "v1.0.0",
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusNotFound,
			wantRelease:  false,
		},
		{
			name:    "latest equal to current returns nil",
			current: "v1.0.0",
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusOK,
			wantRelease:  false,
		},
		{
			name:    "latest older than current returns nil",
			current: "v1.5.0",
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusOK,
			wantRelease:  false,
		},
		{
			name:    "newer release without v prefix is prefixed",
			current: "v1.0.0",
			serverResponse: &Release{
				TagName: "1.5.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusOK,
			wantRelease:  true,
		},
		{
			name:    "newer release with v prefix",
			current: "v1.0.0",
			serverResponse: &Release{
				TagName: "v2.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusOK,
			wantRelease:  true,
		},
		{
			name:    "newer release with platform asset",
			current: "v1.0.0",
			serverResponse: &Release{
				TagName: "v2.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets: []Asset{
					{
						Name:               fmt.Sprintf("litelens-%s-%s", goruntime.GOOS, goruntime.GOARCH),
						URL:                "https://api.github.com/repos/litelensapp/litelens/releases/assets/1",
						BrowserDownloadURL: "https://example.com/download",
						Size:               1024,
					},
				},
			},
			serverStatus: http.StatusOK,
			wantRelease:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/latest" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(tt.serverStatus)
				if tt.serverResponse != nil && tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

			got, err := Check(tt.current, "")
			if (got != nil) != tt.wantRelease {
				t.Errorf("Check(%q, \"\"): got release=%v, want release=%v", tt.current, got != nil, tt.wantRelease)
			}
			// For the primary happy-path test, we expect no error on success or no-update cases
			if tt.wantRelease && err != nil {
				t.Errorf("Check(%q, \"\"): got err=%v, want nil", tt.current, err)
			}

			if tt.wantRelease && got != nil {
				if got.TagName != tt.serverResponse.TagName && got.TagName != "v"+tt.serverResponse.TagName {
					t.Errorf("Check() TagName = %q, want %q or v%q", got.TagName, tt.serverResponse.TagName, tt.serverResponse.TagName)
				}
				if len(tt.serverResponse.Assets) > 0 && got.AssetURL == "" {
					t.Errorf("Check() expected AssetURL to be populated for platform asset")
				}
			}
		})
	}
}

func TestCheckPrivateRepoAccess(t *testing.T) {
	tests := []struct {
		name              string
		privateRepoAccess string
		wantAssetURL      string
	}{
		{
			name:              "private repo access enabled uses URL field",
			privateRepoAccess: "true",
			wantAssetURL:      "https://api.github.com/repos/test-owner/test-repo/releases/assets/1",
		},
		{
			name:              "private repo access disabled uses BrowserDownloadURL field",
			privateRepoAccess: "false",
			wantAssetURL:      "https://example.com/download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/latest" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(&Release{
					TagName: "v2.0.0",
					Body:    "Release notes",
					HTMLURL: "https://example.com",
					Assets: []Asset{
						{
							Name:               fmt.Sprintf("litelens-%s-%s", goruntime.GOOS, goruntime.GOARCH),
							URL:                "https://api.github.com/repos/test-owner/test-repo/releases/assets/1",
							BrowserDownloadURL: "https://example.com/download",
							Size:               1024,
						},
					},
				})
			}))
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)
			t.Setenv("PRIVATE_REPO_ACCESS", tt.privateRepoAccess)

			got, err := Check("v1.0.0", "")
			if err != nil {
				t.Fatalf("Check() returned err=%v, want nil", err)
			}
			if got == nil {
				t.Fatalf("Check() returned nil, want Release")
			}
			if got.AssetURL != tt.wantAssetURL {
				t.Errorf("Check() AssetURL = %q, want %q", got.AssetURL, tt.wantAssetURL)
			}
		})
	}
}

func TestCheckWithToken(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if gotAuth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&Release{
			TagName: "v2.0.0",
			Body:    "Release notes",
			HTMLURL: "https://example.com",
			Assets:  []Asset{},
		})
	}))
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	got, err := Check("v1.0.0", "test-token")
	if err != nil {
		t.Errorf("Check() returned err=%v, want nil", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Check() Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
	if got == nil {
		t.Errorf("Check with token should return release")
	}
	if got != nil && got.TagName != "v2.0.0" {
		t.Errorf("Check() TagName = %q, want v2.0.0", got.TagName)
	}
}

func TestFetchRelease(t *testing.T) {
	tests := []struct {
		name           string
		tag            string
		serverResponse *Release
		serverStatus   int
		wantErr        bool
	}{
		{
			name:    "tag without v prefix is prefixed",
			tag:     "1.0.0",
			wantErr: false,
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusOK,
		},
		{
			name:    "tag with v prefix",
			tag:     "v1.0.0",
			wantErr: false,
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusOK,
		},
		{
			name:    "server error returns error",
			tag:     "v1.0.0",
			wantErr: true,
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets:  []Asset{},
			},
			serverStatus: http.StatusInternalServerError,
		},
		{
			name:         "not found returns error",
			tag:          "v99.0.0",
			wantErr:      true,
			serverStatus: http.StatusNotFound,
		},
		{
			name:    "release with multiple assets",
			tag:     "v1.0.0",
			wantErr: false,
			serverResponse: &Release{
				TagName: "v1.0.0",
				Body:    "Release notes",
				HTMLURL: "https://example.com",
				Assets: []Asset{
					{
						Name:               "litelens-other-platform-decoy",
						URL:                "https://api.github.com/repos/litelensapp/litelens/releases/assets/1",
						BrowserDownloadURL: "https://example.com/decoy",
						Size:               2048,
					},
					{
						Name:               fmt.Sprintf("litelens-%s-%s", goruntime.GOOS, goruntime.GOARCH),
						URL:                "https://api.github.com/repos/litelensapp/litelens/releases/assets/2",
						BrowserDownloadURL: "https://example.com/match",
						Size:               1024,
					},
				},
			},
			serverStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedTag := tt.tag
				if len(tt.tag) > 0 && tt.tag[0] != 'v' {
					expectedTag = "v" + tt.tag
				}
				expectedPath := fmt.Sprintf("/tags/%s", expectedTag)
				if r.URL.Path != expectedPath {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(tt.serverStatus)
				if tt.serverResponse != nil && tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

			got, err := FetchRelease(tt.tag, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchRelease(%q, \"\"): got err=%v, wantErr=%v", tt.tag, err, tt.wantErr)
			}

			if !tt.wantErr && got == nil {
				t.Errorf("FetchRelease(%q, \"\"): got nil, want Release", tt.tag)
			}
			if !tt.wantErr && got != nil && tt.serverResponse != nil && len(tt.serverResponse.Assets) > 0 {
				if got.AssetURL == "" {
					t.Errorf("FetchRelease(%q, \"\") expected AssetURL to be populated", tt.tag)
				}
				if got.DownloadSize == 0 {
					t.Errorf("FetchRelease(%q, \"\") expected DownloadSize to be populated", tt.tag)
				}
			}
		})
	}
}

func TestFetchReleasePrivateRepoAccess(t *testing.T) {
	tests := []struct {
		name              string
		privateRepoAccess string
		wantAssetURL      string
	}{
		{
			name:              "private repo access enabled uses URL field",
			privateRepoAccess: "true",
			wantAssetURL:      "https://api.github.com/repos/test-owner/test-repo/releases/assets/1",
		},
		{
			name:              "private repo access disabled uses BrowserDownloadURL field",
			privateRepoAccess: "false",
			wantAssetURL:      "https://example.com/download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/tags/v1.0.0" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(&Release{
					TagName: "v1.0.0",
					Body:    "Release notes",
					HTMLURL: "https://example.com",
					Assets: []Asset{
						{
							Name:               fmt.Sprintf("litelens-%s-%s", goruntime.GOOS, goruntime.GOARCH),
							URL:                "https://api.github.com/repos/test-owner/test-repo/releases/assets/1",
							BrowserDownloadURL: "https://example.com/download",
							Size:               1024,
						},
					},
				})
			}))
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)
			t.Setenv("PRIVATE_REPO_ACCESS", tt.privateRepoAccess)

			got, err := FetchRelease("v1.0.0", "")
			if err != nil {
				t.Fatalf("FetchRelease() returned error: %v", err)
			}
			if got == nil {
				t.Fatalf("FetchRelease() returned nil, want Release")
			}
			if got.AssetURL != tt.wantAssetURL {
				t.Errorf("FetchRelease() AssetURL = %q, want %q", got.AssetURL, tt.wantAssetURL)
			}
		})
	}
}

func TestFetchReleaseWithToken(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if gotAuth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&Release{
			TagName: "v1.0.0",
			Body:    "Release notes",
			HTMLURL: "https://example.com",
			Assets:  []Asset{},
		})
	}))
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	got, err := FetchRelease("v1.0.0", "test-token")
	if gotAuth != "Bearer test-token" {
		t.Errorf("FetchRelease() Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
	if err != nil {
		t.Errorf("FetchRelease with token returned error: %v", err)
	}
	if got == nil {
		t.Errorf("FetchRelease with token should return release")
	}
}

func TestPlatformAsset(t *testing.T) {
	matchName := fmt.Sprintf("litelens-%s-%s", goruntime.GOOS, goruntime.GOARCH)
	middleMatchName := fmt.Sprintf("prefix-%s-%s-suffix", goruntime.GOOS, goruntime.GOARCH)
	tests := []struct {
		name      string
		assets    []Asset
		wantAsset *string
	}{
		{
			name: "finds asset with os-arch suffix",
			assets: []Asset{
				{Name: matchName, BrowserDownloadURL: "https://example.com/match"},
				{Name: "litelens-decoy-platform", BrowserDownloadURL: "https://example.com/decoy"},
			},
			wantAsset: &matchName,
		},
		{
			name:   "no asset for empty list",
			assets: []Asset{},
		},
		{
			name: "suffix match in middle of name",
			assets: []Asset{
				{Name: middleMatchName, BrowserDownloadURL: "https://example.com/asset"},
			},
			wantAsset: &middleMatchName,
		},
		{
			name: "skips sha256 checksum asset that shares the os-arch suffix",
			assets: []Asset{
				{Name: matchName + ".sha256", BrowserDownloadURL: "https://example.com/checksum", Size: 64},
				{Name: matchName, BrowserDownloadURL: "https://example.com/match", Size: 123456},
			},
			wantAsset: &matchName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platformAsset(tt.assets)
			if tt.wantAsset == nil {
				if got != nil {
					t.Errorf("platformAsset(): got %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("platformAsset(): got nil, want asset with name %q", *tt.wantAsset)
				} else if got.Name != *tt.wantAsset {
					t.Errorf("platformAsset(): got %q, want %q", got.Name, *tt.wantAsset)
				}
			}
		})
	}
}

// Test_Check_EdgeCases covers additional edge cases and boundary conditions.
func Test_Check_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		current        string
		serverResponse *Release
		serverStatus   int
		wantRelease    bool
		description    string
	}{
		{
			name:        "invalid semver without v prefix",
			current:     "1.2.3",
			wantRelease: false,
			description: "missing v prefix should fail validation",
		},
		{
			name:         "incomplete version is valid shorthand, no matching release",
			current:      "v1.2",
			serverStatus: http.StatusNotFound,
			wantRelease:  false,
			description:  "x/mod/semver accepts \"v1.2\" as valid (implicit .0 patch), so Check reaches the network; a 404 means no release",
		},
		{
			name:        "invalid semver too many segments",
			current:     "v1.2.3.4",
			wantRelease: false,
			description: "too many segments should fail validation",
		},
		{
			name:        "invalid semver non-numeric",
			current:     "v.a.b.c",
			wantRelease: false,
			description: "non-numeric version should fail validation",
		},
		{
			name:        "invalid semver just v prefix",
			current:     "v",
			wantRelease: false,
			description: "bare v prefix should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/latest" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				status := tt.serverStatus
				if status == 0 {
					t.Errorf("test case %q reached the network but sets no serverStatus", tt.name)
					status = http.StatusNotFound
				}
				w.WriteHeader(status)
				if tt.serverResponse != nil && status == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

			got, err := Check(tt.current, "")
			if (got != nil) != tt.wantRelease {
				t.Errorf("Check(%q, \"\"): got release=%v, want release=%v (%s)",
					tt.current, got != nil, tt.wantRelease, tt.description)
			}
			// For edge cases, we expect no error on success or no-update cases
			if tt.wantRelease && err != nil {
				t.Errorf("Check(%q, \"\"): got err=%v, want nil (%s)", tt.current, err, tt.description)
			}
		})
	}
}

// Test_Check_MalformedJSON verifies a 200 response with an invalid JSON body
// is treated as a failed check (nil), not a panic or partial decode.
func Test_Check_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{not valid json"))
	}))
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	got, err := Check("v1.0.0", "")
	if got != nil {
		t.Errorf("Check() with malformed JSON body = %v, want nil", got)
	}
	if err == nil {
		t.Errorf("Check() with malformed JSON should return error, got nil")
	}
}

// Test_FetchRelease_EdgeCases covers error cases and status code variations.
func Test_FetchRelease_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		tag              string
		serverStatus     int
		wantErr          bool
		wantRateLimitErr bool
		description      string
	}{
		{
			name:             "forbidden status",
			tag:              "v1.0.0",
			serverStatus:     http.StatusForbidden,
			wantErr:          true,
			wantRateLimitErr: true,
			description:      "403 Forbidden should return RateLimitError",
		},
		{
			name:         "service unavailable",
			tag:          "v1.0.0",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
			description:  "503 Service Unavailable should return error with HTTP status",
		},
		{
			name:         "bad gateway",
			tag:          "v1.0.0",
			serverStatus: http.StatusBadGateway,
			wantErr:      true,
			description:  "502 Bad Gateway should return error with HTTP status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

			_, err := FetchRelease(tt.tag, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchRelease(%q, \"\"): got err=%v, wantErr=%v (%s)",
					tt.tag, err != nil, tt.wantErr, tt.description)
			}
			if tt.wantRateLimitErr && err != nil {
				var rateLimitErr *ratelimiter.RateLimitError
				if !errors.As(err, &rateLimitErr) {
					t.Errorf("FetchRelease(%q) expected RateLimitError, got %T: %v", tt.tag, err, err)
				}
			} else if err != nil && !strings.Contains(err.Error(), "HTTP") {
				t.Errorf("FetchRelease error message should contain HTTP status code: %v", err)
			}
		})
	}
}

// Test_PlatformAsset_EdgeCases covers boundary conditions in asset selection.
func Test_PlatformAsset_EdgeCases(t *testing.T) {
	exactMatchName := fmt.Sprintf("%s-%s", goruntime.GOOS, goruntime.GOARCH)
	multiMatchName := fmt.Sprintf("app-%s-%s.tar.gz", goruntime.GOOS, goruntime.GOARCH)
	tests := []struct {
		name        string
		assets      []Asset
		wantAsset   *string
		description string
	}{
		{
			name:        "empty asset list",
			assets:      []Asset{},
			wantAsset:   nil,
			description: "no assets should return nil",
		},
		{
			name: "asset name exactly matching suffix",
			assets: []Asset{
				{Name: exactMatchName, BrowserDownloadURL: "https://example.com/asset"},
			},
			wantAsset:   &exactMatchName,
			description: "exact suffix match should be found",
		},
		{
			name: "multiple assets, only one matches",
			assets: []Asset{
				{Name: "app-decoy-platform.exe", BrowserDownloadURL: "https://example.com/decoy1"},
				{Name: "app-another-decoy", BrowserDownloadURL: "https://example.com/decoy2"},
				{Name: multiMatchName, BrowserDownloadURL: "https://example.com/match"},
			},
			wantAsset:   &multiMatchName,
			description: "should find the platform-specific asset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platformAsset(tt.assets)
			if tt.wantAsset == nil {
				if got != nil {
					t.Errorf("platformAsset(): got %v, want nil (%s)", got, tt.description)
				}
			} else {
				if got == nil {
					t.Errorf("platformAsset(): got nil, want asset (%s)", tt.description)
				} else if got.Name != *tt.wantAsset {
					t.Errorf("platformAsset(): got %q, want %q (%s)", got.Name, *tt.wantAsset, tt.description)
				}
			}
		})
	}
}

// Test_PlatformAsset_ReturnsPointerNotCopy verifies the return value is a pointer.
func Test_PlatformAsset_ReturnsPointerNotCopy(t *testing.T) {
	assets := []Asset{
		{Name: fmt.Sprintf("%s-%s", goruntime.GOOS, goruntime.GOARCH), BrowserDownloadURL: "http://example.com", Size: 1000},
	}
	result := platformAsset(assets)
	if result == nil {
		t.Fatalf("platformAsset() = nil, want match for %s-%s", goruntime.GOOS, goruntime.GOARCH)
	}
	if result != &assets[0] {
		t.Errorf("platformAsset should return pointer to asset, not a copy")
	}
}

// TestCheck_RateLimit403 verifies that a 403 response with X-RateLimit-Reset
// header returns a RateLimitError with a parsed reset time.
func TestCheck_RateLimit403(t *testing.T) {
	resetTime := time.Now().Add(1 * time.Hour)
	resetUnix := resetTime.Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	_, err := Check("v1.0.0", "")
	if err == nil {
		t.Fatalf("Check() with 403 should return error, got nil")
	}

	var rateLimitErr *ratelimiter.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("Check() with 403 should return RateLimitError, got %T: %v", err, err)
	}

	// Verify the error message contains something more specific than "HTTP 403"
	if !strings.Contains(rateLimitErr.Message, "rate limit exceeded") {
		t.Errorf("Check() RateLimitError message should mention rate limit, got: %s", rateLimitErr.Message)
	}

	// Verify the reset time was parsed
	if rateLimitErr.ResetTime == nil {
		t.Errorf("Check() RateLimitError should have ResetTime parsed, got nil")
	} else if rateLimitErr.ResetTime.Unix() != resetUnix {
		t.Errorf("Check() RateLimitError ResetTime = %d, want %d", rateLimitErr.ResetTime.Unix(), resetUnix)
	}
}

// TestCheck_RateLimit403NoHeader verifies that a 403 response without
// X-RateLimit-Reset header returns a generic rate-limit message.
func TestCheck_RateLimit403NoHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		// No X-RateLimit-Reset header
	}))
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	_, err := Check("v1.0.0", "")
	if err == nil {
		t.Fatalf("Check() with 403 and no header should return error, got nil")
	}

	var rateLimitErr *ratelimiter.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("Check() with 403 and no header should return RateLimitError, got %T: %v", err, err)
	}

	// Verify it has the generic message
	if !strings.Contains(rateLimitErr.Message, "rate limit exceeded") {
		t.Errorf("Check() RateLimitError message should mention rate limit, got: %s", rateLimitErr.Message)
	}

	// Verify ResetTime is nil (not parseable)
	if rateLimitErr.ResetTime != nil {
		t.Errorf("Check() RateLimitError with no header should have nil ResetTime, got %v", rateLimitErr.ResetTime)
	}
}

// TestFetchRelease_RateLimit403 verifies that a 403 response in FetchRelease
// returns a RateLimitError.
func TestFetchRelease_RateLimit403(t *testing.T) {
	resetTime := time.Now().Add(30 * time.Minute)
	resetUnix := resetTime.Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tags/v1.0.0" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	_, err := FetchRelease("v1.0.0", "")
	if err == nil {
		t.Fatalf("FetchRelease() with 403 should return error, got nil")
	}

	var rateLimitErr *ratelimiter.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("FetchRelease() with 403 should return RateLimitError, got %T: %v", err, err)
	}

	if !strings.Contains(rateLimitErr.Message, "rate limit exceeded") {
		t.Errorf("FetchRelease() RateLimitError message should mention rate limit, got: %s", rateLimitErr.Message)
	}

	if rateLimitErr.ResetTime == nil {
		t.Errorf("FetchRelease() RateLimitError should have ResetTime parsed, got nil")
	}
}
