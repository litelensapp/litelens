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
	return getEnvOrDefault("INSTALL_SCRIPT_URL", "https://api.github.com/repos/gknguyen/litelens/contents/scripts/install.sh")
}

// GetReleasesBaseURL returns the GitHub API base URL for the repo's releases,
// allowing it to be overridden via the APP_VERSION_RELEASES_BASE_URL
// environment variable (primarily for testing). Used by the app's own
// auto-updater (internal/updater); plugin marketplace fetches use
// GetMarketplaceBaseURL instead.
func GetReleasesBaseURL() string {
	return getEnvOrDefault("APP_VERSION_RELEASES_BASE_URL", "https://api.github.com/repos/gknguyen/litelens/releases")
}

// GetMarketplaceBaseURL returns the GitHub API base URL used as the default
// plugin marketplace source, allowing it to be overridden via the
// MARKETPLACE_BASE_URL environment variable (primarily for testing). This is
// split from GetReleasesBaseURL so the app's own update checks and the
// plugin marketplace default can be pointed at different repos independently.
// internal/plugin/download.go falls back to this only when the user hasn't
// configured a per-instance marketplace repo URL in Settings.
func GetMarketplaceBaseURL() string {
	return getEnvOrDefault("MARKETPLACE_BASE_URL", "https://api.github.com/repos/gknguyen/litelens/releases")
}

// IsPrivateRepoAccess reports whether GitHub release assets should be fetched
// via the authenticated API asset endpoint (required for private repos) rather
// than the public browser_download_url (which rejects Bearer tokens on private
// repos). Overridable via the PRIVATE_REPO_ACCESS environment variable.
// Defaults to true because the shipped repo (gknguyen/litelens) is private.
// Reserved for the app's own self-update flow (internal/app/updater.go,
// internal/updater/updater.go) — the plugin marketplace does not use this;
// see internal/app/plugin.go's use of MarketplaceRepository.Private instead.
func IsPrivateRepoAccess() bool {
	return getBoolEnvOrDefault("PRIVATE_REPO_ACCESS", true)
}
