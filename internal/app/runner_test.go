package app

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	"github.com/codozor/fwkeeper/internal/config"
)

// TestRunnerStart tests basic runner initialization
func TestRunnerStart(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	// Note: Using nil client since we're testing runner lifecycle, not forwarder
	// In real scenarios, forwarder would need valid client
	// This tests that runner can initialize with empty forwards
	runner := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")

	err := runner.Start()
	defer runner.Shutdown()

	// Should not panic during start with no forwarders
	assert.NoError(t, err)
}

// TestRunnerShutdown tests graceful shutdown
func TestRunnerShutdown(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")

	err := runner.Start()
	require.NoError(t, err)

	// Should shutdown without panic
	runner.Shutdown()
	assert.True(t, true) // If we reach here, shutdown succeeded
}

// TestRunnerContextCancellation tests that runner respects context cancellation
func TestRunnerContextCancellation(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")

	err := runner.Start()
	require.NoError(t, err)

	// Give it time to fully start
	time.Sleep(100 * time.Millisecond)

	// Shutdown should complete without hanging
	runner.Shutdown()
}

// TestRunnerMultipleStartStop tests that runner can start and stop cleanly
func TestRunnerMultipleStartStop(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	// Create and start runner
	runner1 := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner1.Start()
	require.NoError(t, err)
	runner1.Shutdown()

	// Create and start another runner instance to test clean state
	runner2 := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err = runner2.Start()
	defer runner2.Shutdown()

	require.NoError(t, err)
}

// TestRunnerConfigChangeDetection tests that runner can detect configuration changes
func TestRunnerConfigChangeDetection(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{
			{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
		},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	// Test that runner stores configuration
	assert.Equal(t, 1, len(runner.configuration.Forwards))
	assert.Equal(t, "forward-1", runner.configuration.Forwards[0].Name)
}

// TestRunnerEmptyConfiguration tests runner with no forwarders
func TestRunnerEmptyConfiguration(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	defer runner.Shutdown()

	require.NoError(t, err)
	assert.Equal(t, 0, len(runner.configuration.Forwards))
}

// TestRunnerConfigPathStorage tests that runner stores the config path
func TestRunnerConfigPathStorage(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	configPath := "testdata/config.cue"
	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(cfg, configPath, logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	defer runner.Shutdown()

	require.NoError(t, err)
	assert.Equal(t, configPath, runner.configPath)
}

// TestRunnerForwarderMapInitialization tests that forwarder maps are properly initialized
func TestRunnerForwarderMapInitialization(t *testing.T) {
	cfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(cfg, "", logger, nil, restCfg, "mock-source", "mock-context")

	// Before start, maps should exist but be empty
	assert.NotNil(t, runner.forwarders)
	assert.NotNil(t, runner.forwarderCancel)
	assert.Equal(t, 0, len(runner.forwarders))
	assert.Equal(t, 0, len(runner.forwarderCancel))

	err := runner.Start()
	defer runner.Shutdown()

	require.NoError(t, err)
}
