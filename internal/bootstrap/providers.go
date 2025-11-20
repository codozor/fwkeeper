package bootstrap

import (
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/codozor/fwkeeper/internal/app"
	"github.com/codozor/fwkeeper/internal/config"
	kubeinternal "github.com/codozor/fwkeeper/internal/kubernetes"
)

// Providers registers all service providers for dependency injection.
var Providers = do.Package(
	do.Lazy(restConfigInfoProvider),
	do.Lazy(restConfigProvider),
	do.Lazy(kubernetesProvider),
	do.Lazy(runnerProvider),
)

// restConfigInfoProvider creates a Kubernetes REST client configuration with source info.
func restConfigInfoProvider(injector do.Injector) (kubeinternal.RestConfigInfo, error) {
	return kubeinternal.NewRestConfig()
}

// restConfigProvider extracts just the config from RestConfigInfo.
func restConfigProvider(injector do.Injector) (*rest.Config, error) {
	info := do.MustInvoke[kubeinternal.RestConfigInfo](injector)
	return info.Config, nil
}

// kubernetesProvider creates a Kubernetes client.
func kubernetesProvider(injector do.Injector) (kubernetes.Interface, error) {
	config := do.MustInvoke[*rest.Config](injector)
	return kubeinternal.NewClient(config)
}

// runnerProvider creates the application runner with all dependencies.
// Note: SPDY transport and upgrader are created per-forwarder to avoid data races.
func runnerProvider(injector do.Injector) (*app.Runner, error) {
	cfg := do.MustInvoke[config.Configuration](injector)
	logger := do.MustInvoke[zerolog.Logger](injector)
	client := do.MustInvoke[kubernetes.Interface](injector)
	restCfg := do.MustInvoke[*rest.Config](injector)
	restConfigInfo := do.MustInvoke[kubeinternal.RestConfigInfo](injector)

	return app.New(cfg, logger, client, restCfg, restConfigInfo.Source, restConfigInfo.Context), nil
}
