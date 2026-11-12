package kubernetes

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Options selects which kubeconfig and context to use.
// Empty fields fall back to kubectl's defaults (KUBECONFIG, ~/.kube/config, current-context).
type Options struct {
	Kubeconfig string // explicit kubeconfig file, takes precedence over KUBECONFIG
	Context    string // context name, overrides the kubeconfig current-context
}

// RestConfigInfo contains the REST config and a description of where it comes from.
type RestConfigInfo struct {
	Config   *rest.Config
	Source   string   // Human-readable description of the kubeconfig source
	Context  string   // Context actually used
	Warnings []string // Non-fatal issues detected while loading (e.g. duplicate entries)
}

// NewRestConfig resolves the Kubernetes REST configuration exactly like kubectl does:
//   - opts.Kubeconfig if set (no merging),
//   - otherwise every file listed in KUBECONFIG, merged (first file to define a name wins),
//   - otherwise ~/.kube/config,
//   - otherwise in-cluster configuration.
//
// opts.Context overrides the current-context of the kubeconfig.
func NewRestConfig(opts Options) (RestConfigInfo, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		rules.ExplicitPath = opts.Kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{CurrentContext: opts.Context}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)

	source := describeSource(rules)

	raw, err := cc.RawConfig()
	if err != nil {
		return RestConfigInfo{}, fmt.Errorf("loading kubeconfig from %s: %w", source, err)
	}

	cfg, err := cc.ClientConfig()
	if err != nil {
		return RestConfigInfo{}, fmt.Errorf("building client config from %s: %w", source, err)
	}

	contextName := raw.CurrentContext
	if opts.Context != "" {
		contextName = opts.Context
	}

	// No context in any kubeconfig: client-go fell back to the in-cluster configuration.
	if len(raw.Contexts) == 0 {
		return RestConfigInfo{
			Config:  cfg,
			Source:  "in-cluster (running inside Kubernetes)",
			Context: "in-cluster",
		}, nil
	}

	var warnings []string
	if rules.ExplicitPath == "" {
		warnings = duplicateWarnings(rules.GetLoadingPrecedence(), raw, contextName)
	}

	return RestConfigInfo{
		Config:   cfg,
		Source:   source,
		Context:  contextName,
		Warnings: warnings,
	}, nil
}

// describeSource returns a human-readable description of the kubeconfig files used.
func describeSource(rules *clientcmd.ClientConfigLoadingRules) string {
	if rules.ExplicitPath != "" {
		return rules.ExplicitPath
	}
	if env := os.Getenv(clientcmd.RecommendedConfigPathEnvVar); env != "" {
		return clientcmd.RecommendedConfigPathEnvVar + "=" + env
	}
	return strings.Join(rules.GetLoadingPrecedence(), ", ")
}

// duplicateWarnings reports when the selected context, or the cluster/user it refers to,
// is defined in several kubeconfig files. kubectl silently keeps the first definition,
// which can make a context point to an unexpected cluster.
func duplicateWarnings(files []string, merged clientcmdapi.Config, contextName string) []string {
	if len(files) < 2 {
		return nil
	}

	ctx, ok := merged.Contexts[contextName]
	if !ok {
		return nil
	}

	var ctxFiles, clusterFiles, userFiles []string
	for _, f := range files {
		cfg, err := clientcmd.LoadFromFile(f)
		if err != nil {
			continue // missing or unreadable files are handled (or ignored) by the main loader
		}
		if _, ok := cfg.Contexts[contextName]; ok {
			ctxFiles = append(ctxFiles, f)
		}
		if _, ok := cfg.Clusters[ctx.Cluster]; ok {
			clusterFiles = append(clusterFiles, f)
		}
		if _, ok := cfg.AuthInfos[ctx.AuthInfo]; ok {
			userFiles = append(userFiles, f)
		}
	}

	var warnings []string
	add := func(kind, name string, defined []string) {
		if len(defined) > 1 {
			warnings = append(warnings, fmt.Sprintf("%s %q is defined in %s, using the one from %s",
				kind, name, strings.Join(defined, ", "), defined[0]))
		}
	}
	add("context", contextName, ctxFiles)
	add("cluster", ctx.Cluster, clusterFiles)
	add("user", ctx.AuthInfo, userFiles)

	return warnings
}

// NewClient creates a new Kubernetes client from a REST configuration.
func NewClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}
