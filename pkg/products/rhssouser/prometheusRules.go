package rhssouser

import (
	"fmt"
	"strings"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertsReconciler(logger l.Logger, installType string, namespace string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]

	alertName := "user-sso-ksm-endpoint-alerts"
	operatorAlertName := "user-sso-operator-ksm-endpoint-alerts"
	userSsoAlerts := "rhssouser-general"

	return &resources.AlertReconcilerImpl{
		ProductName:  "RHSSO User",
		Installation: r.Installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertName,
				GroupName: "user-rhsso-endpoint.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "RHOAMUserRhssoKeycloakServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{endpoint='keycloak', namespace='%s'})", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMUserRhssoKeycloakDiscoveryServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{endpoint='keycloak-discovery', namespace='%s'})", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: userSsoAlerts,
				GroupName: "rhoam-general-user-rhsso.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "KeycloakInstanceNotAvailable",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlKeycloakInstanceNotAvailable,
							"message": fmt.Sprintf(`Keycloak instance in namespace %s has not been available for the last 5 minutes.`, r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf(`sum(kube_pod_status_ready{condition="true",namespace="%[1]s",pod=~"keycloak.*"} * on(pod, namespace) group_left() kube_pod_status_phase{phase="Running",namespace="%[1]s"}) < 1 OR absent(kube_pod_status_ready{condition="true",namespace="%[1]s",pod=~"keycloak.*"})`, r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
				},
			},

			{
				AlertName: operatorAlertName,
				Namespace: namespace,
				GroupName: "user-rhsso-operator-endpoint.rules",
				Rules: []monv1.Rule{
					{
						Alert: "RHOAMUserRhssoOperatorRhmiRegistryCsMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{endpoint='rhmi-registry-cs', namespace='%s'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMUserRhssoKeycloakOperatorMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{endpoint='rhsso-operator-metrics', namespace='%s'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},

			//SLO-availability-alerts:
			//https://sre.google/workbook/alerting-on-slos/
			//https://promtools.dev/alerts/errors
			{
				AlertName: "user-sso-slo-availability-alerts",
				Namespace: namespace,
				GroupName: "user-sso-slo-availability.rules",
				Rules: []monv1.Rule{
					{
						Alert: fmt.Sprintf("%sUserSsoAvailability5mto1hErrorBudgetBurn", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlSloUserSsoAvailabilityAlert,
							"message": "High 5m and 1h error budget burn for SSO User",
						},
						Expr: intstr.FromString(fmt.Sprintf(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[5m]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[5m]))) > (14.40 * (1-0.99000))
							and
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[1h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[1h]))) > (14.40 * (1-0.99000))`, r.Config.GetNamespace())),
						For:    "2m",
						Labels: map[string]string{"severity": "warning", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
					{
						Alert: fmt.Sprintf("%sUserSsoAvailability30mto6hErrorBudgetBurn", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlSloUserSsoAvailabilityAlert,
							"message": "High 30m and 6h error budget burn for SSO User",
						},
						Expr: intstr.FromString(fmt.Sprintf(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[30m]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[30m]))) > (6.00 * (1-0.99000))
							and 
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[6h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[6h]))) > (6.00 * (1-0.99000))`, r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
					{
						Alert: fmt.Sprintf("%sUserSsoAvailability2hto1dErrorBudgetBurn", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlSloUserSsoAvailabilityAlert,
							"message": "High 2h and 1d error budget burn for SSO User",
						},
						Expr: intstr.FromString(fmt.Sprintf(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[2h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[2h]))) > (3.00 * (1-0.99000))
							and
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[1d]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[1d]))) > (3.00 * (1-0.99000))`, r.Config.GetNamespace())),
						For:    "1h",
						Labels: map[string]string{"severity": "warning", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
					{
						Alert: fmt.Sprintf("%sUserSsoAvailability6hto3dErrorBudgetBurn", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlSloUserSsoAvailabilityAlert,
							"message": "High 6h and 3d error budget burn for SSO User",
						},
						Expr: intstr.FromString(fmt.Sprintf(`
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[6h]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[6h]))) > (6.00 * (1-0.99000))
							and 
							sum( sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s', code="5xx"}[3d]))
								/sum(rate(haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace='%[1]s'}[3d]))) > (6.00 * (1-0.99000))`, r.Config.GetNamespace())),
						For:    "3h",
						Labels: map[string]string{"severity": "warning", "route": "keycloak", "service": "keycloak", "product": installationName},
					},
				},
			},
		},
	}
}
