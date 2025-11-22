package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadConfigurationValid tests loading a valid configuration
func TestReadConfigurationValid(t *testing.T) {
	cfg, err := ReadConfiguration("testdata/valid.cue")

	require.NoError(t, err)
	assert.Equal(t, 2, len(cfg.Forwards))
	assert.Equal(t, "api", cfg.Forwards[0].Name)
	assert.Equal(t, "default", cfg.Forwards[0].Namespace)
	assert.Equal(t, "api-server", cfg.Forwards[0].Resource)
	assert.Equal(t, []string{"8080:8080", "9000:9000"}, cfg.Forwards[0].Ports)
	assert.Equal(t, "database", cfg.Forwards[1].Name)
	assert.Equal(t, []string{"5432"}, cfg.Forwards[1].Ports)
}

// TestReadConfigurationPortConflict tests detection of duplicate local ports
func TestReadConfigurationPortConflict(t *testing.T) {
	_, err := ReadConfiguration("testdata/port-conflict.cue")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port conflict")
	assert.Contains(t, err.Error(), "8080")
}

// TestReadConfigurationInvalidPorts tests validation of port ranges
func TestReadConfigurationInvalidPorts(t *testing.T) {
	_, err := ReadConfiguration("testdata/invalid-ports.cue")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
}

// TestReadConfigurationDuplicateNames tests handling of duplicate forward names
// Note: CUE allows duplicate keys and the last one wins (map behavior)
// This is a known limitation - should be caught at config validation level
func TestReadConfigurationDuplicateNames(t *testing.T) {
	t.Skip("CUE allows duplicate keys (last wins). Should add explicit validation for unique names.")
}

// TestPortValidation tests port number range validation
func TestPortValidation(t *testing.T) {
	testCases := []struct {
		name      string
		port      string
		expectErr bool
	}{
		{"valid single port", "8080", false},
		{"valid port mapping", "8080:9000", false},
		{"port 1", "1", false},
		{"port 65535", "65535", false},
		{"port 0", "0", true},
		{"port 65536", "65536", true},
		{"port 99999", "99999", true},
		{"negative port", "-1", true},
		{"non-numeric", "abc", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configStr := `
logs: {
  level: "info"
  pretty: false
}
forwards: [{
  name: "test"
  ports: ["` + tc.port + `"]
  namespace: "default"
  resource: "pod"
}]
`
			tempFile := t.TempDir() + "/test.cue"
			err := writeTestFile(tempFile, configStr)
			require.NoError(t, err)

			_, err = ReadConfiguration(tempFile)

			if tc.expectErr {
				assert.Error(t, err, "expected error for port %s", tc.port)
			} else {
				assert.NoError(t, err, "expected success for port %s", tc.port)
			}
		})
	}
}

// Helper function to write test files
func writeTestFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
