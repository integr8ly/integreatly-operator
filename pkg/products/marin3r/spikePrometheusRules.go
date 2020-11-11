package marin3r

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newSpikeLimitAlertsReconciler() resources.AlertReconciler {
	return &resources.AlertReconcilerImpl{
		ProductName:  "3scale",
		Installation: r.installation,
		Logger:       r.logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "rate-limit-spike",
				GroupName: "ratelimit-spike.rules",
				Namespace: r.Config.GetNamespace(),
				Interval:  "30m",
				Rules: []monitoringv1.Rule{
					{
						Alert: fmt.Sprintf("RHOAMApiUsageOverLimit"),
						Expr: intstr.FromString(
							fmt.Sprintf("max_over_time((increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m]))[30m:]) > 13888 * 2"),
						),
						Annotations: map[string]string{
							"message": fmt.Sprintf("hard limit breached at least once in the last 30 minutes"),
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
				},
			},
		},
	}
}
