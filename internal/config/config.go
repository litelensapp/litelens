package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type ClusterProxy struct {
	HttpProxy  string `json:"httpProxy"`
	HttpsProxy string `json:"httpsProxy"`
}

type MarketplaceRepository struct {
	URL         string `json:"url"`
	Private     bool   `json:"private"`
	AccessToken string `json:"accessToken"`
	Locked      bool   `json:"locked"`
	Disabled    bool   `json:"disabled"`
}

type Settings struct {
	AccessToken             string                  `json:"accessToken"`
	ClusterProxies          map[string]ClusterProxy `json:"clusterProxies"`
	ShellPath               string                  `json:"shellPath"`
	KubeconfigPaths         []string                `json:"kubeconfigPaths"`
	Locale                  string                  `json:"locale"`
	PluginsDir              string                  `json:"pluginsDir"`
	MarketplaceRepositories []MarketplaceRepository `json:"marketplaceRepositories"`
}

// legacySettings is used internally for migration only: it mirrors the old flat structure.
type legacySettings struct {
	MarketplaceRepoURL     string `json:"marketplaceRepoURL"`
	MarketplacePrivate     bool   `json:"marketplacePrivate"`
	MarketplaceAccessToken string `json:"marketplaceAccessToken"`
}

func Load() (Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}

	// Migration: if MarketplaceRepositories is empty and the JSON contains old flat fields,
	// convert them into a single array entry.
	if len(s.MarketplaceRepositories) == 0 {
		var legacy legacySettings
		if err := json.Unmarshal(data, &legacy); err == nil && legacy.MarketplaceRepoURL != "" {
			s.MarketplaceRepositories = []MarketplaceRepository{
				{
					URL:         legacy.MarketplaceRepoURL,
					Private:     legacy.MarketplacePrivate,
					AccessToken: legacy.MarketplaceAccessToken,
				},
			}
		}
	}

	return s, nil
}

func Save(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "litelens", "settings.json"), nil
}
