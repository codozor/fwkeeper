package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/client-go/kubernetes"
)

type Locator interface {
	locate(ctx context.Context) (string, []string, error)
}


type podLocator struct {
	podName string
	namespace string

	ports []string

	client kubernetes.Interface
}

func NewPodLocator(podName string, namespace string, ports []string, client kubernetes.Interface) (*podLocator, error) {
	return &podLocator{ podName: podName, namespace: namespace, ports: ports, client: client }, nil
}

func (l *podLocator) locate(ctx context.Context) (string, []string, error) {

	pod, err := l.client.CoreV1().Pods(l.namespace).Get(ctx, l.podName, metav1.GetOptions{})
	if err != nil {
		return "", []string{}, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return "", []string{}, fmt.Errorf("pod is not in running state")
	}

	return l.podName, l.ports, nil
}

type svcLocator struct {
	svcName string
	namespace string

	ports []string

	client kubernetes.Interface
}

func NewServiceLocator(svcName string, namespace string, ports []string, client kubernetes.Interface) (*svcLocator, error)  {
	return &svcLocator{ svcName: svcName, namespace: namespace, ports: ports, client: client }, nil
}

func (l *svcLocator) locate(ctx context.Context) (string, []string, error) {
	svc, err := l.client.CoreV1().Services(l.namespace).Get(ctx, l.svcName, metav1.GetOptions{})
	if err != nil {
		return "", []string{}, err
	}

	labelSelector := labels.Set(svc.Spec.Selector).AsSelector()

	pods , err := l.client.CoreV1().Pods(l.namespace).List(ctx, metav1.ListOptions{
		LabelSelector:  labelSelector.String(),
	})
	if err != nil {
		return "", []string{}, err
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
	
	return "", []string{}, fmt.Errorf("no running pod found")
}

func (l *svcLocator) mapPorts(svc *corev1.Service, pod *corev1.Pod) ([]string, error) {
	result := []string{}

	for i := range l.ports {
		parts := strings.Split(l.ports[i], ":")

		srcPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return []string{}, err
		}

		dstPort := srcPort
		if len(parts) > 1 {
			dstPort, err = strconv.Atoi(parts[1])
			if err != nil {
				return []string{}, err
			}
		}

		sp, ok := lo.Find(svc.Spec.Ports, func (p corev1.ServicePort) bool {
			return p.Port == int32(dstPort)
		})
		if ! ok {
			return []string{}, fmt.Errorf("port %d does not exists in service %s", dstPort, svc.Name)
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
				return []string{}, fmt.Errorf("port %d does not exists in service %s", dstPort, svc.Name)
			}

			dstPort = int(pp.HostPort)
		}

		if dstPort == srcPort {
			result = append(result, fmt.Sprintf("%d", srcPort))
		} else {
			result = append(result, fmt.Sprintf("%d:%d", srcPort, dstPort))
		}
	}

	return result, nil
}

func buildLocator(resource string, namespace string, ports []string, client kubernetes.Interface) (Locator, error) {
	parts := strings.Split(resource, "/")

	if len(parts) == 1 {
		return NewPodLocator(resource, namespace, ports, client)
	} else if len(parts) == 2 {
		if parts[0] == "svc" || parts[0] == "service" || parts[0] == "services" {
			return NewServiceLocator(parts[1], namespace, ports, client)
		} else {
			return nil, fmt.Errorf("unhandled resource")	
		}
	} else {
		return nil, fmt.Errorf("invalid resource name")
	}
}