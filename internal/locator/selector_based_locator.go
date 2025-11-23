package locator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		// Classify API errors
		if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
			return "", []string{}, NewAPITransientError(fmt.Sprintf("API timeout listing pods for %s %s", l.resourceType, l.resourceName), err)
		}
		if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
			return "", []string{}, NewPermissionDeniedError("list", fmt.Sprintf("pods for %s %s", l.resourceType, l.resourceName), err)
		}
		return "", []string{}, NewAPITransientError(fmt.Sprintf("failed to list pods for %s %s", l.resourceType, l.resourceName), err)
	}

	// Find the first running pod
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning {
			return p.Name, l.ports, nil
		}
	}

	return "", []string{}, &LocateError{
		Type:    ErrorTypeNoPodAvailable,
		Message: fmt.Sprintf("no running pod found for %s %s", l.resourceType, l.resourceName),
		Err:     nil,
	}
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
		return nil, NewConfigInvalidError(fmt.Sprintf("unsupported resource type: %s", l.resourceType), nil)
	}
}

// getDeploymentSelector retrieves the selector from a Deployment.
func (l *SelectorBasedLocator) getDeploymentSelector(ctx context.Context) (labels.Selector, error) {
	deployment, err := l.client.AppsV1().Deployments(l.namespace).Get(ctx, l.resourceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, NewResourceNotFoundError("deployment", l.resourceName, err)
		}
		if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
			return nil, NewAPITransientError(fmt.Sprintf("API timeout getting deployment %s", l.resourceName), err)
		}
		return nil, NewAPITransientError(fmt.Sprintf("failed to get deployment %s", l.resourceName), err)
	}

	if deployment.Spec.Selector == nil {
		return nil, NewConfigInvalidError(fmt.Sprintf("deployment %s has no selector", l.resourceName), nil)
	}

	return labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector(), nil
}

// getStatefulSetSelector retrieves the selector from a StatefulSet.
func (l *SelectorBasedLocator) getStatefulSetSelector(ctx context.Context) (labels.Selector, error) {
	statefulSet, err := l.client.AppsV1().StatefulSets(l.namespace).Get(ctx, l.resourceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, NewResourceNotFoundError("statefulset", l.resourceName, err)
		}
		if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
			return nil, NewAPITransientError(fmt.Sprintf("API timeout getting statefulset %s", l.resourceName), err)
		}
		return nil, NewAPITransientError(fmt.Sprintf("failed to get statefulset %s", l.resourceName), err)
	}

	if statefulSet.Spec.Selector == nil {
		return nil, NewConfigInvalidError(fmt.Sprintf("statefulset %s has no selector", l.resourceName), nil)
	}

	return labels.Set(statefulSet.Spec.Selector.MatchLabels).AsSelector(), nil
}

// getDaemonSetSelector retrieves the selector from a DaemonSet.
func (l *SelectorBasedLocator) getDaemonSetSelector(ctx context.Context) (labels.Selector, error) {
	daemonSet, err := l.client.AppsV1().DaemonSets(l.namespace).Get(ctx, l.resourceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, NewResourceNotFoundError("daemonset", l.resourceName, err)
		}
		if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
			return nil, NewAPITransientError(fmt.Sprintf("API timeout getting daemonset %s", l.resourceName), err)
		}
		return nil, NewAPITransientError(fmt.Sprintf("failed to get daemonset %s", l.resourceName), err)
	}

	if daemonSet.Spec.Selector == nil {
		return nil, NewConfigInvalidError(fmt.Sprintf("daemonset %s has no selector", l.resourceName), nil)
	}

	return labels.Set(daemonSet.Spec.Selector.MatchLabels).AsSelector(), nil
}
