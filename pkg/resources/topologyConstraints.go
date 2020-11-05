package resources

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutateZoneTopologySpreadConstraints creates a PodTemplateMutation that
// sets the TopologySpreadConstraints for Multi AZ
func MutateZoneTopologySpreadConstraints(labelMatch string) PodTemplateMutation {
	return func(obj metav1.Object, podTemplate *corev1.PodTemplateSpec) error {
		labels := obj.GetLabels()
		if labels == nil {
			return errors.New("object has no labels")
		}

		labelValue, ok := labels[labelMatch]
		if !ok {
			return fmt.Errorf("label %s not found in object", labelMatch)
		}

		podTemplate.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           1,
				TopologyKey:       ZoneLabel,
				WhenUnsatisfiable: corev1.ScheduleAnyway,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						labelMatch: labelValue,
					},
				},
			},
		}

		return nil
	}
}
