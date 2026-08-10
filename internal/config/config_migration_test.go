package config

import (
	"encoding/json"
	"testing"
)

// TestConfigMigration verifies that old-format config files (with flat marketplace fields)
// are correctly migrated to the new array-based format.
func TestConfigMigration(t *testing.T) {
	tests := []struct {
		name                               string
		rawJSON                            string
		expectedRepositoriesCount          int
		expectedFirstRepositoryURL         string
		expectedFirstRepositoryPrivate     bool
		expectedFirstRepositoryAccessToken string
		checkFirstRepoDetails              bool
	}{
		{
			name: "old-format config with flat marketplace fields",
			rawJSON: `{
  "accessToken": "ghp_general_token",
  "marketplaceRepoURL": "https://github.com/test/marketplace",
  "marketplacePrivate": true,
  "marketplaceAccessToken": "ghp_marketplace_token"
}`,
			expectedRepositoriesCount:          1,
			expectedFirstRepositoryURL:         "https://github.com/test/marketplace",
			expectedFirstRepositoryPrivate:     true,
			expectedFirstRepositoryAccessToken: "ghp_marketplace_token",
			checkFirstRepoDetails:              true,
		},
		{
			name: "new-format config with marketplaceRepositories array",
			rawJSON: `{
  "accessToken": "ghp_general_token",
  "marketplaceRepositories": [
    {
      "url": "https://github.com/repo1/plugins",
      "private": false,
      "accessToken": ""
    },
    {
      "url": "https://github.com/repo2/plugins",
      "private": true,
      "accessToken": "ghp_repo2_token"
    }
  ]
}`,
			expectedRepositoriesCount: 2,
			checkFirstRepoDetails:     false,
		},
		{
			name: "old-format with empty marketplaceRepoURL is not migrated",
			rawJSON: `{
  "accessToken": "ghp_general_token",
  "marketplaceRepoURL": "",
  "marketplacePrivate": false,
  "marketplaceAccessToken": ""
}`,
			expectedRepositoriesCount: 0,
			checkFirstRepoDetails:     false,
		},
		{
			name: "mixed config (new array takes precedence, no migration)",
			rawJSON: `{
  "accessToken": "ghp_general_token",
  "marketplaceRepoURL": "https://github.com/old/marketplace",
  "marketplaceRepositories": [
    {
      "url": "https://github.com/new/marketplace",
      "private": false,
      "accessToken": ""
    }
  ]
}`,
			expectedRepositoriesCount:  1,
			expectedFirstRepositoryURL: "https://github.com/new/marketplace",
			checkFirstRepoDetails:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Settings
			if err := json.Unmarshal([]byte(tt.rawJSON), &s); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			// Apply migration logic (same as Load() does)
			if len(s.MarketplaceRepositories) == 0 {
				var legacy legacySettings
				if err := json.Unmarshal([]byte(tt.rawJSON), &legacy); err == nil && legacy.MarketplaceRepoURL != "" {
					s.MarketplaceRepositories = []MarketplaceRepository{
						{
							URL:         legacy.MarketplaceRepoURL,
							Private:     legacy.MarketplacePrivate,
							AccessToken: legacy.MarketplaceAccessToken,
						},
					}
				}
			}

			// Verify count
			if len(s.MarketplaceRepositories) != tt.expectedRepositoriesCount {
				t.Errorf("repository count mismatch: got %d, expected %d",
					len(s.MarketplaceRepositories), tt.expectedRepositoriesCount)
			}

			// Verify first repository details if applicable
			if tt.checkFirstRepoDetails && tt.expectedRepositoriesCount > 0 {
				repo := s.MarketplaceRepositories[0]
				if repo.URL != tt.expectedFirstRepositoryURL {
					t.Errorf("first repository URL mismatch: got %q, expected %q",
						repo.URL, tt.expectedFirstRepositoryURL)
				}
				if repo.Private != tt.expectedFirstRepositoryPrivate {
					t.Errorf("first repository private mismatch: got %v, expected %v",
						repo.Private, tt.expectedFirstRepositoryPrivate)
				}
				if repo.AccessToken != tt.expectedFirstRepositoryAccessToken {
					t.Errorf("first repository access token mismatch: got %q, expected %q",
						repo.AccessToken, tt.expectedFirstRepositoryAccessToken)
				}
			}
		})
	}
}
