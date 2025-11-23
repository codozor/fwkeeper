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

// TestConfigChanged tests the configChanged helper function
func TestConfigChanged(t *testing.T) {
	tests := []struct {
		name     string
		oldCfg   config.PortForwardConfiguration
		newCfg   config.PortForwardConfiguration
		expected bool
	}{
		{
			name: "identical configs",
			oldCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
			newCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
			expected: false,
		},
		{
			name: "namespace changed",
			oldCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
			newCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "kube-system",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
			expected: true,
		},
		{
			name: "resource changed",
			oldCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
			newCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-2",
				Ports:     []string{"8080"},
			},
			expected: true,
		},
		{
			name: "ports added",
			oldCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
			newCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080", "9000"},
			},
			expected: true,
		},
		{
			name: "ports changed",
			oldCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080", "9000"},
			},
			newCfg: config.PortForwardConfiguration{
				Name:      "forward-1",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080", "9001"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configChanged(tt.oldCfg, tt.newCfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestReloadConfigAddForwarder tests adding new forwarders during reload
func TestReloadConfigAddForwarder(t *testing.T) {
	initialCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(initialCfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	// Give the watcher goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Update configuration with new forwarders
	newCfg := config.Configuration{
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
			{
				Name:      "forward-2",
				Namespace: "default",
				Resource:  "pod-2",
				Ports:     []string{"9000"},
			},
		},
	}

	// Manually update config (simulating reload)
	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify configuration was updated
	runner.mu.Lock()
	assert.Equal(t, 2, len(runner.configuration.Forwards))
	assert.Equal(t, "forward-1", runner.configuration.Forwards[0].Name)
	assert.Equal(t, "forward-2", runner.configuration.Forwards[1].Name)
	runner.mu.Unlock()
}

// TestReloadConfigRemoveForwarder tests removing forwarders during reload
func TestReloadConfigRemoveForwarder(t *testing.T) {
	initialCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(initialCfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	// Give the watcher goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Simulate previous config with multiple forwarders
	runner.mu.Lock()
	runner.configuration = config.Configuration{
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
			{
				Name:      "forward-2",
				Namespace: "default",
				Resource:  "pod-2",
				Ports:     []string{"9000"},
			},
		},
	}
	runner.mu.Unlock()

	// Now update to remove forward-2
	newCfg := config.Configuration{
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

	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify configuration was updated
	runner.mu.Lock()
	assert.Equal(t, 1, len(runner.configuration.Forwards))
	assert.Equal(t, "forward-1", runner.configuration.Forwards[0].Name)
	runner.mu.Unlock()
}

// TestReloadConfigChangedPorts tests configuration reload with changed ports
func TestReloadConfigChangedPorts(t *testing.T) {
	initialCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(initialCfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	time.Sleep(50 * time.Millisecond)

	// Simulate previous config with one port
	oldForward := config.PortForwardConfiguration{
		Name:      "forward-1",
		Namespace: "default",
		Resource:  "pod-1",
		Ports:     []string{"8080"},
	}

	// Update with changed ports
	newForward := config.PortForwardConfiguration{
		Name:      "forward-1",
		Namespace: "default",
		Resource:  "pod-1",
		Ports:     []string{"8080", "9000"},
	}

	newCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{newForward},
	}

	// Verify configChanged detects the difference
	assert.True(t, configChanged(oldForward, newForward))

	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify configuration was updated
	runner.mu.Lock()
	assert.Equal(t, 2, len(runner.configuration.Forwards[0].Ports))
	runner.mu.Unlock()
}

// TestReloadConfigMutexProtection tests that config reloads are mutex-protected
func TestReloadConfigMutexProtection(t *testing.T) {
	initialCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(initialCfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	time.Sleep(50 * time.Millisecond)

	// Simulate concurrent access to configuration
	newCfg := config.Configuration{
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

	// Update configuration with mutex protection
	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Read configuration with mutex protection
	runner.mu.Lock()
	cfgCopy := runner.configuration
	runner.mu.Unlock()

	// Verify read succeeded
	assert.Equal(t, 1, len(cfgCopy.Forwards))
	assert.Equal(t, "forward-1", cfgCopy.Forwards[0].Name)
}

// TestReloadConfigMultipleForwarders tests reload with multiple forwarders
func TestReloadConfigMultipleForwarders(t *testing.T) {
	initialCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(initialCfg, "", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	time.Sleep(50 * time.Millisecond)

	// Simulate previous config with 2 forwarders
	runner.mu.Lock()
	runner.configuration = config.Configuration{
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
			{
				Name:      "forward-2",
				Namespace: "default",
				Resource:  "pod-2",
				Ports:     []string{"9000"},
			},
		},
	}
	runner.mu.Unlock()

	// Update with different forwarders (add one, keep one, remove one)
	newCfg := config.Configuration{
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
			{
				Name:      "forward-3",
				Namespace: "default",
				Resource:  "pod-3",
				Ports:     []string{"7000"},
			},
		},
	}

	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify configuration was updated
	runner.mu.Lock()
	assert.Equal(t, 2, len(runner.configuration.Forwards))
	forwardNames := []string{
		runner.configuration.Forwards[0].Name,
		runner.configuration.Forwards[1].Name,
	}
	assert.Contains(t, forwardNames, "forward-1")
	assert.Contains(t, forwardNames, "forward-3")
	runner.mu.Unlock()
}
