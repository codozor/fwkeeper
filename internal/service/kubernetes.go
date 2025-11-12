package service

import (
	"os"
	"path/filepath"

	"github.com/samber/do/v2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func restConfigProvider(injector do.Injector) (*rest.Config, error) {
    if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
    } else if home := homedir.HomeDir(); home != "" {
        return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
    } else {
        return rest.InClusterConfig()
    }
}

func kubernetesProvider(injector do.Injector) (kubernetes.Interface, error) {
	config := do.MustInvoke[*rest.Config](injector)

    return kubernetes.NewForConfig(config)
}

