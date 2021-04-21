package rhssouser

import (
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertsReconciler(logger l.Logger, installType string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]
	return &resources.AlertReconcilerImpl{
		ProductName:  "RHSSO User",
		Installation: r.Installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-endpoint-alerts",
				GroupName: "user-rhsso-endpoint.rules",
				Namespace: r.Config.GetNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIUserRhssoKeycloakServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='keycloak', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIUserRhssoKeycloakDiscoveryServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='keycloak-discovery', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},

			{
				AlertName: "ksm-endpoint-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "user-rhsso-operator-endpoint.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIUserRhssoOperatorRhmiRegistryCsMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIUserRhssoKeycloakOperatorMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='keycloak-operator-metrics', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},

			//SLO-availability-alerts:
			//https://sre.google/workbook/alerting-on-slos/
			//https://promtools.dev/alerts/errors
			{
				AlertName: "rhoam-user-sso-slo-availability-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "rhoam-user-sso-slo-availability.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHOAMUserSsoAvailability5mto1hErrorBudgetBurn",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMUserSsoAvailabilityAlert,
							"message": "High 5m and 1h error budget burn for RHOAM SSO User",
						},
						Expr: intstr.FromString(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[5m]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[5m])) > (14.40 * (1-0.99000)))
							and
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[1h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[1h])) > (14.40 * (1-0.99000)))`),
						For:    "2m",
						Labels: map[string]string{"severity": "info", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
					{
						Alert: "RHOAMUserSsoAvailability30mto6hErrorBudgetBurn",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMUserSsoAvailabilityAlert,
							"message": "High 30m and 6h error budget burn for RHOAM SSO User",
						},
						Expr: intstr.FromString(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[30m]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[30m])) > (6.00 * (1-0.99000)))
							and 
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[6h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[6h])) > (6.00 * (1-0.99000)))`),
						For:    "15m",
						Labels: map[string]string{"severity": "info", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
					{
						Alert: "RHOAMUserSsoAvailability2hto1dErrorBudgetBurn",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMUserSsoAvailabilityAlert,
							"message": "High 2h and 1d error budget burn for RHOAM SSO User",
						},
						Expr: intstr.FromString(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[2h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[2h])) > (3.00 * (1-0.99000)))
							and
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[1d]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[1d])) > (3.00 * (1-0.99000)))`),
						For:    "1h",
						Labels: map[string]string{"severity": "info", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
					{
						Alert: "RHOAMUserSsoAvailability6hto3dErrorBudgetBurn",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMUserSsoAvailabilityAlert,
							"message": "High 6h and 3d error budget burn for RHOAM SSO User",
						},
						Expr: intstr.FromString(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[6h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[6h])) > (6.00 * (1-0.99000)))
							and 
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso", code="5xx"}[3d]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace="redhat-rhoam-user-sso"}[3d])) > (6.00 * (1-0.99000)))`),
						For:    "3h",
						Labels: map[string]string{"severity": "info", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
				},
			},
		},
	}
}
