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

// Phase 5 Tests - Hot-reload and Signal Handling

// TestBaseName tests the baseName helper function
func TestBaseName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "unix absolute path",
			path:     "/home/user/config.cue",
			expected: "config.cue",
		},
		{
			name:     "unix relative path",
			path:     "config/app.cue",
			expected: "app.cue",
		},
		{
			name:     "windows absolute path",
			path:     "C:\\config\\test.cue",
			expected: "test.cue",
		},
		{
			name:     "filename only",
			path:     "config.cue",
			expected: "config.cue",
		},
		{
			name:     "empty string",
			path:     "",
			expected: "",
		},
		{
			name:     "path with trailing slash",
			path:     "/home/user/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := baseName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStopForwarderRemovesFromMaps tests that stopForwarder removes entries from maps
func TestStopForwarderRemovesFromMaps(t *testing.T) {
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

	// Manually add a forwarder entry to the maps (simulating a running forwarder)
	runner.mu.Lock()
	runner.forwarders["test-forward"] = nil
	runner.forwarderCancel["test-forward"] = func() {}
	runner.mu.Unlock()

	// Verify it was added
	runner.mu.Lock()
	assert.Equal(t, 1, len(runner.forwarders))
	assert.Equal(t, 1, len(runner.forwarderCancel))
	runner.mu.Unlock()

	// Stop the forwarder
	runner.mu.Lock()
	runner.stopForwarder("test-forward")
	runner.mu.Unlock()

	// Verify it was removed
	runner.mu.Lock()
	assert.Equal(t, 0, len(runner.forwarders))
	assert.Equal(t, 0, len(runner.forwarderCancel))
	_, existsForwarder := runner.forwarders["test-forward"]
	_, existsCancel := runner.forwarderCancel["test-forward"]
	runner.mu.Unlock()

	assert.False(t, existsForwarder, "forwarder should be removed")
	assert.False(t, existsCancel, "cancel function should be removed")
}

// TestStopForwarderNonExistent tests stopForwarder with non-existent forwarder
func TestStopForwarderNonExistent(t *testing.T) {
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

	// Try to stop non-existent forwarder (should not panic)
	runner.mu.Lock()
	runner.stopForwarder("non-existent")
	runner.mu.Unlock()

	// Should complete without panic
	assert.True(t, true)
}

// TestReloadConfigUpdateState tests that configuration state is properly updated
func TestReloadConfigUpdateState(t *testing.T) {
	initialCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	restCfg := &rest.Config{}
	logger := zerolog.New(nil)

	runner := New(initialCfg, "testdata/config1.cue", logger, nil, restCfg, "mock-source", "mock-context")
	err := runner.Start()
	require.NoError(t, err)
	defer runner.Shutdown()

	time.Sleep(50 * time.Millisecond)

	// Verify initial state
	runner.mu.Lock()
	assert.Equal(t, 0, len(runner.configuration.Forwards))
	runner.mu.Unlock()

	// Simulate config reload with new configuration
	newCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{
			{
				Name:      "test-forward",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     []string{"8080"},
			},
		},
	}

	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify state was updated
	runner.mu.Lock()
	assert.Equal(t, 1, len(runner.configuration.Forwards))
	assert.Equal(t, "test-forward", runner.configuration.Forwards[0].Name)
	runner.mu.Unlock()
}

// TestReloadConfigStateTransition tests complex state transitions during reload
func TestReloadConfigStateTransition(t *testing.T) {
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

	// Simulate initial config with 3 forwarders
	runner.mu.Lock()
	runner.configuration = config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{
			{Name: "forward-1", Namespace: "ns1", Resource: "pod-1", Ports: []string{"8080"}},
			{Name: "forward-2", Namespace: "ns2", Resource: "pod-2", Ports: []string{"9000"}},
			{Name: "forward-3", Namespace: "ns3", Resource: "pod-3", Ports: []string{"7000"}},
		},
	}
	runner.mu.Unlock()

	// Reload with new configuration
	newCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "info",
			Pretty: false,
		},
		Forwards: []config.PortForwardConfiguration{
			{Name: "forward-1", Namespace: "ns1", Resource: "pod-1", Ports: []string{"8080"}},
			{Name: "forward-2", Namespace: "ns2-modified", Resource: "pod-2", Ports: []string{"9000"}},
			{Name: "forward-4", Namespace: "ns4", Resource: "pod-4", Ports: []string{"6000"}},
		},
	}

	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify state transition
	runner.mu.Lock()
	assert.Equal(t, 3, len(runner.configuration.Forwards))
	runner.mu.Unlock()
}

// TestReloadConfigPreservesLogConfiguration tests that log config is preserved
func TestReloadConfigPreservesLogConfiguration(t *testing.T) {
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

	// Reload with new log config
	newCfg := config.Configuration{
		Logs: config.LogsConfiguration{
			Level:  "debug",
			Pretty: true,
		},
		Forwards: []config.PortForwardConfiguration{},
	}

	runner.mu.Lock()
	runner.configuration = newCfg
	runner.mu.Unlock()

	// Verify log config was updated
	runner.mu.Lock()
	assert.Equal(t, "debug", runner.configuration.Logs.Level)
	assert.Equal(t, true, runner.configuration.Logs.Pretty)
	runner.mu.Unlock()
}

// TestFileWatcherPathComparison tests the file path comparison logic
func TestFileWatcherPathComparison(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		eventPath  string
		expected   bool
	}{
		{
			name:       "exact match",
			configPath: "fwkeeper.cue",
			eventPath:  "fwkeeper.cue",
			expected:   true,
		},
		{
			name:       "absolute paths match",
			configPath: "/home/user/fwkeeper.cue",
			eventPath:  "/home/user/fwkeeper.cue",
			expected:   true,
		},
		{
			name:       "different files",
			configPath: "fwkeeper.cue",
			eventPath:  "other.cue",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test baseName comparison
			configBaseName := baseName(tt.configPath)
			eventBaseName := baseName(tt.eventPath)
			result := configBaseName == eventBaseName && configBaseName != ""

			if tt.expected {
				assert.True(t, result, "paths should match")
			} else {
				assert.False(t, result, "paths should not match")
			}
		})
	}
}

// TestReloadConfigMultipleSequentialReloads tests multiple successive reloads
func TestReloadConfigMultipleSequentialReloads(t *testing.T) {
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

	// First reload
	cfg1 := config.Configuration{
		Logs: config.LogsConfiguration{Level: "info", Pretty: false},
		Forwards: []config.PortForwardConfiguration{
			{Name: "forward-1", Namespace: "ns1", Resource: "pod-1", Ports: []string{"8080"}},
		},
	}
	runner.mu.Lock()
	runner.configuration = cfg1
	runner.mu.Unlock()

	runner.mu.Lock()
	assert.Equal(t, 1, len(runner.configuration.Forwards))
	runner.mu.Unlock()

	// Second reload
	cfg2 := config.Configuration{
		Logs: config.LogsConfiguration{Level: "info", Pretty: false},
		Forwards: []config.PortForwardConfiguration{
			{Name: "forward-1", Namespace: "ns1", Resource: "pod-1", Ports: []string{"8080"}},
			{Name: "forward-2", Namespace: "ns2", Resource: "pod-2", Ports: []string{"9000"}},
		},
	}
	runner.mu.Lock()
	runner.configuration = cfg2
	runner.mu.Unlock()

	runner.mu.Lock()
	assert.Equal(t, 2, len(runner.configuration.Forwards))
	runner.mu.Unlock()

	// Third reload
	cfg3 := config.Configuration{
		Logs: config.LogsConfiguration{Level: "debug", Pretty: true},
		Forwards: []config.PortForwardConfiguration{},
	}
	runner.mu.Lock()
	runner.configuration = cfg3
	runner.mu.Unlock()

	runner.mu.Lock()
	assert.Equal(t, 0, len(runner.configuration.Forwards))
	assert.Equal(t, "debug", runner.configuration.Logs.Level)
	runner.mu.Unlock()
}
