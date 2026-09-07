package kube

import (
	"net/http"
	"net/url"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Ping verifies the API server is reachable by fetching the server version.
func Ping(cs kubernetes.Interface) error {
	_, err := cs.Discovery().ServerVersion()
	return err
}

// NewClientset builds a Kubernetes clientset for the given context.
// httpProxy / httpsProxy route API-server traffic through a proxy when set
// (e.g. EKS behind a corporate proxy). Pass empty strings for a direct connection.
// kubeconfigPaths lists the kubeconfig files to load; pass nil to use the default rules.
func NewClientset(contextName, httpProxy, httpsProxy string, kubeconfigPaths []string) (*kubernetes.Clientset, *rest.Config, error) {
	rules := LoadingRules(kubeconfigPaths)
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	restConfig, err := cfg.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	restConfig.Proxy = ProxyFunc(httpProxy, httpsProxy)
	// client-go defaults to QPS:5/Burst:10 when unset, which throttles this
	// app's own requests. Connect() starts ~9 cluster-wide informers plus,
	// per selected namespace, one informer for each of the ~24
	// namespace-scoped (nsscope) resource kinds — e.g. 7 namespaces selected
	// means ~168 informers all doing their initial LIST at once. Raise the
	// ceiling well above kubectl/Lens/k9s-style single-digit-namespace usage
	// so that burst doesn't self-throttle into spurious sync timeouts.
	restConfig.QPS = 100
	restConfig.Burst = 300

	cs, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	return cs, restConfig, nil
}

// ProxyFunc returns a proxy selector for rest.Config.Proxy / http.Transport.Proxy.
// Values are captured at call time; the returned function is goroutine-safe.
func ProxyFunc(httpProxy, httpsProxy string) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		if req.URL.Scheme == "https" && httpsProxy != "" {
			return url.Parse(httpsProxy)
		}
		if httpProxy != "" {
			return url.Parse(httpProxy)
		}
		return nil, nil
	}
}
