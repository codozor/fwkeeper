package locator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SelectorBasedLocator locates a pod backing a Kubernetes resource with a selector
// (Deployment, StatefulSet, DaemonSet, etc) and returns the first running pod.
type SelectorBasedLocator struct {
	resourceType string // "deployment", "statefulset", "daemonset"
	resourceName string
	namespace    string
	ports        []string
	client       kubernetes.Interface
}

// NewSelectorBasedLocator creates a locator for any resource type with a selector.
func NewSelectorBasedLocator(resourceType string, resourceName string, namespace string, ports []string, client kubernetes.Interface) (*SelectorBasedLocator, error) {
	return &SelectorBasedLocator{
		resourceType: resourceType,
		resourceName: resourceName,
		namespace:    namespace,
		ports:        ports,
		client:       client,
	}, nil
}

// Locate finds a running pod backing the resource and returns its name and ports.
func (l *SelectorBasedLocator) Locate(ctx context.Context) (string, []string, error) {
	// Get the selector based on resource type
	labelSelector, err := l.getSelector(ctx)
	if err != nil {
		return "", []string{}, err
	}

	// List pods matching the selector
	pods, err := l.client.CoreV1().Pods(l.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("failed to list pods for %s %s: %w", l.resourceType, l.resourceName, err)
	}

	// Find the first running pod
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning {
			return p.Name, l.ports, nil
		}
	}

	return "", []string{}, fmt.Errorf("no running pod found for %s %s", l.resourceType, l.resourceName)
}

// getSelector retrieves the label selector for the resource based on its type.
func (l *SelectorBasedLocator) getSelector(ctx context.Context) (labels.Selector, error) {
	switch l.resourceType {
	case "deployment", "dep":
		return l.getDeploymentSelector(ctx)
	case "statefulset", "sts":
		return l.getStatefulSetSelector(ctx)
	case "daemonset", "ds":
		return l.getDaemonSetSelector(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", l.resourceType)
	}
}

// getDeploymentSelector retrieves the selector from a Deployment.
func (l *SelectorBasedLocator) getDeploymentSelector(ctx context.Context) (labels.Selector, error) {
	deployment, err := l.client.AppsV1().Deployments(l.namespace).Get(ctx, l.resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s: %w", l.resourceName, err)
	}

	if deployment.Spec.Selector == nil {
		return nil, fmt.Errorf("deployment %s has no selector", l.resourceName)
	}

	return labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector(), nil
}

// getStatefulSetSelector retrieves the selector from a StatefulSet.
func (l *SelectorBasedLocator) getStatefulSetSelector(ctx context.Context) (labels.Selector, error) {
	statefulSet, err := l.client.AppsV1().StatefulSets(l.namespace).Get(ctx, l.resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulset %s: %w", l.resourceName, err)
	}

	if statefulSet.Spec.Selector == nil {
		return nil, fmt.Errorf("statefulset %s has no selector", l.resourceName)
	}

	return labels.Set(statefulSet.Spec.Selector.MatchLabels).AsSelector(), nil
}

// getDaemonSetSelector retrieves the selector from a DaemonSet.
func (l *SelectorBasedLocator) getDaemonSetSelector(ctx context.Context) (labels.Selector, error) {
	daemonSet, err := l.client.AppsV1().DaemonSets(l.namespace).Get(ctx, l.resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonset %s: %w", l.resourceName, err)
	}

	if daemonSet.Spec.Selector == nil {
		return nil, fmt.Errorf("daemonset %s has no selector", l.resourceName)
	}

	return labels.Set(daemonSet.Spec.Selector.MatchLabels).AsSelector(), nil
}
