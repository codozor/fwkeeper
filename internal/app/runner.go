package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
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
	configPath    string

	logger zerolog.Logger

	client kubernetes.Interface

	restCfg *rest.Config

	kubeConfigSource  string
	kubeConfigContext string

	// forwarders is a map of forward name to forwarder for easy management
	forwarders map[string]*forwarder.Forwarder

	// forwarderCancel maps forward name to its context cancel function
	forwarderCancel map[string]context.CancelFunc

	ctx    context.Context
	cancel context.CancelFunc

	wg sync.WaitGroup
	mu sync.Mutex // protects forwarders and forwarderCancel maps
}

// New creates a new Runner with all dependencies injected.
func New(
	configuration config.Configuration,
	configPath string,
	logger zerolog.Logger,
	client kubernetes.Interface,
	restCfg *rest.Config,
	kubeConfigSource string,
	kubeConfigContext string,
) *Runner {
	return &Runner{
		configuration:     configuration,
		configPath:        configPath,
		logger:            logger,
		client:            client,
		restCfg:           restCfg,
		kubeConfigSource:  kubeConfigSource,
		kubeConfigContext: kubeConfigContext,
		forwarders:        make(map[string]*forwarder.Forwarder),
		forwarderCancel:   make(map[string]context.CancelFunc),
	}
}

// Start initializes and starts all configured forwarders.
func (r *Runner) Start() error {
	ctx := r.logger.WithContext(context.Background())
	ctx, r.cancel = context.WithCancel(ctx)
	r.ctx = ctx

	r.startBanner(ctx)

	log := zerolog.Ctx(ctx)

	log.Info().Msgf("Kubernetes config source: %s (context: %s)", r.kubeConfigSource, r.kubeConfigContext)

	// Start initial forwarders
	nErr := r.startForwarders(ctx)
	if nErr > 0 {
		return fmt.Errorf("cannot start: %d configuration error(s) - see logs above", nErr)
	}

	// Start watcher for config changes and signal handling
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.watchConfigAndSignals(ctx)
	}()

	return nil
}

// startForwarders creates and starts all configured forwarders.
// Returns the number of errors encountered.
func (r *Runner) startForwarders(ctx context.Context) int {
	log := zerolog.Ctx(ctx)
	nErr := 0

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, pf := range r.configuration.Forwards {
		if err := r.startForwarder(ctx, pf); err != nil {
			log.Err(err).Msgf("Cannot configure forwarder %s", pf.Name)
			nErr++
		}
	}

	return nErr
}

// startForwarder creates and starts a single forwarder.
// Must be called with r.mu locked.
func (r *Runner) startForwarder(ctx context.Context, pf config.PortForwardConfiguration) error {
	log := zerolog.Ctx(ctx)

	// Skip if already running
	if _, exists := r.forwarders[pf.Name]; exists {
		return nil
	}

	loc, err := locator.BuildLocator(pf.Resource, pf.Namespace, pf.Ports, r.client)
	if err != nil {
		return fmt.Errorf("failed to build locator: %w", err)
	}

	f, err := forwarder.New(loc, pf, r.client, r.restCfg)
	if err != nil {
		return fmt.Errorf("failed to create forwarder: %w", err)
	}

	// Create a child context for this forwarder
	fwdCtx, fwdCancel := context.WithCancel(ctx)

	r.forwarders[pf.Name] = f
	r.forwarderCancel[pf.Name] = fwdCancel

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		log.Info().Msgf("Starting forwarder: %s", pf.Name)
		f.Start(fwdCtx)
		log.Info().Msgf("Forwarder stopped: %s", pf.Name)
	}()

	return nil
}

// stopForwarder gracefully stops a single forwarder.
// Must be called with r.mu locked.
func (r *Runner) stopForwarder(name string) {
	log := r.logger
	if cancel, exists := r.forwarderCancel[name]; exists {
		log.Info().Msgf("Stopping forwarder: %s", name)
		cancel()
		delete(r.forwarders, name)
		delete(r.forwarderCancel, name)
	}
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

// watchConfigAndSignals watches for config file changes and signal handling.
func (r *Runner) watchConfigAndSignals(ctx context.Context) {
	log := zerolog.Ctx(ctx)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Err(err).Msg("Failed to create config file watcher")
		return
	}
	defer watcher.Close()

	// Watch the config file directory (not the file directly, as editors may replace it)
	configPath := r.configPath
	if configPath == "" {
		configPath = "fwkeeper.cue"
	}

	// Get the directory of the config file
	configDir := "."
	// Simple directory extraction (handles both absolute and relative paths)
	for i := len(configPath) - 1; i >= 0; i-- {
		if configPath[i] == '/' || configPath[i] == '\\' {
			configDir = configPath[:i]
			break
		}
	}
	if configDir == "" {
		configDir = "."
	}

	if err := watcher.Add(configDir); err != nil {
		log.Err(err).Msgf("Failed to watch config directory: %s", configDir)
		return
	}

	// Normalize config path for comparison (use absolute path to handle all cases)
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		absConfigPath = configPath // Fallback to original if abs fails
	}

	log.Info().Msgf("Watching config for changes: %s", configPath)

	// Setup SIGHUP handler for manual config reload
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Normalize event path for comparison
			absEventPath, err := filepath.Abs(event.Name)
			if err != nil {
				absEventPath = event.Name
			}

			// Check if this event is for the config file
			isConfigFile := absEventPath == absConfigPath || baseName(absEventPath) == baseName(absConfigPath)

			// Only process Write and Create events on the config file
			if (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) &&
				isConfigFile {
				log.Info().Msg("Config file changed, reloading")
				r.reloadConfig(ctx)
			}

		case <-sigCh:
			log.Info().Msg("Received SIGHUP signal, reloading config")
			r.reloadConfig(ctx)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Err(err).Msg("Config watcher error")
		}
	}
}

// baseName returns the filename part of a path.
func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// reloadConfig reloads the configuration from file and applies changes.
func (r *Runner) reloadConfig(ctx context.Context) {
	log := zerolog.Ctx(ctx)

	// Load new configuration
	newConfig, err := config.ReadConfiguration(r.configPath)
	if err != nil {
		log.Error().
			Err(err).
			Str("config_file", r.configPath).
			Msg("Configuration reload failed - keeping previous configuration. Fix the configuration file and try again")
		return
	}

	log.Info().Msg("Configuration reloaded successfully")

	r.mu.Lock()
	defer r.mu.Unlock()

	// Find forwarders to remove
	for name := range r.forwarders {
		found := false
		for _, pf := range newConfig.Forwards {
			if pf.Name == name {
				found = true
				break
			}
		}
		if !found {
			r.stopForwarder(name)
			log.Info().Msgf("Removed forward: %s", name)
		}
	}

	// Find forwarders to add or restart
	for _, pf := range newConfig.Forwards {
		if existing, exists := r.forwarders[pf.Name]; exists {
			// Check if configuration changed
			if configChanged(existing.Config(), pf) {
				r.stopForwarder(pf.Name)
				if err := r.startForwarder(ctx, pf); err != nil {
					log.Err(err).Msgf("Failed to restart forwarder: %s", pf.Name)
				} else {
					log.Info().Msgf("Restarted forward: %s", pf.Name)
				}
			}
		} else {
			// New forwarder
			if err := r.startForwarder(ctx, pf); err != nil {
				log.Err(err).Msgf("Failed to start new forwarder: %s", pf.Name)
			} else {
				log.Info().Msgf("Added forward: %s", pf.Name)
			}
		}
	}

	// Update the current configuration
	r.configuration = newConfig
}

// configChanged checks if a forwarder's configuration has changed.
func configChanged(oldConfig config.PortForwardConfiguration, newConfig config.PortForwardConfiguration) bool {
	// Check if namespace or resource changed
	if oldConfig.Namespace != newConfig.Namespace || oldConfig.Resource != newConfig.Resource {
		return true
	}

	// Check if ports changed
	if len(oldConfig.Ports) != len(newConfig.Ports) {
		return true
	}
	for i, port := range oldConfig.Ports {
		if i >= len(newConfig.Ports) || port != newConfig.Ports[i] {
			return true
		}
	}

	return false
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

