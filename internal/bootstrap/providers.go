package bootstrap

import (
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"

	"github.com/codozor/fwkeeper/internal/app"
	"github.com/codozor/fwkeeper/internal/config"
	kubeinternal "github.com/codozor/fwkeeper/internal/kubernetes"
)

// Providers registers all service providers for dependency injection.
var Providers = do.Package(
	do.Lazy(restConfigProvider),
	do.Lazy(kubernetesProvider),
	do.Lazy(runnerProvider),
)

// restConfigProvider creates a Kubernetes REST client configuration.
func restConfigProvider(injector do.Injector) (*rest.Config, error) {
	return kubeinternal.NewRestConfig()
}

// kubernetesProvider creates a Kubernetes client.
func kubernetesProvider(injector do.Injector) (kubernetes.Interface, error) {
	config := do.MustInvoke[*rest.Config](injector)
	return kubeinternal.NewClient(config)
}

// runnerProvider creates the application runner with all dependencies.
func runnerProvider(injector do.Injector) (*app.Runner, error) {
	cfg := do.MustInvoke[config.Configuration](injector)
	logger := do.MustInvoke[zerolog.Logger](injector)
	client := do.MustInvoke[kubernetes.Interface](injector)
	restCfg := do.MustInvoke[*rest.Config](injector)

	// SPDY Transport for port-forward
	transport, upgrader, err := spdy.RoundTripperFor(restCfg)
	if err != nil {
		return nil, err
	}

	return app.New(cfg, logger, client, restCfg, transport, upgrader), nil
}
