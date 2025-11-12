package service

type Locator interface {
	locate() (string, error)
}


type podLocator struct {
	podName string
	namespace string
}

func NewPodLocator(podName string, namespace string) (*podLocator, error) {
	return &podLocator{ podName: podName, namespace: namespace }, nil
}

func (l *podLocator) locate() (string, error) {
	return l.podName, nil
}
