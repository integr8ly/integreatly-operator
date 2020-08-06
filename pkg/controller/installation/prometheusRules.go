package installation

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/intstr"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

func (r *ReconcileInstallation) newAlertsReconciler(logger *logrus.Entry, installation *integreatlyv1alpha1.RHMI) resources.AlertReconciler {
	return &resources.AlertReconcilerImpl{
		ProductName:  "installation",
		Installation: installation,
		Logger:       logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "rhmi-installation-controller-alerts",
				Namespace: installation.Namespace,
				GroupName: "rhmi-installation.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIInstallationControllerIsNotReconciling",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "RHMI operator has not reconciled successfully in the interval of 15m over the past 1 hour",
						},
						Expr:   intstr.FromString(fmt.Sprint("rhmi_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='success'}[15m]) == 0")),
						For:    "1h",
						Labels: map[string]string{"severity": "warning"},
					},
					{
						Alert: "RHMIInstallationControllerStoppedReconciling",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "RHMI operator has not reconciled successfully in the interval of 30m over the past 2 hours",
						},
						Expr:   intstr.FromString(fmt.Sprint("rhmi_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='success'}[30m]) == 0")),
						For:    "2h",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
			{
				AlertName: "rhmi-installation-alerts",
				Namespace: "openshift-monitoring",
				GroupName: "rhmi-installation.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIOperatorInstallDelayed",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "RHMI operator is taking more than 2 hours to go to a complete stage",
						},
						Expr:   intstr.FromString(fmt.Sprint("absent(rhmi_status{stage='complete'} == 1)")),
						For:    "120m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
			{
				AlertName: "rhmi-upgrade-alerts",
				Namespace: "openshift-monitoring",
				GroupName: "rhmi-upgrade.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIUpgradeExpectedDurationExceeded",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "RHMI operator upgrade is taking more than 10 minutes",
						},
						Expr:   intstr.FromString(fmt.Sprintf(`absent((rhmi_version * on(version) csv_succeeded{exported_namespace=~"%s"}) or absent(rhmi_version))`, installation.Namespace)),
						For:    "10m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
		},
	}
}
