package forwarder

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/codozor/fwkeeper/internal/config"
	"github.com/codozor/fwkeeper/internal/locator"
)

// RetryConfig defines exponential backoff retry strategy.
type RetryConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// DefaultRetryConfig returns sensible defaults for retry strategy.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   1.5,
		Jitter:       true,
	}
}

// Forwarder manages port forwarding for a single pod.
type Forwarder struct {
	locator       locator.Locator
	configuration config.PortForwardConfiguration

	client     kubernetes.Interface
	restConfig *rest.Config

	transport http.RoundTripper
	upgrader  spdy.Upgrader

	retryConfig RetryConfig
	attempt     uint
}

// forwarderWriter adapts Kubernetes portforward output to structured logging.
type forwarderWriter struct {
	logger *zerolog.Logger
	level  zerolog.Level
}

func (w *forwarderWriter) Write(buf []byte) (n int, err error) {
	n = len(buf)

	if n > 0 && buf[n-1] == '\n' {
		buf = buf[:n-1]
	}

	switch w.level {
	case zerolog.ErrorLevel:
		w.logger.Error().Msg(string(buf))
	case zerolog.WarnLevel:
		w.logger.Warn().Msg(string(buf))
	case zerolog.InfoLevel:
		w.logger.Info().Msg(string(buf))
	default:
		w.logger.Debug().Msg(string(buf))
	}
	return n, nil
}

// New creates a new forwarder for the given pod and configuration.
// Each forwarder gets its own SPDY transport and upgrader to avoid data races
// when multiple forwarders run concurrently.
func New(loc locator.Locator, configuration config.PortForwardConfiguration, client kubernetes.Interface, restCfg *rest.Config) (*Forwarder, error) {
	// Create a dedicated transport AND upgrader for this forwarder.
	// They must come from the same RoundTripperFor() call to be compatible.
	transport, upgrader, err := spdy.RoundTripperFor(restCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPDY transport: %w", err)
	}

	return &Forwarder{
		locator:       loc,
		configuration: configuration,
		client:        client,

		restConfig:    restCfg,

		transport:     transport,
		upgrader:      upgrader,

		retryConfig: DefaultRetryConfig(),
		attempt:     0,
	}, nil
}

// forwarderInfo returns a formatted string with forwarder details for logging.
func (f *Forwarder) forwarderInfo() string {
	return fmt.Sprintf("%s(%s %s) ports:%v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
}

// Start begins the port forwarding loop, attempting to locate and forward to the pod.
// It runs until the context is cancelled.
func (f *Forwarder) Start(ctx context.Context) {
	log := zerolog.Ctx(ctx)

	log.Info().Msgf("START - Forwarder %s", f.forwarderInfo())

	for {
		if ctx.Err() != nil {
			break
		}

		podName, ports, err := f.locator.Locate(ctx)
		if err != nil {
			log.Error().Err(err).Msgf("ERROR - Forwarder %s", f.forwarderInfo())
			f.delayRetry(ctx)
			f.attempt++
			continue
		}

		// Prepare URL
		req := f.client.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(f.configuration.Namespace).
			Name(podName).
			SubResource("portforward")

		// Create the dialer
		dialer := f.createDialer(req.URL(), log)

		// Prepare channel for stop/ready
		stopCh := make(chan struct{})
		readyCh := make(chan struct{})
		doneCh := make(chan struct{})
		errCh := make(chan error)

		outWriter := &forwarderWriter{logger: log, level: zerolog.InfoLevel}
		errWriter := &forwarderWriter{logger: log, level: zerolog.ErrorLevel}

		fw, err := portforward.New(dialer, ports, stopCh, readyCh, outWriter, errWriter)
		if err != nil {
			log.Error().Err(err).Msgf("ERROR - Forwarder %s", f.forwarderInfo())
			f.delayRetry(ctx)
			f.attempt++
			continue
		}

		// Stop the forwarder when context canceled
		go func(stop chan struct{}) {
			select {
			case <-ctx.Done():
			case <-doneCh:
			}
			close(stop)
		}(stopCh)

		// Start forwards
		go func() {
			errCh <- fw.ForwardPorts()
			close(doneCh)
		}()

		select {
		case <-readyCh:
			log.Info().Msgf("READY - Forwarder %s", f.forwarderInfo())
			f.attempt = 0
		case err = <-errCh:
			log.Error().Err(err).Msgf("ERROR - Forwarder %s", f.forwarderInfo())
			f.delayRetry(ctx)
			f.attempt++
			continue
		}

		err = <-errCh

		log.Error().Err(err).Msgf("ERROR - Forwarder %s", f.forwarderInfo())
		f.delayRetry(ctx)
		f.attempt++
	}

	log.Info().Msgf("STOP Forwarder %s", f.forwarderInfo())
}

// calculateBackoff computes exponential backoff with optional jitter.
// Formula: initialDelay * (multiplier ^ attempt), capped at maxDelay
func (f *Forwarder) calculateBackoff() time.Duration {
	delay := f.retryConfig.InitialDelay * time.Duration(math.Pow(f.retryConfig.Multiplier, float64(f.attempt)))

	if delay > f.retryConfig.MaxDelay {
		delay = f.retryConfig.MaxDelay
	}

	if f.retryConfig.Jitter {
		// Add jitter: ±10% randomization
		jitterAmount := delay / 10
		jitterRange := rand.Int63n(int64(2 * jitterAmount))
		delay = delay - jitterAmount + time.Duration(jitterRange)
	}

	return delay
}

// delayRetry pauses before retrying with exponential backoff, respecting context cancellation.
func (f *Forwarder) delayRetry(ctx context.Context) {
	delay := f.calculateBackoff()
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}

// createDialer creates a port-forward dialer with WebSocket primary and SPDY fallback.
// This helps resolve connection stability issues on some Kubernetes clusters.
func (f *Forwarder) createDialer(forwardURL *url.URL, log *zerolog.Logger) httpstream.Dialer {
	// Create the standard SPDY dialer as fallback
	spdyDialer := spdy.NewDialer(f.upgrader, &http.Client{Transport: f.transport}, "POST", forwardURL)

	// Try to create the WebSocket tunneling dialer
	tunnelingDialer, err := portforward.NewSPDYOverWebsocketDialer(forwardURL, f.restConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create WebSocket dialer, using SPDY only")
		return spdyDialer
	}

	log.Debug().Msg("WebSocket port-forward enabled")

	// Use WebSocket as primary with SPDY as fallback
	return portforward.NewFallbackDialer(
		tunnelingDialer,
		spdyDialer,
		func(err error) bool {
			return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
		},
	)
}

// Config returns the port forward configuration for this forwarder.
func (f *Forwarder) Config() config.PortForwardConfiguration {
	return f.configuration
}
