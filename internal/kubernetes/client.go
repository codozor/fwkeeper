package kubernetes

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// NewRestConfig creates a Kubernetes REST client configuration.
// It attempts to load the configuration from:
// 1. KUBECONFIG environment variable
// 2. ~/.kube/config (default kubeconfig location)
// 3. In-cluster configuration (when running in a pod)
func NewRestConfig() (*rest.Config, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if home := homedir.HomeDir(); home != "" {
		return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	} else {
		return rest.InClusterConfig()
	}
}

// NewClient creates a new Kubernetes client from a REST configuration.
func NewClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}
