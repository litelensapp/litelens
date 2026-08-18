package kube

import (
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
