package config

import (
	"testing"
)

// Test_getEnvOrDefault_UnsetEnvReturnsDefault verifies that unset env var returns default.
func Test_getEnvOrDefault_UnsetEnvReturnsDefault(t *testing.T) {
	result := getEnvOrDefault("DEFINITELY_NOT_SET_VAR_XYZ", "default_value")
	if result != "default_value" {
		t.Errorf("expected default value for unset env var, got %s", result)
	}
}

// Test_getEnvOrDefault_EmptyStringVsUnset verifies that empty string is distinguished from unset.
func Test_getEnvOrDefault_EmptyStringVsUnset(t *testing.T) {
	t.Setenv("TEST_EMPTY_VAR", "")

	// When set to empty string, getEnvOrDefault should return the empty string, not the default
	result := getEnvOrDefault("TEST_EMPTY_VAR", "default_value")
	if result != "" {
		t.Errorf("expected empty string when env var is set to empty, got %s", result)
	}

	// When unset, it should return the default
	result2 := getEnvOrDefault("UNSET_VAR_DEFINITELY", "default_value")
	if result2 != "default_value" {
		t.Errorf("expected default value for truly unset var, got %s", result2)
	}
}

// Test_GetInstallScriptURL_DefaultAndOverride verifies default vs. override behavior.
// Run as subtests of one parent so the unset case can't be polluted by t.Setenv
// leaking into a later test (t.Setenv has no way to unset a var once set).
func Test_GetInstallScriptURL_DefaultAndOverride(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		result := GetInstallScriptURL()
		expected := "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/install.sh"
		if result != expected {
			t.Errorf("expected default URL %s, got %s", expected, result)
		}
	})

	t.Run("override when set", func(t *testing.T) {
		customURL := "https://custom.example.com/install.sh"
		t.Setenv("INSTALL_SCRIPT_URL", customURL)

		result := GetInstallScriptURL()
		if result != customURL {
			t.Errorf("expected custom URL %s, got %s", customURL, result)
		}
	})
}

// Test_GetInstallScriptURL_EmptyStringWhenSetToEmpty verifies empty string overrides default.
func Test_GetInstallScriptURL_EmptyStringWhenSetToEmpty(t *testing.T) {
	t.Setenv("INSTALL_SCRIPT_URL", "")

	result := GetInstallScriptURL()
	if result != "" {
		t.Errorf("expected empty string when INSTALL_SCRIPT_URL is set to empty, got %s", result)
	}
}

// Test_GetReleasesBaseURL_DefaultAndOverride verifies default vs. override behavior.
// Run as subtests of one parent so the unset case can't be polluted by t.Setenv
// leaking into a later test (t.Setenv has no way to unset a var once set).
func Test_GetReleasesBaseURL_DefaultAndOverride(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		result := GetReleasesBaseURL()
		expected := "https://api.github.com/repos/litelensapp/litelens/releases"
		if result != expected {
			t.Errorf("expected default URL %s, got %s", expected, result)
		}
	})

	t.Run("override when set", func(t *testing.T) {
		customURL := "https://custom-releases.example.com"
		t.Setenv("APP_VERSION_RELEASES_BASE_URL", customURL)

		result := GetReleasesBaseURL()
		if result != customURL {
			t.Errorf("expected custom URL %s, got %s", customURL, result)
		}
	})
}

// Test_GetReleasesBaseURL_EmptyStringWhenSetToEmpty verifies empty string overrides default.
func Test_GetReleasesBaseURL_EmptyStringWhenSetToEmpty(t *testing.T) {
	t.Setenv("APP_VERSION_RELEASES_BASE_URL", "")

	result := GetReleasesBaseURL()
	if result != "" {
		t.Errorf("expected empty string when APP_VERSION_RELEASES_BASE_URL is set to empty, got %s", result)
	}
}

// Test_GetMarketplaceBaseURL_DefaultAndOverride verifies default vs. override behavior,
// independent of APP_VERSION_RELEASES_BASE_URL.
func Test_GetMarketplaceBaseURL_DefaultAndOverride(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		result := GetMarketplaceBaseURL()
		expected := "https://api.github.com/repos/litelensapp/litelens/releases"
		if result != expected {
			t.Errorf("expected default URL %s, got %s", expected, result)
		}
	})

	t.Run("override when set", func(t *testing.T) {
		customURL := "https://custom-marketplace.example.com"
		t.Setenv("MARKETPLACE_BASE_URL", customURL)

		result := GetMarketplaceBaseURL()
		if result != customURL {
			t.Errorf("expected custom URL %s, got %s", customURL, result)
		}
	})

	t.Run("independent of APP_VERSION_RELEASES_BASE_URL", func(t *testing.T) {
		t.Setenv("APP_VERSION_RELEASES_BASE_URL", "https://releases.example.com")

		result := GetMarketplaceBaseURL()
		expected := "https://api.github.com/repos/litelensapp/litelens/releases"
		if result != expected {
			t.Errorf("expected GetMarketplaceBaseURL to ignore APP_VERSION_RELEASES_BASE_URL, got %s", result)
		}
	})
}

// Test_GetMarketplaceBaseURL_EmptyStringWhenSetToEmpty verifies empty string overrides default.
func Test_GetMarketplaceBaseURL_EmptyStringWhenSetToEmpty(t *testing.T) {
	t.Setenv("MARKETPLACE_BASE_URL", "")

	result := GetMarketplaceBaseURL()
	if result != "" {
		t.Errorf("expected empty string when MARKETPLACE_BASE_URL is set to empty, got %s", result)
	}
}

// Test_getEnvOrDefault_WithWhitespacePreservesIt verifies whitespace is preserved in values.
func Test_getEnvOrDefault_WithWhitespacePreservesIt(t *testing.T) {
	valueWithSpaces := "  https://example.com/path  "
	t.Setenv("TEST_WHITESPACE_VAR", valueWithSpaces)

	result := getEnvOrDefault("TEST_WHITESPACE_VAR", "default")
	if result != valueWithSpaces {
		t.Errorf("expected whitespace to be preserved, got %q", result)
	}
}

// Test_GetReleasesBaseURL_SpecialCharactersInURL verifies special chars are preserved.
func Test_GetReleasesBaseURL_SpecialCharactersInURL(t *testing.T) {
	specialURL := "https://example.com/api?token=abc123&version=latest#v1"
	t.Setenv("APP_VERSION_RELEASES_BASE_URL", specialURL)

	result := GetReleasesBaseURL()
	if result != specialURL {
		t.Errorf("expected special characters to be preserved, got %s", result)
	}
}
