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
	// app's own requests — Connect() starts ~31 informers (each doing an
	// initial LIST) off one shared clientset. Raise the ceiling so the host
	// app doesn't self-throttle, in line with kubectl/Lens/k9s.
	restConfig.QPS = 50
	restConfig.Burst = 100

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
