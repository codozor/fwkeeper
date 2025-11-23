package locator

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestMockClient creates a fake Kubernetes client for testing
func newTestMockClient(objects ...runtime.Object) *fake.Clientset {
	return fake.NewClientset(objects...)
}

// TestPodLocatorFound tests that a running pod is found
func TestPodLocatorFound(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	client := newTestMockClient(pod)
	locator, err := NewPodLocator("api-server", "default", []string{"8080"}, client)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "api-server", podName)
	assert.Equal(t, []string{"8080"}, ports)
}

// TestPodLocatorNotFound tests error when pod doesn't exist
func TestPodLocatorNotFound(t *testing.T) {
	client := newTestMockClient()
	locator, err := NewPodLocator("nonexistent", "default", []string{"8080"}, client)
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
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-server",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: tc.phase,
				},
			}

			client := newTestMockClient(pod)
			locator, err := NewPodLocator("api-server", "default", []string{"8080"}, client)
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
	selector := map[string]string{"app": "api"}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server-1",
			Namespace: "default",
			Labels:    selector,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	client := newTestMockClient(svc, pod)
	locator, err := NewServiceLocator("api-svc", "default", []string{"8080"}, client)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "api-server-1", podName)
	assert.Equal(t, []string{"8080"}, ports)
}

// TestServiceLocatorNotFound tests error when service doesn't exist
func TestServiceLocatorNotFound(t *testing.T) {
	client := newTestMockClient()
	locator, err := NewServiceLocator("nonexistent-svc", "default", []string{"8080"}, client)
	require.NoError(t, err)

	_, _, err = locator.Locate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get service")
}

// TestServiceLocatorNoRunningPods tests error when service has no running pods
func TestServiceLocatorNoRunningPods(t *testing.T) {
	selector := map[string]string{"app": "api"}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server-1",
			Namespace: "default",
			Labels:    selector,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	client := newTestMockClient(svc, pod)
	locator, err := NewServiceLocator("api-svc", "default", []string{"8080"}, client)
	require.NoError(t, err)

	_, _, err = locator.Locate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running pod")
}

// TestDeploymentLocatorFound tests that a deployment with running pods is found
func TestDeploymentLocatorFound(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "api"},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: selector,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-deploy-abc123",
			Namespace: "default",
			Labels:    selector.MatchLabels,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	client := newTestMockClient(deploy, pod)
	locator, err := NewSelectorBasedLocator("deployment", "api-deploy", "default", []string{"8080"}, client)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "api-deploy-abc123", podName)
	assert.Equal(t, []string{"8080"}, ports)
}

// TestStatefulSetLocatorFound tests that a statefulset with running pods is found
func TestStatefulSetLocatorFound(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "postgres"},
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-sts",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: selector,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-sts-0",
			Namespace: "default",
			Labels:    selector.MatchLabels,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	client := newTestMockClient(sts, pod)
	locator, err := NewSelectorBasedLocator("statefulset", "postgres-sts", "default", []string{"5432"}, client)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "postgres-sts-0", podName)
	assert.Equal(t, []string{"5432"}, ports)
}

// TestDaemonSetLocatorFound tests that a daemonset with running pods is found
func TestDaemonSetLocatorFound(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "monitoring"},
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-ds",
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-ds-node1",
			Namespace: "default",
			Labels:    selector.MatchLabels,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	client := newTestMockClient(ds, pod)
	locator, err := NewSelectorBasedLocator("daemonset", "prometheus-ds", "default", []string{"9090"}, client)
	require.NoError(t, err)

	podName, ports, err := locator.Locate(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "prometheus-ds-node1", podName)
	assert.Equal(t, []string{"9090"}, ports)
}

// TestBuildLocatorPodFormat tests BuildLocator with pod format
func TestBuildLocatorPodFormat(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	client := newTestMockClient(pod)
	locator, err := BuildLocator("api-server", "default", []string{"8080"}, client)

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
			selector := map[string]string{"app": "api"}
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-svc",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Selector: selector,
					Ports: []corev1.ServicePort{
						{Port: 8080, TargetPort: intstr.FromInt(8080)},
					},
				},
			}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-server-1",
					Namespace: "default",
					Labels:    selector,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}

			client := newTestMockClient(svc, pod)
			locator, err := BuildLocator(tc.resource, "default", []string{"8080"}, client)
			require.NoError(t, err)

			_, _, err = locator.Locate(context.Background())
			assert.NoError(t, err)
		})
	}
}

// TestBuildLocatorDeploymentFormats tests BuildLocator with various deployment formats
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
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			}
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: selector,
				},
			}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-deploy-abc",
					Namespace: "default",
					Labels:    selector.MatchLabels,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}

			client := newTestMockClient(deploy, pod)
			locator, err := BuildLocator(tc.resource, "default", []string{"8080"}, client)
			require.NoError(t, err)

			_, _, err = locator.Locate(context.Background())
			assert.NoError(t, err)
		})
	}
}

// TestBuildLocatorInvalidFormat tests BuildLocator with invalid format
func TestBuildLocatorInvalidFormat(t *testing.T) {
	client := newTestMockClient()

	testCases := []struct {
		name     string
		resource string
	}{
		{"invalid type", "invalid/pod-name"},
		{"too many slashes", "dep/name/extra"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildLocator(tc.resource, "default", []string{"8080"}, client)
			assert.Error(t, err)
		})
	}
}
