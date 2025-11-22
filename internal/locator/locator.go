package locator

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Locator is the interface for discovering pods or services in Kubernetes.
type Locator interface {
	// Locate returns the pod name and ports for port forwarding.
	Locate(ctx context.Context) (string, []string, error)
}

// CoreV1Client is the subset of CoreV1Interface used by locators.
type CoreV1Client interface {
	Pods(namespace string) PodClient
	Services(namespace string) ServiceClient
}

// PodClient is the subset of PodInterface used by locators.
type PodClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
}

// ServiceClient is the subset of ServiceInterface used by locators.
type ServiceClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Service, error)
}

// AppsV1Client is the subset of AppsV1Interface used by locators.
type AppsV1Client interface {
	Deployments(namespace string) ResourceClient
	StatefulSets(namespace string) ResourceClient
	DaemonSets(namespace string) ResourceClient
}

// ResourceClient is the subset of Deployment/StatefulSet/DaemonSet interfaces used by locators.
type ResourceClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error)
}

// KubernetesClient is a minimal interface for the kubernetes client used by locators.
// This allows for easier mocking in tests.
type KubernetesClient interface {
	CoreV1() CoreV1Client
	AppsV1() AppsV1Client
}


// BuildLocator creates the appropriate locator based on the resource string.
// Supported formats:
// - "pod-name" - direct pod reference
// - "svc/service-name" or "service/service-name" - service reference
// - "dep/deployment-name" or "deployment/deployment-name" - deployment reference
// - "sts/statefulset-name" or "statefulset/statefulset-name" - statefulset reference
// - "ds/daemonset-name" or "daemonset/daemonset-name" - daemonset reference
func BuildLocator(resource string, namespace string, ports []string, client interface{}) (Locator, error) {
	kubeClient, ok := client.(KubernetesClient)
	if !ok {
		return nil, fmt.Errorf("client does not implement KubernetesClient interface")
	}

	parts := strings.Split(resource, "/")

	if len(parts) == 1 {
		// No prefix: treat as direct pod reference
		return NewPodLocator(resource, namespace, ports, kubeClient)
	} else if len(parts) == 2 {
		prefix := parts[0]
		name := parts[1]

		// Service locator
		if prefix == "svc" || prefix == "service" || prefix == "services" {
			return NewServiceLocator(name, namespace, ports, kubeClient)
		}

		// Deployment locator
		if prefix == "dep" || prefix == "deployment" || prefix == "deployments" {
			return NewSelectorBasedLocator("deployment", name, namespace, ports, kubeClient)
		}

		// StatefulSet locator
		if prefix == "sts" || prefix == "statefulset" || prefix == "statefulsets" {
			return NewSelectorBasedLocator("statefulset", name, namespace, ports, kubeClient)
		}

		// DaemonSet locator
		if prefix == "ds" || prefix == "daemonset" || prefix == "daemonsets" {
			return NewSelectorBasedLocator("daemonset", name, namespace, ports, kubeClient)
		}

		return nil, fmt.Errorf("unsupported resource type: %s (supported: pod, svc/service, dep/deployment, sts/statefulset, ds/daemonset)", prefix)
	} else {
		return nil, fmt.Errorf("invalid resource format: %s (use 'pod-name', 'svc/service-name', 'dep/deployment-name', etc)", resource)
	}
}
