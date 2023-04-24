package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MutatePodPriority(priorityClassName string) PodTemplateMutation {
	return func(_ metav1.Object, podTemplate *corev1.PodTemplateSpec) error {
		podTemplate.Spec.PriorityClassName = priorityClassName
		return nil
	}
}
