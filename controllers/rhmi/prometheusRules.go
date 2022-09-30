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
						"message": fmt.Sprintf("%s operator is taking more than 2.5 hours to go to a complete stage", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(fmt.Sprintf(`%s_version{to_version=~".+", version="" }`, installationName)),
					For:    "150m",
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
					Alert: fmt.Sprintf("%sUpgradeExpectedDuration10minExceeded", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlUpgradeExpectedDurationExceeded,
						"message": fmt.Sprintf("%s operator upgrade is taking more than 10 minutes", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(fmt.Sprintf(`%s_version{to_version=~".+",version=~".+"} and (absent((%s_version{job=~"%s.+"} * on(version) csv_succeeded{exported_namespace=~"%s"})) or %s_version)`, installationName, installationName, installationName, installation.Namespace, installationName)),
					For:    "10m",
					Labels: map[string]string{"severity": "warning", "product": installationName, "addon": getAddonName(installation), "namespace": "openshift-monitoring"},
				},
				{
					Alert: fmt.Sprintf("%sUpgradeExpectedDuration60minExceeded", strings.ToUpper(installationName)),
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

	return &resources.AlertReconcilerImpl{
		ProductName:  "installation",
		Installation: installation,
		Log:          log,
		Alerts:       alerts,
	}
}
