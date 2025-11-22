package locator

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPodLocatorFound tests that a running pod is found
func TestPodLocatorFound(t *testing.T) {
	mock := NewMockKubernetesClient()
	mock.AddPod("default", "api-server", corev1.PodRunning)

	locator, err := NewPodLocator("api-server", "default", []string{"8080"}, mock)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "api-server", podName)
	assert.Equal(t, []string{"8080"}, ports)
}

// TestPodLocatorNotFound tests error when pod doesn't exist
func TestPodLocatorNotFound(t *testing.T) {
	mock := NewMockKubernetesClient()

	locator, err := NewPodLocator("nonexistent", "default", []string{"8080"}, mock)
	require.NoError(t, err)

	_, _, err = locator.Locate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get pod")
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestPodLocatorNotRunning tests error when pod is not in running state
func TestPodLocatorNotRunning(t *testing.T) {
	testCases := []struct {
		name  string
		phase corev1.PodPhase
	}{
		{"pending", corev1.PodPending},
		{"failed", corev1.PodFailed},
		{"succeeded", corev1.PodSucceeded},
		{"unknown", corev1.PodUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockKubernetesClient()
			mock.AddPod("default", "api-server", tc.phase)

			locator, err := NewPodLocator("api-server", "default", []string{"8080"}, mock)
			require.NoError(t, err)

			_, _, err = locator.Locate(context.Background())

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not running")
			assert.Contains(t, err.Error(), string(tc.phase))
		})
	}
}

// TestServiceLocatorFound tests that a service with running pods is found
func TestServiceLocatorFound(t *testing.T) {
	mock := NewMockKubernetesClient()

	// Add service
	selector := map[string]string{"app": "api"}
	mock.AddService("default", "api-svc", selector, []corev1.ServicePort{
		{
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		},
	})

	// Add matching running pod
	pod := mock.AddPod("default", "api-server-1", corev1.PodRunning)
	pod.Labels = selector

	locator, err := NewServiceLocator("api-svc", "default", []string{"8080"}, mock)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "api-server-1", podName)
	assert.Equal(t, []string{"8080"}, ports)
}

// TestServiceLocatorNotFound tests error when service doesn't exist
func TestServiceLocatorNotFound(t *testing.T) {
	mock := NewMockKubernetesClient()

	locator, err := NewServiceLocator("nonexistent-svc", "default", []string{"8080"}, mock)
	require.NoError(t, err)

	_, _, err = locator.Locate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get service")
}

// TestServiceLocatorNoRunningPods tests error when service has no running pods
func TestServiceLocatorNoRunningPods(t *testing.T) {
	mock := NewMockKubernetesClient()

	// Add service
	selector := map[string]string{"app": "api"}
	mock.AddService("default", "api-svc", selector, []corev1.ServicePort{
		{
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		},
	})

	// Add pod but it's not running
	pod := mock.AddPod("default", "api-server-1", corev1.PodPending)
	pod.Labels = selector

	locator, err := NewServiceLocator("api-svc", "default", []string{"8080"}, mock)
	require.NoError(t, err)

	_, _, err = locator.Locate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running pod")
}

// TestDeploymentLocatorFound tests that a deployment with running pods is found
func TestDeploymentLocatorFound(t *testing.T) {
	mock := NewMockKubernetesClient()

	// Add deployment with selector
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "api"},
	}
	mock.AddDeployment("default", "api-deploy", selector)

	// Add matching running pod
	pod := mock.AddPod("default", "api-deploy-abc123", corev1.PodRunning)
	pod.Labels = selector.MatchLabels

	locator, err := NewSelectorBasedLocator("deployment", "api-deploy", "default", []string{"8080"}, mock)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "api-deploy-abc123", podName)
	assert.Equal(t, []string{"8080"}, ports)
}

// TestStatefulSetLocatorFound tests that a statefulset with running pods is found
func TestStatefulSetLocatorFound(t *testing.T) {
	mock := NewMockKubernetesClient()

	// Add statefulset
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "postgres"},
	}
	mock.AddStatefulSet("default", "postgres-sts", selector)

	// Add matching running pod
	pod := mock.AddPod("default", "postgres-sts-0", corev1.PodRunning)
	pod.Labels = selector.MatchLabels

	locator, err := NewSelectorBasedLocator("statefulset", "postgres-sts", "default", []string{"5432"}, mock)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "postgres-sts-0", podName)
	assert.Equal(t, []string{"5432"}, ports)
}

// TestDaemonSetLocatorFound tests that a daemonset with running pods is found
func TestDaemonSetLocatorFound(t *testing.T) {
	mock := NewMockKubernetesClient()

	// Add daemonset
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "monitoring"},
	}
	mock.AddDaemonSet("default", "prometheus-ds", selector)

	// Add matching running pod
	pod := mock.AddPod("default", "prometheus-ds-node1", corev1.PodRunning)
	pod.Labels = selector.MatchLabels

	locator, err := NewSelectorBasedLocator("daemonset", "prometheus-ds", "default", []string{"9090"}, mock)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "prometheus-ds-node1", podName)
	assert.Equal(t, []string{"9090"}, ports)
}

// TestBuildLocatorPodFormat tests BuildLocator with pod format
func TestBuildLocatorPodFormat(t *testing.T) {
	mock := NewMockKubernetesClient()
	mock.AddPod("default", "api-server", corev1.PodRunning)

	locator, err := BuildLocator("api-server", "default", []string{"8080"}, mock)

	require.NoError(t, err)
	assert.NotNil(t, locator)

	podName, _, err := locator.Locate(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "api-server", podName)
}

// TestBuildLocatorServiceFormats tests BuildLocator with various service formats
func TestBuildLocatorServiceFormats(t *testing.T) {
	testCases := []struct {
		name     string
		resource string
	}{
		{"short format", "svc/api-svc"},
		{"long format", "service/api-svc"},
		{"plural format", "services/api-svc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockKubernetesClient()
			selector := map[string]string{"app": "api"}
			mock.AddService("default", "api-svc", selector, []corev1.ServicePort{
				{Port: 8080, TargetPort: intstr.FromInt(8080)},
			})
			pod := mock.AddPod("default", "api-server-1", corev1.PodRunning)
			pod.Labels = selector

			locator, err := BuildLocator(tc.resource, "default", []string{"8080"}, mock)
			require.NoError(t, err)

			_, _, err = locator.Locate(context.Background())
			assert.NoError(t, err)
		})
	}
}

// TestBuildLocatorDeploymentFormats tests BuildLocator with deployment formats
func TestBuildLocatorDeploymentFormats(t *testing.T) {
	testCases := []struct {
		name     string
		resource string
	}{
		{"short format", "dep/api-deploy"},
		{"long format", "deployment/api-deploy"},
		{"plural format", "deployments/api-deploy"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockKubernetesClient()
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			}
			mock.AddDeployment("default", "api-deploy", selector)
			pod := mock.AddPod("default", "api-deploy-abc", corev1.PodRunning)
			pod.Labels = selector.MatchLabels

			locator, err := BuildLocator(tc.resource, "default", []string{"8080"}, mock)
			require.NoError(t, err)

			_, _, err = locator.Locate(context.Background())
			assert.NoError(t, err)
		})
	}
}

// TestBuildLocatorInvalidFormat tests BuildLocator with invalid format
func TestBuildLocatorInvalidFormat(t *testing.T) {
	mock := NewMockKubernetesClient()

	testCases := []struct {
		name     string
		resource string
	}{
		{"invalid type", "invalid/pod-name"},
		{"too many slashes", "dep/name/extra"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildLocator(tc.resource, "default", []string{"8080"}, mock)
			assert.Error(t, err)
		})
	}
}
