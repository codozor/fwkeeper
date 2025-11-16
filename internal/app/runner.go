package app

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/codozor/fwkeeper/internal/config"
	"github.com/codozor/fwkeeper/internal/forwarder"
	"github.com/codozor/fwkeeper/internal/locator"
)

// Runner orchestrates multiple port forwarders and manages their lifecycle.
type Runner struct {
	configuration config.Configuration

	logger zerolog.Logger

	client kubernetes.Interface

	restCfg *rest.Config

	forwarders []*forwarder.Forwarder

	cancel context.CancelFunc

	wg sync.WaitGroup
}

// New creates a new Runner with all dependencies injected.
func New(
	configuration config.Configuration,
	logger zerolog.Logger,
	client kubernetes.Interface,
	restCfg *rest.Config,
) *Runner {
	return &Runner{
		configuration: configuration,
		logger:        logger,
		client:        client,
		restCfg:       restCfg,
	}
}

// Start initializes and starts all configured forwarders.
func (r *Runner) Start() error {
	ctx := r.logger.WithContext(context.Background())
	ctx, r.cancel = context.WithCancel(ctx)

	r.startBanner(ctx)

	for _, pf := range r.configuration.Forwards {
		loc, err := locator.BuildLocator(pf.Resource, pf.Namespace, pf.Ports, r.client)
		if err != nil {
			return err
		}

		f, err := forwarder.New(loc, pf, r.client, r.restCfg)
		if err != nil {
			return err
		}

		r.forwarders = append(r.forwarders, f)
	}

	for _, f := range r.forwarders {
		f := f // capture in closure
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			f.Start(ctx)
		}()
	}

	return nil
}

// startBanner logs the application startup banner.
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

// Shutdown gracefully shuts down the runner and all forwarders.
func (r *Runner) Shutdown() {
	log := r.logger

	r.cancel()

	log.Info().Msg(`fwkeeper Stopping...`)

	r.wg.Wait()

	log.Info().Msg(`------------------------------------------------------------------`)
	log.Info().Msg(`fwkeeper Stopped`)
	log.Info().Msg(`------------------------------------------------------------------`)
}
