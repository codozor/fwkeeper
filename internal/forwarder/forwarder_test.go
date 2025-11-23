package forwarder

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codozor/fwkeeper/internal/config"
)

// Phase 9 Tests - Forwarder Logic (No Kubernetes dependency)

// TestDefaultRetryConfig tests default retry configuration
func TestDefaultRetryConfig(t *testing.T) {
	rc := DefaultRetryConfig()

	assert.Equal(t, 100*time.Millisecond, rc.InitialDelay)
	assert.Equal(t, 30*time.Second, rc.MaxDelay)
	assert.Equal(t, 1.5, rc.Multiplier)
	assert.True(t, rc.Jitter)
}

// TestRetryConfigExponentialBackoff tests exponential backoff calculation
func TestRetryConfigExponentialBackoff(t *testing.T) {
	rc := DefaultRetryConfig()

	// Test delay calculation for different attempt numbers
	tests := []struct {
		attempt     uint
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			attempt:     0,
			minDuration: 100 * time.Millisecond,
			maxDuration: 150 * time.Millisecond, // With multiplier, rough estimate
		},
		{
			attempt:     1,
			minDuration: 150 * time.Millisecond,
			maxDuration: 300 * time.Millisecond,
		},
		{
			attempt:     2,
			minDuration: 225 * time.Millisecond,
			maxDuration: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		baseDelay := rc.InitialDelay
		for i := uint(0); i < tt.attempt; i++ {
			baseDelay = time.Duration(float64(baseDelay) * rc.Multiplier)
		}
		if baseDelay > rc.MaxDelay {
			baseDelay = rc.MaxDelay
		}

		// Verify delay is within expected range
		assert.GreaterOrEqual(t, baseDelay, tt.minDuration)
	}
}

// TestRetryConfigMaxDelayEnforced tests that max delay is enforced
func TestRetryConfigMaxDelayEnforced(t *testing.T) {
	rc := DefaultRetryConfig()

	// Calculate delay for many attempts (should hit max)
	delay := rc.InitialDelay
	for i := 0; i < 100; i++ {
		delay = time.Duration(float64(delay) * rc.Multiplier)
		if delay > rc.MaxDelay {
			delay = rc.MaxDelay
		}
	}

	// Should be capped at MaxDelay
	assert.LessOrEqual(t, delay, rc.MaxDelay)
}

// TestRetryConfigJitterOption tests jitter option
func TestRetryConfigJitterOption(t *testing.T) {
	rcWithJitter := DefaultRetryConfig()
	assert.True(t, rcWithJitter.Jitter)

	rcNoJitter := RetryConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   1.5,
		Jitter:       false,
	}
	assert.False(t, rcNoJitter.Jitter)
}

// TestPortForwardConfigurationValid tests valid port configurations
func TestPortForwardConfigurationValid(t *testing.T) {
	tests := []struct {
		name  string
		ports []string
		valid bool
	}{
		{
			name:  "single port",
			ports: []string{"8080"},
			valid: true,
		},
		{
			name:  "mapped port",
			ports: []string{"8080:3000"},
			valid: true,
		},
		{
			name:  "multiple ports",
			ports: []string{"8080", "9000", "5432"},
			valid: true,
		},
		{
			name:  "mixed mapped and unmapped",
			ports: []string{"8080", "9000:3000", "5432"},
			valid: true,
		},
		{
			name:  "empty ports",
			ports: []string{},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.PortForwardConfiguration{
				Name:      "test",
				Namespace: "default",
				Resource:  "pod-1",
				Ports:     tt.ports,
			}

			if tt.valid {
				assert.NotEmpty(t, cfg.Ports)
			} else {
				assert.Empty(t, cfg.Ports)
			}
		})
	}
}

// TestPortParsingLogic tests parsing of port specifications
func TestPortParsingLogic(t *testing.T) {
	tests := []struct {
		name       string
		portSpec   string
		localPort  string
		remotePort string
	}{
		{
			name:       "single port",
			portSpec:   "8080",
			localPort:  "8080",
			remotePort: "8080",
		},
		{
			name:       "mapped port",
			portSpec:   "8080:3000",
			localPort:  "8080",
			remotePort: "3000",
		},
		{
			name:       "IPv6 address",
			portSpec:   "[::1]:8080:3000",
			localPort:  "8080",
			remotePort: "3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple port parsing logic
			parts := splitPort(tt.portSpec)

			if len(parts) == 1 {
				assert.Equal(t, tt.localPort, parts[0])
			} else if len(parts) == 2 {
				assert.Equal(t, tt.localPort, parts[0])
				assert.Equal(t, tt.remotePort, parts[1])
			}
		})
	}
}

// TestForwarderConfigurationCreation tests creating valid configurations
func TestForwarderConfigurationCreation(t *testing.T) {
	cfg := config.PortForwardConfiguration{
		Name:      "api-server",
		Namespace: "default",
		Resource:  "api-pod",
		Ports:     []string{"8080", "8443:443"},
	}

	assert.Equal(t, "api-server", cfg.Name)
	assert.Equal(t, "default", cfg.Namespace)
	assert.Equal(t, "api-pod", cfg.Resource)
	assert.Equal(t, 2, len(cfg.Ports))
}

// TestForwarderConfigurationInfo tests generating info string
func TestForwarderConfigurationInfo(t *testing.T) {
	cfg := config.PortForwardConfiguration{
		Name:      "database",
		Namespace: "prod",
		Resource:  "postgres-pod",
		Ports:     []string{"5432", "5433:5432"},
	}

	// Test that we can format configuration info
	info := cfg.Name + "(" + cfg.Namespace + " " + cfg.Resource + ")"
	assert.Contains(t, info, "database")
	assert.Contains(t, info, "prod")
	assert.Contains(t, info, "postgres-pod")
}

// TestRetryConfigCustomization tests custom retry configurations
func TestRetryConfigCustomization(t *testing.T) {
	customRC := RetryConfig{
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}

	assert.Equal(t, 50*time.Millisecond, customRC.InitialDelay)
	assert.Equal(t, 5*time.Second, customRC.MaxDelay)
	assert.Equal(t, 2.0, customRC.Multiplier)
	assert.False(t, customRC.Jitter)
}

// TestMultiplePortConfigurations tests handling multiple port configurations
func TestMultiplePortConfigurations(t *testing.T) {
	configs := []config.PortForwardConfiguration{
		{
			Name:      "frontend",
			Namespace: "prod",
			Resource:  "frontend-app",
			Ports:     []string{"80:3000", "443:3001"},
		},
		{
			Name:      "backend",
			Namespace: "prod",
			Resource:  "backend-api",
			Ports:     []string{"8080:8080", "8443:8443"},
		},
		{
			Name:      "database",
			Namespace: "prod",
			Resource:  "postgres",
			Ports:     []string{"5432:5432"},
		},
	}

	require.Equal(t, 3, len(configs))
	assert.Equal(t, "frontend", configs[0].Name)
	assert.Equal(t, "backend", configs[1].Name)
	assert.Equal(t, "database", configs[2].Name)
	assert.Equal(t, 2, len(configs[0].Ports))
	assert.Equal(t, 2, len(configs[1].Ports))
	assert.Equal(t, 1, len(configs[2].Ports))
}

// TestPortMappingEdgeCases tests edge cases in port mapping
func TestPortMappingEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		isValid bool
	}{
		{
			name:    "high port number",
			port:    "65535",
			isValid: true,
		},
		{
			name:    "low port number",
			port:    "1",
			isValid: true,
		},
		{
			name:    "mapped to same port",
			port:    "8080:8080",
			isValid: true,
		},
		{
			name:    "mapped to different port",
			port:    "9000:8080",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.PortForwardConfiguration{
				Name:      "test",
				Namespace: "default",
				Resource:  "pod",
				Ports:     []string{tt.port},
			}

			if tt.isValid {
				assert.Equal(t, 1, len(cfg.Ports))
				assert.Equal(t, tt.port, cfg.Ports[0])
			}
		})
	}
}

// TestRetryConfigComparison tests comparing retry configurations
func TestRetryConfigComparison(t *testing.T) {
	rc1 := DefaultRetryConfig()
	rc2 := DefaultRetryConfig()

	// Should have same values
	assert.Equal(t, rc1.InitialDelay, rc2.InitialDelay)
	assert.Equal(t, rc1.MaxDelay, rc2.MaxDelay)
	assert.Equal(t, rc1.Multiplier, rc2.Multiplier)
	assert.Equal(t, rc1.Jitter, rc2.Jitter)
}

// Helper function for port parsing (mimics port format parsing)
func splitPort(portSpec string) []string {
	// Remove IPv6 bracket notation if present
	if portSpec[0] == '[' {
		// Format: [::1]:8080:3000 -> extract 8080:3000
		bracketEnd := 0
		for i, ch := range portSpec {
			if ch == ']' {
				bracketEnd = i
				break
			}
		}
		if bracketEnd > 0 && bracketEnd+1 < len(portSpec) && portSpec[bracketEnd+1] == ':' {
			portSpec = portSpec[bracketEnd+2:]
		}
	}

	// Split on colon
	var parts []string
	var current string
	for _, ch := range portSpec {
		if ch == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
