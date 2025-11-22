package locator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodLocator locates a specific pod by name and returns its port mappings.
type PodLocator struct {
	podName   string
	namespace string
	ports     []string
	client    KubernetesClient
}

// NewPodLocator creates a new pod locator for the specified pod name.
func NewPodLocator(podName string, namespace string, ports []string, client KubernetesClient) (*PodLocator, error) {
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
		return "", []string{}, fmt.Errorf("failed to get pod %s: %w", l.podName, err)
	}

	if pod.Status.Phase != corev1.PodRunning {
		return "", []string{}, fmt.Errorf("pod %s is not running (phase: %s)", l.podName, pod.Status.Phase)
	}

	return l.podName, l.ports, nil
}
