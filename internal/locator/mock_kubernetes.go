package locator

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// MockKubernetesClient is a minimal mock for testing
type MockKubernetesClient struct {
	pods         map[string]*corev1.Pod
	services     map[string]*corev1.Service
	deployments  map[string]*appsv1.Deployment
	statefulsets map[string]*appsv1.StatefulSet
	daemonsets   map[string]*appsv1.DaemonSet
}

// NewMockKubernetesClient creates a new mock Kubernetes client
func NewMockKubernetesClient() *MockKubernetesClient {
	return &MockKubernetesClient{
		pods:         make(map[string]*corev1.Pod),
		services:     make(map[string]*corev1.Service),
		deployments:  make(map[string]*appsv1.Deployment),
		statefulsets: make(map[string]*appsv1.StatefulSet),
		daemonsets:   make(map[string]*appsv1.DaemonSet),
	}
}

// AddPod adds a pod to the mock
func (m *MockKubernetesClient) AddPod(namespace, name string, phase corev1.PodPhase) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
	m.pods[namespace+"/"+name] = pod
	return pod
}

// AddService adds a service to the mock with matching pods
func (m *MockKubernetesClient) AddService(namespace, name string, selector map[string]string, ports []corev1.ServicePort) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports:    ports,
		},
	}
	m.services[namespace+"/"+name] = svc
	return svc
}

// AddDeployment adds a deployment to the mock with matching pods
func (m *MockKubernetesClient) AddDeployment(namespace, name string, selector *metav1.LabelSelector) *appsv1.Deployment {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: selector,
		},
	}
	m.deployments[namespace+"/"+name] = deploy
	return deploy
}

// AddStatefulSet adds a statefulset to the mock
func (m *MockKubernetesClient) AddStatefulSet(namespace, name string, selector *metav1.LabelSelector) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: selector,
		},
	}
	m.statefulsets[namespace+"/"+name] = sts
	return sts
}

// AddDaemonSet adds a daemonset to the mock
func (m *MockKubernetesClient) AddDaemonSet(namespace, name string, selector *metav1.LabelSelector) *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
		},
	}
	m.daemonsets[namespace+"/"+name] = ds
	return ds
}

// CoreV1 returns a mock CoreV1Client
func (m *MockKubernetesClient) CoreV1() CoreV1Client {
	return &MockCoreV1{mock: m}
}

// AppsV1 returns a mock AppsV1Client
func (m *MockKubernetesClient) AppsV1() AppsV1Client {
	return &MockAppsV1{mock: m}
}

// MockCoreV1 implements CoreV1Client
type MockCoreV1 struct {
	mock *MockKubernetesClient
}

func (m *MockCoreV1) Pods(namespace string) PodClient {
	return &MockPodInterface{mock: m.mock, namespace: namespace}
}

func (m *MockCoreV1) Services(namespace string) ServiceClient {
	return &MockServiceInterface{mock: m.mock, namespace: namespace}
}

// MockPodInterface implements PodClient
type MockPodInterface struct {
	mock      *MockKubernetesClient
	namespace string
}

func (m *MockPodInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error) {
	pod, exists := m.mock.pods[m.namespace+"/"+name]
	if !exists {
		return nil, apierrors.NewNotFound(corev1.Resource("pods"), name)
	}
	return pod, nil
}

func (m *MockPodInterface) List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	result := &corev1.PodList{}
	var selector labels.Selector
	var err error

	if opts.LabelSelector != "" {
		selector, err = labels.Parse(opts.LabelSelector)
		if err != nil {
			return nil, err
		}
	}

	for _, pod := range m.mock.pods {
		if pod.Namespace == m.namespace {
			if selector != nil {
				if selector.Matches(labels.Set(pod.Labels)) {
					result.Items = append(result.Items, *pod)
				}
			} else {
				result.Items = append(result.Items, *pod)
			}
		}
	}
	return result, nil
}

// MockServiceInterface implements ServiceClient
type MockServiceInterface struct {
	mock      *MockKubernetesClient
	namespace string
}

func (m *MockServiceInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Service, error) {
	svc, exists := m.mock.services[m.namespace+"/"+name]
	if !exists {
		return nil, apierrors.NewNotFound(corev1.Resource("services"), name)
	}
	return svc, nil
}

// MockAppsV1 implements AppsV1Client
type MockAppsV1 struct {
	mock *MockKubernetesClient
}

func (m *MockAppsV1) Deployments(namespace string) ResourceClient {
	return &MockDeploymentInterface{mock: m.mock, namespace: namespace}
}

func (m *MockAppsV1) StatefulSets(namespace string) ResourceClient {
	return &MockStatefulSetInterface{mock: m.mock, namespace: namespace}
}

func (m *MockAppsV1) DaemonSets(namespace string) ResourceClient {
	return &MockDaemonSetInterface{mock: m.mock, namespace: namespace}
}

// MockDeploymentInterface implements ResourceClient for Deployments
type MockDeploymentInterface struct {
	mock      *MockKubernetesClient
	namespace string
}

func (m *MockDeploymentInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error) {
	deploy, exists := m.mock.deployments[m.namespace+"/"+name]
	if !exists {
		return nil, apierrors.NewNotFound(appsv1.Resource("deployments"), name)
	}
	return deploy, nil
}

// MockStatefulSetInterface implements ResourceClient for StatefulSets
type MockStatefulSetInterface struct {
	mock      *MockKubernetesClient
	namespace string
}

func (m *MockStatefulSetInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error) {
	sts, exists := m.mock.statefulsets[m.namespace+"/"+name]
	if !exists {
		return nil, apierrors.NewNotFound(appsv1.Resource("statefulsets"), name)
	}
	return sts, nil
}

// MockDaemonSetInterface implements ResourceClient for DaemonSets
type MockDaemonSetInterface struct {
	mock      *MockKubernetesClient
	namespace string
}

func (m *MockDaemonSetInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error) {
	ds, exists := m.mock.daemonsets[m.namespace+"/"+name]
	if !exists {
		return nil, apierrors.NewNotFound(appsv1.Resource("daemonsets"), name)
	}
	return ds, nil
}
