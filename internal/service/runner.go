package service

import (
	"context"
	"net/http"
	"sync"

	"github.com/rs/zerolog"

	"github.com/samber/do/v2"

	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	

	"github.com/codozor/fwkeeper/internal/config"
)

type Runner struct {
	configuration config.Configuration

	logger zerolog.Logger

	client kubernetes.Interface

	restCfg *rest.Config

	forwarders []*Forwarder

	cancel context.CancelFunc

	wg sync.WaitGroup

	transport http.RoundTripper

	upgrader spdy.Upgrader
}

func runnerProvider(injector do.Injector) (*Runner, error) {
	configuration := do.MustInvoke[config.Configuration](injector)

	logger := do.MustInvoke[zerolog.Logger](injector)

	client := do.MustInvoke[kubernetes.Interface](injector)

	restCfg := do.MustInvoke[*rest.Config](injector)

	// SPDY Transport
	transport, upgrader, err := spdy.RoundTripperFor(restCfg)
    if err != nil {
		return nil, err
    }

	return &Runner{ 
		configuration: configuration,
		logger: logger,
		client: client,
		restCfg: restCfg,
		transport: transport,
		upgrader: upgrader,
	}, nil
}

func (r *Runner) Start() error {

	ctx := r.logger.WithContext(context.Background())
	ctx, r.cancel = context.WithCancel(ctx)

	r.startBanner(ctx)

	for _, pf := range r.configuration.Forwards {
		locator, err := NewPodLocator(pf.Resource, pf.Namespace)
		if err != nil {
			return err			
		}

		f, err := NewForwarder(locator, pf, r.client, r.transport, r.upgrader)
		if err != nil {
			return err			
		}

		r.forwarders = append(r.forwarders, f)
	}

	for _, f := range r.forwarders {
		r.wg.Go(func() {
			f.Start(ctx)
		})
	}

	return nil
}

func (r *Runner) startBanner(ctx context.Context) {
	log := zerolog.Ctx(ctx)

	log.Info().Msg(`----------------------------------------------`)
	log.Info().Msg(`   ___                                        `)
	log.Info().Msg(`  / __\_      __/\ /\___  ___ _ __   ___ _ __ `)
	log.Info().Msg(` / _\ \ \ /\ / / //_/ _ \/ _ \ '_ \ / _ \ '__|`)
	log.Info().Msg(`/ /    \ V  V / __ \  __/  __/ |_) |  __/ |   `)
	log.Info().Msg(`\/      \_/\_/\/  \/\___|\___| .__/ \___|_|   `)
	log.Info().Msg(`----------------------------------------------`)                                                                                            
}

func (r *Runner) Shutdown() {
	log := r.logger

	r.cancel()

	log.Info().Msg(`fwkeeper Stopping...`)

	r.wg.Wait()

	log.Info().Msg(`------------------------------------------------------------------`)
	log.Info().Msg(`fwkeeper Stopped`)
	log.Info().Msg(`------------------------------------------------------------------`)
}
