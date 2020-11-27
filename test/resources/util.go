package resources

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"
)

func RunningInProw(inst *integreatlyv1alpha1.RHMI) bool {
	if v, ok := inst.Annotations["in_prow"]; !ok || v == "false" {
		logrus.Info("detected that this operator is not running in prow")
		return false
	}
	logrus.Info("detected that this operator is running in prow")
	return true
}
