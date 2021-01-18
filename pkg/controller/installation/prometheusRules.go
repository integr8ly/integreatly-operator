package installation

import (
	"fmt"
	"strings"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

var (
	installationNames = map[string]string{
		string(integreatlyv1alpha1.InstallationTypeManaged):    "rhmi",
		string(integreatlyv1alpha1.InstallationTypeManagedApi): "rhoam",
	}
)

func (r *ReconcileInstallation) newAlertsReconciler(installation *integreatlyv1alpha1.RHMI) resources.AlertReconciler {
	installationName := installationNames[installation.Spec.Type]

	return &resources.AlertReconcilerImpl{
		ProductName:  "installation",
		Installation: installation,
		Log:          log,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: fmt.Sprintf("%s-installation-controller-alerts", installationName),
				Namespace: installation.Namespace,
				GroupName: fmt.Sprintf("%s-installation.rules", installationName),
				Rules: []monitoringv1.Rule{
					{
						Alert: fmt.Sprintf("%sInstallationControllerIsNotReconciling", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": fmt.Sprintf("%s operator has not reconciled successfully in the interval of 15m over the past 1 hour", strings.ToUpper(installationName)),
						},
						Expr:   intstr.FromString(fmt.Sprintf("%s_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='success'}[15m]) == 0", installationName)),
						For:    "1h",
						Labels: map[string]string{"severity": "warning"},
					},
					{
						Alert: fmt.Sprintf("%sInstallationControllerStoppedReconciling", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": fmt.Sprintf("%s operator has not reconciled successfully in the interval of 30m over the past 2 hours", strings.ToUpper(installationName)),
						},
						Expr:   intstr.FromString(fmt.Sprintf("%s_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='success'}[30m]) == 0", installationName)),
						For:    "2h",
						Labels: map[string]string{"severity": "warning"},
					},
				},
			},
			{
				AlertName: fmt.Sprintf("%s-installation-alerts", installationName),
				Namespace: monitoring.OpenshiftMonitoringNamespace,
				GroupName: fmt.Sprintf("%s-installation.rules", installationName),
				Rules: []monitoringv1.Rule{
					{
						Alert: fmt.Sprintf("%sOperatorInstallDelayed", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlOperatorInstallDelayed,
							"message": fmt.Sprintf("%s operator is taking more than 2 hours to go to a complete stage", strings.ToUpper(installationName)),
						},
						Expr:   intstr.FromString(fmt.Sprintf(`absent(%s_status{stage='complete'} == 1) and absent(%s_version{version=~".+"})`, installationName, installationName)),
						For:    "120m",
						Labels: map[string]string{"severity": "critical", "addon": "managed-api-service"},
					},
				},
			},
			{
				AlertName: fmt.Sprintf("%s-upgrade-alerts", installationName),
				Namespace: monitoring.OpenshiftMonitoringNamespace,
				GroupName: fmt.Sprintf("%s-upgrade.rules", installationName),
				Rules: []monitoringv1.Rule{
					{
						Alert: fmt.Sprintf("%sUpgradeExpectedDurationExceeded", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlUpgradeExpectedDurationExceeded,
							"message": fmt.Sprintf("%s operator upgrade is taking more than 10 minutes", strings.ToUpper(installationName)),
						},
						Expr:   intstr.FromString(fmt.Sprintf(`%s_version{to_version=~".+",version=~".+"} and (absent((%s_version * on(version) csv_succeeded{exported_namespace=~"%s"})) or %s_version)`, installationName, installationName, installation.Namespace, installationName)),
						For:    "10m",
						Labels: map[string]string{"severity": "critical", "addon": "managed-api-service"},
					},
				},
			},
		},
	}
}
