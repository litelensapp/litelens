package kube

import (
	"sort"

	"k8s.io/client-go/tools/clientcmd"
)

// LoadingRules returns kubeconfig loading rules for the given paths.
// When paths is empty it falls back to the default rules (KUBECONFIG env var / ~/.kube/config).
func LoadingRules(paths []string) *clientcmd.ClientConfigLoadingRules {
	if len(paths) == 0 {
		return clientcmd.NewDefaultClientConfigLoadingRules()
	}
	return &clientcmd.ClientConfigLoadingRules{Precedence: paths}
}

// ListContexts returns the sorted list of context names from the given kubeconfig files.
func ListContexts(paths []string) ([]string, error) {
	rawConfig, err := LoadingRules(paths).GetStartingConfig()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(rawConfig.Contexts))
	for name := range rawConfig.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// CurrentContext returns the current-context field from the given kubeconfig files.
func CurrentContext(paths []string) (string, error) {
	rawConfig, err := LoadingRules(paths).GetStartingConfig()
	if err != nil {
		return "", err
	}
	return rawConfig.CurrentContext, nil
}
