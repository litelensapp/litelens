package updater

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	goruntime "runtime"
	"testing"
)

// TestCheck_NoToken_UnauthenticatedPath verifies that Check with an empty
// token resolves the latest release entirely via the public github.com
// redirect/download convention and never touches an api.github.com-shaped
// endpoint (mirrors scripts/install.sh's unauthenticated path).
func TestCheck_NoToken_UnauthenticatedPath(t *testing.T) {
	assetName, err := assetFileName(goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform for this test: %v", err)
	}

	tests := []struct {
		name        string
		current     string
		latestTag   string
		wantRelease bool
	}{
		{name: "newer release available", current: "v1.0.0", latestTag: "v9.9.9", wantRelease: true},
		{name: "already up to date", current: "v9.9.9", latestTag: "v9.9.9", wantRelease: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiHit bool
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
				apiHit = true
				w.WriteHeader(http.StatusInternalServerError)
			})
			mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/releases/tag/"+tt.latestTag, http.StatusFound)
			})
			mux.HandleFunc("/releases/tag/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			mux.HandleFunc("/releases/download/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "2048")
				w.WriteHeader(http.StatusOK)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

			got, err := Check(tt.current, "")
			if err != nil {
				t.Fatalf("Check() returned err=%v, want nil", err)
			}
			if apiHit {
				t.Fatalf("Check() with no token must never call the api.github.com-shaped path")
			}
			if (got != nil) != tt.wantRelease {
				t.Fatalf("Check(): got release=%v, want release=%v", got != nil, tt.wantRelease)
			}
			if got != nil {
				wantAssetURL := fmt.Sprintf("%s/releases/download/%s/%s", server.URL, tt.latestTag, assetName)
				if got.AssetURL != wantAssetURL {
					t.Errorf("Check() AssetURL = %q, want %q", got.AssetURL, wantAssetURL)
				}
				if got.TagName != tt.latestTag {
					t.Errorf("Check() TagName = %q, want %q", got.TagName, tt.latestTag)
				}
				if got.DownloadSize != 2048 {
					t.Errorf("Check() DownloadSize = %d, want 2048", got.DownloadSize)
				}
			}
		})
	}
}

// TestFetchRelease_NoToken_UnauthenticatedPath verifies FetchRelease with an
// empty token builds the Release entirely from public URLs for the given
// tag, without any api.github.com-shaped request.
func TestFetchRelease_NoToken_UnauthenticatedPath(t *testing.T) {
	assetName, err := assetFileName(goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform for this test: %v", err)
	}

	var apiHit bool
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
		apiHit = true
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/releases/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4096")
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

	got, err := FetchRelease("v3.4.5", "")
	if err != nil {
		t.Fatalf("FetchRelease() returned err=%v, want nil", err)
	}
	if apiHit {
		t.Fatalf("FetchRelease() with no token must never call the api.github.com-shaped path")
	}
	if got == nil {
		t.Fatalf("FetchRelease() returned nil, want Release")
	}
	if got.TagName != "v3.4.5" {
		t.Errorf("FetchRelease() TagName = %q, want v3.4.5", got.TagName)
	}
	wantAssetURL := fmt.Sprintf("%s/releases/download/v3.4.5/%s", server.URL, assetName)
	if got.AssetURL != wantAssetURL {
		t.Errorf("FetchRelease() AssetURL = %q, want %q", got.AssetURL, wantAssetURL)
	}
	if got.DownloadSize != 4096 {
		t.Errorf("FetchRelease() DownloadSize = %d, want 4096", got.DownloadSize)
	}
}

func TestAssetFileName(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
		wantErr      bool
	}{
		{"darwin", "arm64", "litelens-darwin-arm64.zip", false},
		{"darwin", "amd64", "litelens-darwin-amd64.zip", false},
		{"linux", "amd64", "litelens-linux-amd64.tar.gz", false},
		{"windows", "amd64", "litelens-windows-amd64.exe", false},
		{"linux", "arm64", "", true},
		{"freebsd", "amd64", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got, err := assetFileName(tt.goos, tt.goarch)
			if (err != nil) != tt.wantErr {
				t.Fatalf("assetFileName(%q, %q) err=%v, wantErr=%v", tt.goos, tt.goarch, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("assetFileName(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestResolveLatestTagUnauthenticated(t *testing.T) {
	t.Run("resolves tag from redirect", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/releases/tag/v7.8.9", http.StatusFound)
		})
		mux.HandleFunc("/releases/tag/v7.8.9", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

		tag, err := resolveLatestTagUnauthenticated()
		if err != nil {
			t.Fatalf("resolveLatestTagUnauthenticated() returned err=%v", err)
		}
		if tag != "v7.8.9" {
			t.Errorf("resolveLatestTagUnauthenticated() = %q, want v7.8.9", tag)
		}
	})

	t.Run("non-200 final status returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

		_, err := resolveLatestTagUnauthenticated()
		if err == nil {
			t.Fatal("resolveLatestTagUnauthenticated() expected error for non-200 status, got nil")
		}
	})

	t.Run("no tag segment in resolved URL returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		t.Setenv("APP_VERSION_RELEASES_BASE_URL", server.URL)

		_, err := resolveLatestTagUnauthenticated()
		if err == nil {
			t.Fatal("resolveLatestTagUnauthenticated() expected error when no /releases/tag/ segment present, got nil")
		}
	})
}
