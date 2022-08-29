package custom_domain

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	installationNames = map[string]string{
		string(v1alpha1.InstallationTypeManagedApi):            "rhoam",
		string(v1alpha1.InstallationTypeMultitenantManagedApi): "rhoam",
	}
)

func Alerts(installation *v1alpha1.RHMI, log logger.Logger, namespace string) resources.AlertReconciler {
	installationName := installationNames[installation.Spec.Type]
	alerts := []resources.AlertConfiguration{
		customDomainCRErrorState(installationName, namespace),
	}
	return &resources.AlertReconcilerImpl{
		ProductName:  "installation",
		Installation: installation,
		Log:          log,
		Alerts:       alerts,
	}
}

func customDomainCRErrorState(installationName string, namespace string) resources.AlertConfiguration {
	rule := resources.AlertConfiguration{
		AlertName: fmt.Sprintf("%s-custom-domain-alert", installationName),
		Namespace: namespace,
		GroupName: fmt.Sprintf("%s-custom-domaim.rules", installationName),
		Rules: []monitoringv1.Rule{
			{
				Alert: "CustomDomainCRErrorState",
				Annotations: map[string]string{
					"sop_url": resources.SopUrlRHOAMServiceDefinition,
					"message": "Error configuring custom domain, please refer to the documentation to resolve the error.",
				},
				Expr:   intstr.FromString(fmt.Sprintf("%s_custom_domain{active='true'} > 0", installationName)),
				For:    "5m",
				Labels: map[string]string{"severity": "warning", "product": installationName},
			},
			{
				Alert: "DnsBypassThreeScaleAdminUI",
				Annotations: map[string]string{
					"sop_url": resources.SopUrlDnsBypassThreeScaleAdminUI,
					"message": "3Scale Admin UI, bypassing DNS: If this console is unavailable, the client is unable to configure or administer their API setup.",
				},
				Expr:   intstr.FromString("threescale_portals{system_master='false'} > 0"),
				For:    "15m",
				Labels: map[string]string{"severity": "critical", "product": installationName},
			},
			{
				Alert: "DnsBypassThreeScaleDeveloperUI",
				Annotations: map[string]string{
					"sop_url": resources.SopUrlDnsBypassThreeScaleDeveloperUI,
					"message": "3Scale Developer UI, bypassing DNS: If this console is unavailable, the client developers are unable signup or perform API management.",
				},
				Expr:   intstr.FromString("threescale_portals{system_developer='false'} > 0"),
				For:    "15m",
				Labels: map[string]string{"severity": "critical", "product": installationName},
			},
			{
				Alert: "DnsBypassThreeScaleSystemAdminUI",
				Annotations: map[string]string{
					"sop_url": resources.SopUrlDnsBypassThreeScaleSystemAdminUI,
					"message": "3Scale System Admin UI, bypassing DNS: If this console is unavailable, the client is unable to perform Account Management, Analytics or Billing.",
				},
				Expr:   intstr.FromString("threescale_portals{system_provider='false'} > 0"),
				For:    "15m",
				Labels: map[string]string{"severity": "critical", "product": installationName},
			},
		},
	}
	return rule
}
