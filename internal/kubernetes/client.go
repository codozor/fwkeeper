package kubernetes

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// RestConfigInfo contains the REST config, its source, and the active context.
type RestConfigInfo struct {
	Config  *rest.Config
	Source  string // Human-readable description of the kubeconfig source
	Context string // Active Kubernetes context name
}

// getCurrentContext extracts the active context name from kubeconfig.
// Returns "unknown" if context cannot be determined.
func getCurrentContext(kubeconfig string) string {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}

	config, err := rules.Load()
	if err != nil || config.CurrentContext == "" {
		return "unknown"
	}

	return config.CurrentContext
}

// NewRestConfig creates a Kubernetes REST client configuration.
// It attempts to load the configuration from:
// 1. KUBECONFIG environment variable
// 2. ~/.kube/config (default kubeconfig location)
// 3. In-cluster configuration (when running in a pod)
//
// Returns the config, a description of which source was used, and the active context.
func NewRestConfig() (RestConfigInfo, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		return RestConfigInfo{
			Config:  config,
			Source:  "KUBECONFIG=" + kubeconfig,
			Context: getCurrentContext(kubeconfig),
		}, err
	}

	if home := homedir.HomeDir(); home != "" {
		configPath := filepath.Join(home, ".kube", "config")
		config, err := clientcmd.BuildConfigFromFlags("", configPath)
		return RestConfigInfo{
			Config:  config,
			Source:  "~/.kube/config",
			Context: getCurrentContext(configPath),
		}, err
	}

	config, err := rest.InClusterConfig()
	return RestConfigInfo{
		Config:  config,
		Source:  "in-cluster (running inside Kubernetes)",
		Context: "unknown",
	}, err
}

// NewClient creates a new Kubernetes client from a REST configuration.
func NewClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}
