package marin3r

import (
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const rejectedRequestsAlertExpr = "abs(clamp_min(increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m]) - %f, 0) / increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m]) - (increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_over_limit[1m]) / increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m]))) > 0.3"

func (r *Reconciler) newRejectedRequestsAlertsReconciler(logger l.Logger, installType string) (resources.AlertReconciler, error) {
	installationName := resources.InstallationNames[installType]

	limitPerMinute, err := config.ConvertRate(
		r.RateLimitConfig.Unit,
		config.Minute,
		int(r.RateLimitConfig.RequestsPerUnit),
	)
	if err != nil {
		return nil, err
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "3Scale",
		Installation: r.installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "rejected-requests",
				GroupName: "rejected-requests.rules",
				Namespace: r.Config.GetNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHOAMApiUsageRejectedRequestsMismatch",
						Annotations: map[string]string{
							"message": "The volume of rejected requests doesn't match the expected volume given the incoming requests and the configuration",
						},
						Expr:   intstr.FromString(fmt.Sprintf(rejectedRequestsAlertExpr, limitPerMinute)),
						Labels: map[string]string{"severity": "info", "product": installationName},
						For:    "30s",
					},
				},
			},
		},
	}, nil
}
