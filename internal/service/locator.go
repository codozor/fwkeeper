package service

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/client-go/kubernetes"
)

type Locator interface {
	locate(ctx context.Context) (string, error)
}


type podLocator struct {
	podName string
	namespace string
}

func NewPodLocator(podName string, namespace string) (*podLocator, error) {
	return &podLocator{ podName: podName, namespace: namespace }, nil
}

func (l *podLocator) locate(ctx context.Context) (string, error) {
	return l.podName, nil
}

type svcLocator struct {
	svcName string
	namespace string

	client kubernetes.Interface
}

func NewServiceLocator(svcName string, namespace string, client kubernetes.Interface) (*svcLocator, error)  {
	return &svcLocator{ svcName: svcName, namespace: namespace, client: client}, nil
}

func (l *svcLocator) locate(ctx context.Context) (string, error) {
	svc, err := l.client.CoreV1().Services(l.namespace).Get(ctx, l.svcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	labelSelector := labels.Set(svc.Spec.Selector).AsSelector()

	pods , err := l.client.CoreV1().Pods(l.namespace).List(ctx, metav1.ListOptions{
		LabelSelector:  labelSelector.String(),
	})
	if err != nil {
		return "", err
	}

	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning {
			return p.Name, nil
		}
	}
	
	return "", fmt.Errorf("no running pod found")
}

func buildLocator(resource string, namespace string, client kubernetes.Interface) (Locator, error) {
	parts := strings.Split(resource, "/")

	if len(parts) == 1 {
		return NewPodLocator(resource, namespace)
	} else if len(parts) == 2 {
		if parts[0] == "svc" || parts[0] == "service" || parts[0] == "services" {
			return NewServiceLocator(parts[1], namespace, client)
		} else {
			return nil, fmt.Errorf("unhandled resource")	
		}
	} else {
		return nil, fmt.Errorf("invalid resource name")
	}
}