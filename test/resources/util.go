package resources

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

func ScaleDown(inst *integreatlyv1alpha1.RHMI) bool {
	if v, ok := inst.Annotations["scale_down"]; !ok || v == "false" {
		return false
	}
	return true
}
