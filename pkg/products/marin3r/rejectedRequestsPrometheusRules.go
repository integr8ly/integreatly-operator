package marin3r

import (
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const rejectedRequestsAlertExpr = "abs(clamp_min(increase(limited_calls[1m]) - %f, 0) / (sum(increase(authorized_calls[1m])) + sum(increase(limited_calls[1m]))) - (increase(limited_calls[1m]) / (sum(increase(authorized_calls[1m])) + sum(increase(limited_calls[1m]))))) > 0.3"

func (r *Reconciler) newRejectedRequestsAlertsReconciler(logger l.Logger, installType, ns string) (resources.AlertReconciler, error) {
	installationName := resources.InstallationNames[installType]
	alertName := "marin3r-rejected-requests"

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
				AlertName: alertName,
				GroupName: "rejected-requests.rules",
				Namespace: ns,
				Rules: []monv1.Rule{
					{
						Alert: "RHOAMApiUsageRejectedRequestsMismatch",
						Annotations: map[string]string{
							"message": "The volume of rejected requests doesn't match the expected volume given the incoming requests and the configuration",
						},
						Expr:   intstr.FromString(fmt.Sprintf(rejectedRequestsAlertExpr, limitPerMinute)),
						Labels: map[string]string{"severity": "info", "product": installationName},
						For:    resources.DurationPtr("30s"),
					},
				},
			},
		},
	}, nil
}
