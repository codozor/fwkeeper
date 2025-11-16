package locator

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// ServiceLocator locates a pod backing a service and maps service ports to pod ports.
type ServiceLocator struct {
	svcName   string
	namespace string
	ports     []string
	client    kubernetes.Interface
}

// NewServiceLocator creates a new service locator for the specified service name.
func NewServiceLocator(svcName string, namespace string, ports []string, client kubernetes.Interface) (*ServiceLocator, error) {
	return &ServiceLocator{
		svcName:   svcName,
		namespace: namespace,
		ports:     ports,
		client:    client,
	}, nil
}

// Locate finds a running pod backing the service and returns its name and mapped ports.
func (l *ServiceLocator) Locate(ctx context.Context) (string, []string, error) {
	svc, err := l.client.CoreV1().Services(l.namespace).Get(ctx, l.svcName, metav1.GetOptions{})
	if err != nil {
		return "", []string{}, fmt.Errorf("failed to get service %s: %w", l.svcName, err)
	}

	labelSelector := labels.Set(svc.Spec.Selector).AsSelector()

	pods, err := l.client.CoreV1().Pods(l.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("failed to list pods for service %s: %w", l.svcName, err)
	}

	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning {
			ports, err := l.mapPorts(svc, &p)
			if err != nil {
				return "", []string{}, err
			}

			return p.Name, ports, nil
		}
	}

	return "", []string{}, fmt.Errorf("no running pod found for service %s", l.svcName)
}

// mapPorts translates service ports to pod container ports.
// It handles both numeric port numbers and named ports.
func (l *ServiceLocator) mapPorts(svc *corev1.Service, pod *corev1.Pod) ([]string, error) {
	result := []string{}

	for i := range l.ports {
		parts := strings.Split(l.ports[i], ":")

		srcPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return []string{}, fmt.Errorf("invalid local port %s: %w", parts[0], err)
		}

		dstPort := srcPort
		if len(parts) > 1 {
			dstPort, err = strconv.Atoi(parts[1])
			if err != nil {
				return []string{}, fmt.Errorf("invalid remote port %s: %w", parts[1], err)
			}
		}

		sp, ok := lo.Find(svc.Spec.Ports, func(p corev1.ServicePort) bool {
			return p.Port == int32(dstPort)
		})
		if !ok {
			return []string{}, fmt.Errorf("service %s does not expose port %d", svc.Name, dstPort)
		}

		if sp.TargetPort.Type == intstr.Int {
			dstPort = int(sp.TargetPort.IntVal)
		} else {
			pp, ok := lo.Find(lo.FlatMap(pod.Spec.Containers, func(c corev1.Container, _ int) []corev1.ContainerPort {
				return c.Ports
			}), func(p corev1.ContainerPort) bool {
				return sp.TargetPort.StrVal == p.Name
			})
			if !ok {
				return []string{}, fmt.Errorf("pod %s does not have named port %s", pod.Name, sp.TargetPort.StrVal)
			}

			dstPort = int(pp.ContainerPort)
		}

		if dstPort == srcPort {
			result = append(result, fmt.Sprintf("%d", srcPort))
		} else {
			result = append(result, fmt.Sprintf("%d:%d", srcPort, dstPort))
		}
	}

	return result, nil
}
