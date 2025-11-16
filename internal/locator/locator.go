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
func BuildLocator(resource string, namespace string, ports []string, client kubernetes.Interface) (Locator, error) {
	parts := strings.Split(resource, "/")

	if len(parts) == 1 {
		return NewPodLocator(resource, namespace, ports, client)
	} else if len(parts) == 2 {
		if parts[0] == "svc" || parts[0] == "service" || parts[0] == "services" {
			return NewServiceLocator(parts[1], namespace, ports, client)
		} else {
			return nil, fmt.Errorf("unsupported resource type: %s (supported: pod, svc, service)", parts[0])
		}
	} else {
		return nil, fmt.Errorf("invalid resource format: %s (use 'pod-name' or 'svc/service-name')", resource)
	}
}
