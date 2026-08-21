package config

import (
	"os"
	"strconv"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetInstallScriptURL returns the URL of the install script, allowing it to be
// overridden via the INSTALL_SCRIPT_URL environment variable (primarily for testing).
func GetInstallScriptURL() string {
	return getEnvOrDefault("INSTALL_SCRIPT_URL", "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/install.sh")
}

// GetReleasesBaseURL returns the base URL for the repo's releases (not
// including the "/releases" path segment — callers append that, plus
// "/latest" or "/tags/{tag}"), allowing it to be overridden via the
// APP_VERSION_RELEASES_BASE_URL environment variable (primarily for
// testing). Defaults to the public github.com host since litelensapp/litelens
// is public and the unauthenticated path takes precedence; set this to
// https://api.github.com/repos/<owner>/<repo> when a private repo requires
// an access token. Used by the app's own auto-updater (internal/updater);
// plugin marketplace fetches use GetMarketplaceBaseURL instead.
func GetReleasesBaseURL() string {
	return getEnvOrDefault("APP_VERSION_RELEASES_BASE_URL", "https://github.com/litelensapp/litelens")
}

// GetMarketplaceBaseURL returns the GitHub API base URL used as the default
// plugin marketplace source, allowing it to be overridden via the
// MARKETPLACE_BASE_URL environment variable (primarily for testing). This is
// split from GetReleasesBaseURL so the app's own update checks and the
// plugin marketplace default can be pointed at different repos independently.
// internal/plugin/download.go falls back to this only when the user hasn't
// configured a per-instance marketplace repo URL in Settings.
func GetMarketplaceBaseURL() string {
	return getEnvOrDefault("MARKETPLACE_BASE_URL", "https://api.github.com/repos/litelensapp/litelens-plugins/releases")
}

// IsPrivateRepoAccess reports whether GitHub release assets should be fetched
// via the authenticated API asset endpoint (required for private repos) rather
// than the public browser_download_url (which rejects Bearer tokens on private
// repos). Overridable via the PRIVATE_REPO_ACCESS environment variable.
// Defaults to false because the shipped repo (litelensapp/litelens) is public.
// Reserved for the app's own self-update flow (internal/app/updater.go,
// internal/updater/updater.go) — the plugin marketplace does not use this;
// see internal/app/plugin.go's use of MarketplaceRepository.Private instead.
func IsPrivateRepoAccess() bool {
	return getBoolEnvOrDefault("PRIVATE_REPO_ACCESS", false)
}

// IsMarketplaceEnabled reports whether the plugin marketplace feature is
// enabled — gating both marketplace discovery/install/remove on the backend
// and the marketplace UI on the frontend. Overridable via the
// MARKETPLACE_ENABLED environment variable. Defaults to false (marketplace
// ships disabled).
func IsMarketplaceEnabled() bool {
	return getBoolEnvOrDefault("MARKETPLACE_ENABLED", false)
}

// GetRootDirOverride returns the LITELENS_ROOT_DIR environment variable, which
// overrides the default ~/.litelens storage directory in production mode.
// Empty when unset. Wired into internal/storage via storage.SetRootDirOverride
// at startup (main.go) — internal/storage cannot import internal/config
// directly since config already depends on storage.
func GetRootDirOverride() string {
	return getEnvOrDefault("LITELENS_ROOT_DIR", "")
}
