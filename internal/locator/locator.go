package locator

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
)

// Locator is the interface for discovering pods or services in Kubernetes.
type Locator interface {
	// Locate returns the pod name and ports for port forwarding.
	Locate(ctx context.Context) (string, []string, error)
}

// BuildLocator creates the appropriate locator based on the resource string.
// Supported formats:
// - "pod-name" - direct pod reference
// - "svc/service-name" or "service/service-name" - service reference
// - "dep/deployment-name" or "deployment/deployment-name" - deployment reference
// - "sts/statefulset-name" or "statefulset/statefulset-name" - statefulset reference
// - "ds/daemonset-name" or "daemonset/daemonset-name" - daemonset reference
func BuildLocator(resource string, namespace string, ports []string, client kubernetes.Interface) (Locator, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}

	parts := strings.Split(resource, "/")

	if len(parts) == 1 {
		// No prefix: treat as direct pod reference
		return NewPodLocator(resource, namespace, ports, client)
	} else if len(parts) == 2 {
		prefix := parts[0]
		name := parts[1]

		// Service locator
		if prefix == "svc" || prefix == "service" || prefix == "services" {
			return NewServiceLocator(name, namespace, ports, client)
		}

		// Deployment locator
		if prefix == "dep" || prefix == "deployment" || prefix == "deployments" {
			return NewSelectorBasedLocator("deployment", name, namespace, ports, client)
		}

		// StatefulSet locator
		if prefix == "sts" || prefix == "statefulset" || prefix == "statefulsets" {
			return NewSelectorBasedLocator("statefulset", name, namespace, ports, client)
		}

		// DaemonSet locator
		if prefix == "ds" || prefix == "daemonset" || prefix == "daemonsets" {
			return NewSelectorBasedLocator("daemonset", name, namespace, ports, client)
		}

		return nil, fmt.Errorf("unsupported resource type: %s (supported: pod, svc/service, dep/deployment, sts/statefulset, ds/daemonset)", prefix)
	} else {
		return nil, fmt.Errorf("invalid resource format: %s (use 'pod-name', 'svc/service-name', 'dep/deployment-name', etc)", resource)
	}
}
