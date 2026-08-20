package config

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/litelensapp/litelens/internal/storage"
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
	ClusterDefaultNamespaces map[string][]string    `json:"clusterDefaultNamespaces"`
	ShellPath               string                  `json:"shellPath"`
	KubeconfigPaths         []string                `json:"kubeconfigPaths"`
	Locale                  string                  `json:"locale"`
	PluginsDir              string                  `json:"pluginsDir"`
	PluginDisabledState     map[string]bool         `json:"pluginDisabledState"`
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
	if err := unmarshalAndMigrate(data, &s); err != nil {
		return Settings{}, err
	}

	return s, nil
}

// unmarshalAndMigrate unmarshals JSON data into Settings and applies legacy field migration.
func unmarshalAndMigrate(data []byte, s *Settings) error {
	if err := json.Unmarshal(data, s); err != nil {
		return err
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

	return nil
}

func Save(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(storage.Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func settingsPath() (string, error) {
	return storage.Dir("settings.json"), nil
}
