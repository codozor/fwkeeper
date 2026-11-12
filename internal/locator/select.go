package locator

import (
	corev1 "k8s.io/api/core/v1"
)

// selectPod picks the best pod to forward to among pods matching a selector:
//  1. Running, Ready and not terminating
//  2. Running and not terminating (e.g. readiness probe failing, still useful in dev)
//
// Terminating pods (DeletionTimestamp set) are never selected: during a rolling update
// they are still Running but about to disappear. Returns nil if no pod qualifies.
func selectPod(pods []corev1.Pod) *corev1.Pod {
	var fallback *corev1.Pod

	for i := range pods {
		p := &pods[i]
		if p.DeletionTimestamp != nil || p.Status.Phase != corev1.PodRunning {
			continue
		}
		if isPodReady(p) {
			return p
		}
		if fallback == nil {
			fallback = p
		}
	}

	return fallback
}

// isPodReady reports whether the pod has the Ready condition set to True.
func isPodReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
