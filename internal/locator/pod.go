package locator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodLocator locates a specific pod by name and returns its port mappings.
type PodLocator struct {
	podName   string
	namespace string
	ports     []string
	client    kubernetes.Interface
}

// NewPodLocator creates a new pod locator for the specified pod name.
func NewPodLocator(podName string, namespace string, ports []string, client kubernetes.Interface) (*PodLocator, error) {
	return &PodLocator{
		podName:   podName,
		namespace: namespace,
		ports:     ports,
		client:    client,
	}, nil
}

// Locate finds the pod and verifies it's running, then returns its name and ports.
func (l *PodLocator) Locate(ctx context.Context) (string, []string, error) {
	pod, err := l.client.CoreV1().Pods(l.namespace).Get(ctx, l.podName, metav1.GetOptions{})
	if err != nil {
		// Classify API errors
		if apierrors.IsNotFound(err) {
			return "", []string{}, NewResourceNotFoundError("pod", l.podName, err)
		}
		if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
			return "", []string{}, NewAPITransientError(fmt.Sprintf("API timeout getting pod %s", l.podName), err)
		}
		if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
			return "", []string{}, NewPermissionDeniedError("get", fmt.Sprintf("pod %s", l.podName), err)
		}
		// Other API errors (network issues, etc.)
		return "", []string{}, NewAPITransientError(fmt.Sprintf("failed to get pod %s", l.podName), err)
	}

	// Check pod status
	if pod.Status.Phase == corev1.PodFailed {
		return "", []string{}, NewPodFailedError(l.podName, nil)
	}

	if pod.Status.Phase != corev1.PodRunning {
		return "", []string{}, NewPodNotRunningError(l.podName, string(pod.Status.Phase), nil)
	}

	return l.podName, l.ports, nil
}
