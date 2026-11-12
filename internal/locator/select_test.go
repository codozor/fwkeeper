package locator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type podOpt func(*corev1.Pod)

func ready(p *corev1.Pod) {
	p.Status.Conditions = append(p.Status.Conditions, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue})
}

func notReady(p *corev1.Pod) {
	p.Status.Conditions = append(p.Status.Conditions, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse})
}

func terminating(p *corev1.Pod) {
	now := metav1.Now()
	p.DeletionTimestamp = &now
	p.Finalizers = []string{"test"} // the fake client requires finalizers on objects being deleted
}

func phase(ph corev1.PodPhase) podOpt {
	return func(p *corev1.Pod) { p.Status.Phase = ph }
}

func testPod(name string, lbls map[string]string, opts ...podOpt) corev1.Pod {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: lbls},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	for _, o := range opts {
		o(&p)
	}
	return p
}

func TestSelectPod(t *testing.T) {
	testCases := []struct {
		name     string
		pods     []corev1.Pod
		expected string // empty: no pod selected
	}{
		{"empty", nil, ""},
		{"ready preferred over running", []corev1.Pod{
			testPod("running", nil, notReady),
			testPod("ready", nil, ready),
		}, "ready"},
		{"terminating ready pod skipped", []corev1.Pod{
			testPod("old", nil, ready, terminating),
			testPod("new", nil, ready),
		}, "new"},
		{"running not ready as fallback", []corev1.Pod{
			testPod("pending", nil, phase(corev1.PodPending)),
			testPod("starting", nil, notReady),
		}, "starting"},
		{"only terminating pods", []corev1.Pod{
			testPod("old", nil, ready, terminating),
		}, ""},
		{"no running pod", []corev1.Pod{
			testPod("pending", nil, phase(corev1.PodPending)),
			testPod("failed", nil, phase(corev1.PodFailed)),
		}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := selectPod(tc.pods)
			if tc.expected == "" {
				assert.Nil(t, p)
				return
			}
			require.NotNil(t, p)
			assert.Equal(t, tc.expected, p.Name)
		})
	}
}

// Rolling update: the service locator must skip the terminating pod.
func TestServiceLocatorSkipsTerminatingPod(t *testing.T) {
	lbls := map[string]string{"app": "api"}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			Selector: lbls,
			Ports:    []corev1.ServicePort{{Port: 8080}},
		},
	}
	old := testPod("api-old", lbls, ready, terminating)
	newPod := testPod("api-new", lbls, ready)

	l, err := NewServiceLocator("api", "default", []string{"8080"}, newTestMockClient(svc, &old, &newPod))
	require.NoError(t, err)

	podName, _, err := l.Locate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "api-new", podName)
}

// A service without selector must not match every pod of the namespace.
func TestServiceLocatorWithoutSelector(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "external", Namespace: "default"},
		Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 8080}}},
	}
	unrelated := testPod("unrelated", map[string]string{"app": "other"}, ready)

	l, err := NewServiceLocator("external", "default", []string{"8080"}, newTestMockClient(svc, &unrelated))
	require.NoError(t, err)

	_, _, err = l.Locate(context.Background())
	require.Error(t, err)
	assert.Equal(t, ErrorTypeConfigInvalid, GetErrorType(err))
}

// matchExpressions must be honoured: previously only matchLabels was used,
// so a selector made only of expressions matched every pod of the namespace.
func TestSelectorBasedLocatorMatchExpressions(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"api"}},
			{Key: "canary", Operator: metav1.LabelSelectorOpDoesNotExist},
		},
	}

	other := testPod("a-other", map[string]string{"app": "db"}, ready)
	canary := testPod("b-canary", map[string]string{"app": "api", "canary": "true"}, ready)
	api := testPod("c-api", map[string]string{"app": "api"}, ready)

	objects := func(workload runtime.Object) []runtime.Object {
		return []runtime.Object{workload, &other, &canary, &api}
	}
	meta := metav1.ObjectMeta{Name: "api", Namespace: "default"}

	testCases := []struct {
		kind     string
		workload runtime.Object
	}{
		{"deployment", &appsv1.Deployment{ObjectMeta: meta, Spec: appsv1.DeploymentSpec{Selector: selector}}},
		{"statefulset", &appsv1.StatefulSet{ObjectMeta: meta, Spec: appsv1.StatefulSetSpec{Selector: selector}}},
		{"daemonset", &appsv1.DaemonSet{ObjectMeta: meta, Spec: appsv1.DaemonSetSpec{Selector: selector}}},
	}

	for _, tc := range testCases {
		t.Run(tc.kind, func(t *testing.T) {
			l, err := NewSelectorBasedLocator(tc.kind, "api", "default", []string{"8080"}, newTestMockClient(objects(tc.workload)...))
			require.NoError(t, err)

			podName, _, err := l.Locate(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "c-api", podName)
		})
	}
}

// An empty selector must be rejected instead of matching every pod.
func TestSelectorBasedLocatorEmptySelector(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{}},
	}
	pod := testPod("unrelated", map[string]string{"app": "other"}, ready)

	l, err := NewSelectorBasedLocator("deployment", "api", "default", []string{"8080"}, newTestMockClient(dep, &pod))
	require.NoError(t, err)

	_, _, err = l.Locate(context.Background())
	require.Error(t, err)
	assert.Equal(t, ErrorTypeConfigInvalid, GetErrorType(err))
}
