package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kubeconfigEntry describes one context with its own cluster and user.
type kubeconfigEntry struct {
	context, cluster, user, server string
}

// writeKubeconfig writes a minimal kubeconfig file and returns its path.
func writeKubeconfig(t *testing.T, name, currentContext string, entries ...kubeconfigEntry) string {
	t.Helper()

	var clusters, users, contexts strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&clusters, "- name: %s\n  cluster:\n    server: %s\n", e.cluster, e.server)
		fmt.Fprintf(&users, "- name: %s\n  user:\n    token: test-token\n", e.user)
		fmt.Fprintf(&contexts, "- name: %s\n  context:\n    cluster: %s\n    user: %s\n", e.context, e.cluster, e.user)
	}

	content := "apiVersion: v1\nkind: Config\n"
	if currentContext != "" {
		content += "current-context: " + currentContext + "\n"
	}
	content += "clusters:\n" + clusters.String() + "users:\n" + users.String() + "contexts:\n" + contexts.String()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func kubeconfigList(paths ...string) string {
	return strings.Join(paths, string(os.PathListSeparator))
}

// Single kubeconfig with several contexts (the most common setup).
func TestNewRestConfigSingleFileContexts(t *testing.T) {
	path := writeKubeconfig(t, "config", "dev",
		kubeconfigEntry{"dev", "dev-cluster", "dev-user", "https://dev.example:6443"},
		kubeconfigEntry{"staging", "staging-cluster", "staging-user", "https://staging.example:6443"},
	)
	t.Setenv("KUBECONFIG", path)

	t.Run("current-context by default", func(t *testing.T) {
		info, err := NewRestConfig(Options{})
		require.NoError(t, err)
		assert.Equal(t, "dev", info.Context)
		assert.Equal(t, "https://dev.example:6443", info.Config.Host)
		assert.Empty(t, info.Warnings)
	})

	t.Run("context override", func(t *testing.T) {
		info, err := NewRestConfig(Options{Context: "staging"})
		require.NoError(t, err)
		assert.Equal(t, "staging", info.Context)
		assert.Equal(t, "https://staging.example:6443", info.Config.Host)
	})

	t.Run("unknown context", func(t *testing.T) {
		_, err := NewRestConfig(Options{Context: "nope"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nope")
	})
}

// KUBECONFIG listing several files must be merged like kubectl does.
func TestNewRestConfigMultipleFiles(t *testing.T) {
	a := writeKubeconfig(t, "a.yaml", "home",
		kubeconfigEntry{"home", "home-cluster", "home-user", "https://home.example:6443"})
	b := writeKubeconfig(t, "b.yaml", "",
		kubeconfigEntry{"ovh", "ovh-cluster", "ovh-user", "https://ovh.example:6443"})
	t.Setenv("KUBECONFIG", kubeconfigList(a, b))

	info, err := NewRestConfig(Options{})
	require.NoError(t, err)
	assert.Equal(t, "home", info.Context)
	assert.Equal(t, "https://home.example:6443", info.Config.Host)
	assert.Contains(t, info.Source, "KUBECONFIG=")

	info, err = NewRestConfig(Options{Context: "ovh"})
	require.NoError(t, err)
	assert.Equal(t, "https://ovh.example:6443", info.Config.Host)
	assert.Empty(t, info.Warnings)
}

// Missing files in KUBECONFIG are ignored, like kubectl.
func TestNewRestConfigMissingFileIgnored(t *testing.T) {
	a := writeKubeconfig(t, "a.yaml", "home",
		kubeconfigEntry{"home", "home-cluster", "home-user", "https://home.example:6443"})
	t.Setenv("KUBECONFIG", kubeconfigList(filepath.Join(t.TempDir(), "missing.yaml"), a))

	info, err := NewRestConfig(Options{})
	require.NoError(t, err)
	assert.Equal(t, "https://home.example:6443", info.Config.Host)
}

// --kubeconfig wins over KUBECONFIG, without merging.
func TestNewRestConfigExplicitPath(t *testing.T) {
	env := writeKubeconfig(t, "env.yaml", "env",
		kubeconfigEntry{"env", "env-cluster", "env-user", "https://env.example:6443"})
	explicit := writeKubeconfig(t, "explicit.yaml", "explicit",
		kubeconfigEntry{"explicit", "explicit-cluster", "explicit-user", "https://explicit.example:6443"})
	t.Setenv("KUBECONFIG", env)

	info, err := NewRestConfig(Options{Kubeconfig: explicit})
	require.NoError(t, err)
	assert.Equal(t, "explicit", info.Context)
	assert.Equal(t, "https://explicit.example:6443", info.Config.Host)
	assert.Equal(t, explicit, info.Source)

	_, err = NewRestConfig(Options{Kubeconfig: explicit, Context: "env"})
	assert.Error(t, err, "contexts from KUBECONFIG must not be merged with an explicit kubeconfig")
}

// Duplicate names across files: the first file wins and a warning is reported.
func TestNewRestConfigDuplicateWarnings(t *testing.T) {
	a := writeKubeconfig(t, "a.yaml", "default",
		kubeconfigEntry{"default", "default", "default", "https://192.168.1.10:6443"})
	b := writeKubeconfig(t, "b.yaml", "",
		kubeconfigEntry{"default", "default", "default", "https://10.0.0.5:6443"})
	t.Setenv("KUBECONFIG", kubeconfigList(a, b))

	info, err := NewRestConfig(Options{})
	require.NoError(t, err)
	assert.Equal(t, "https://192.168.1.10:6443", info.Config.Host, "first file must win")
	require.Len(t, info.Warnings, 3)
	assert.Contains(t, info.Warnings[0], `context "default"`)
	assert.Contains(t, info.Warnings[1], `cluster "default"`)
	assert.Contains(t, info.Warnings[2], `user "default"`)
	for _, w := range info.Warnings {
		assert.True(t, strings.HasSuffix(w, "using the one from "+a), w)
	}
}

// A context in file b referencing a cluster name also defined in file a silently
// resolves to a's cluster: this is the trap the warning is about.
func TestNewRestConfigDuplicateClusterOnly(t *testing.T) {
	a := writeKubeconfig(t, "a.yaml", "home",
		kubeconfigEntry{"home", "default", "home-user", "https://192.168.1.10:6443"})
	b := writeKubeconfig(t, "b.yaml", "",
		kubeconfigEntry{"ovh", "default", "ovh-user", "https://10.0.0.5:6443"})
	t.Setenv("KUBECONFIG", kubeconfigList(a, b))

	info, err := NewRestConfig(Options{Context: "ovh"})
	require.NoError(t, err)
	assert.Equal(t, "https://192.168.1.10:6443", info.Config.Host)
	require.Len(t, info.Warnings, 1)
	assert.Contains(t, info.Warnings[0], `cluster "default"`)
}
