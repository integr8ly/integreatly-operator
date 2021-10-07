package controllers

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/products/observability"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

var (
	installationNames = map[string]string{
		string(integreatlyv1alpha1.InstallationTypeManaged):               "rhmi",
		string(integreatlyv1alpha1.InstallationTypeManagedApi):            "rhoam",
		string(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi): "rhoam",
	}
)

func getAddonName(installation *integreatlyv1alpha1.RHMI) string {
	switch installation.Spec.Type {
	case "managed-api":
		return "managed-api-service"
	case "managed":
		return "integreatly-operator"
	default:
		return ""
	}
}

func (r *RHMIReconciler) newAlertsReconciler(installation *integreatlyv1alpha1.RHMI) resources.AlertReconciler {
	installationName := installationNames[installation.Spec.Type]

	alerts := []resources.AlertConfiguration{
		{
			AlertName: fmt.Sprintf("%s-installation-alerts", installationName),
			Namespace: observability.OpenshiftMonitoringNamespace,
			GroupName: fmt.Sprintf("%s-installation.rules", installationName),
			Rules: []monitoringv1.Rule{
				{
					Alert: fmt.Sprintf("%sOperatorInstallDelayed", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlOperatorInstallDelayed,
						"message": fmt.Sprintf("%s operator is taking more than 2 hours to go to a complete stage", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(fmt.Sprintf(`%s_version{to_version=~".+", version="" }`, installationName)),
					For:    "120m",
					Labels: map[string]string{"severity": "critical", "product": installationName, "addon": getAddonName(installation), "namespace": "openshift-monitoring"},
				},
			},
		},
		{
			AlertName: fmt.Sprintf("%s-upgrade-alerts", installationName),
			Namespace: observability.OpenshiftMonitoringNamespace,
			GroupName: fmt.Sprintf("%s-upgrade.rules", installationName),
			Rules: []monitoringv1.Rule{
				{
					Alert: fmt.Sprintf("%sUpgradeExpectedDurationExceeded", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlUpgradeExpectedDurationExceeded,
						"message": fmt.Sprintf("%s operator upgrade is taking more than 60 minutes", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(fmt.Sprintf(`%s_version{to_version=~".+",version=~".+"} and (absent((%s_version{job=~"%s.+"} * on(version) csv_succeeded{exported_namespace=~"%s"})) or %s_version)`, installationName, installationName, installationName, installation.Namespace, installationName)),
					For:    "60m",
					Labels: map[string]string{"severity": "critical", "product": installationName, "addon": getAddonName(installation), "namespace": "openshift-monitoring"},
				},
			},
		},
	}

	if !integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		installationAlert := resources.AlertConfiguration{
			AlertName: fmt.Sprintf("%s-installation-controller-alerts", installationName),
			Namespace: installation.Namespace,
			GroupName: fmt.Sprintf("%s-installation.rules", installationName),
			Rules: []monitoringv1.Rule{
				{
					Alert: fmt.Sprintf("%sInstallationControllerIsInReconcilingErrorState", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": fmt.Sprintf("%s operator has finished installing, but has been in a error state while reconciling for 5 of the last 10 minutes", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(fmt.Sprintf("%s_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='error'}[5m]) > 0", installationName)),
					For:    "10m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
			},
		}

		alerts = append(alerts, installationAlert)
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "installation",
		Installation: installation,
		Log:          log,
		Alerts:       alerts,
	}
}
