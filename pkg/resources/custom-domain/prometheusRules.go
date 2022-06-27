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
	lastError := installation.Status.CustomDomain.Error

	alerts := []resources.AlertConfiguration{
		customDomainCRErrorState(installationName, namespace, lastError),
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "installation",
		Installation: installation,
		Log:          log,
		Alerts:       alerts,
	}
}

func customDomainCRErrorState(installationName string, namespace string, err string) resources.AlertConfiguration {
	rule := resources.AlertConfiguration{
		AlertName: fmt.Sprintf("%s-custom-domain-alert", installationName),
		Namespace: namespace,
		GroupName: fmt.Sprintf("%s-custom-domaim.rules", installationName),
		Rules: []monitoringv1.Rule{
			{
				Alert: "CustomDomainCRErrorState",
				Annotations: map[string]string{
					"sop_url": resources.SopUrlRHOAMServiceDefinition,
					"message": fmt.Sprintf(fmt.Sprintf("Error configuring custom domain, please refer to the documetaion to resolve the error. Found error: %s", err)),
				},
				Expr:   intstr.FromString(fmt.Sprintf("%s_custom_domain{customDomain='active'} > 0", installationName)),
				For:    "5m",
				Labels: map[string]string{"severity": "warning", "product": string(v1alpha1.InstallationTypeManagedApi)},
			},
		},
	}
	return rule
}
