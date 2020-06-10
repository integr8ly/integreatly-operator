package amqonline

import (
	"context"
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileSloAlerts(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	keycloakServicePortCount := 2

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-amq-online-slo",
			Namespace: r.Config.GetNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: fmt.Sprintf("AMQOnlineConsoleAvailable"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("AMQ-SLO-1.1: AMQ Online console is not available in namespace '%s'", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address_available{endpoint='console',namespace='%s'}==2)", r.Config.GetNamespace())),
			For:    "300s",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: fmt.Sprintf("AMQOnlineKeycloakAvailable"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("AMQ-SLO-1.4: Keycloak is not available in namespace %s", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address_available{endpoint='standard-authservice',namespace='%s'}==%v)", r.Config.GetNamespace(), keycloakServicePortCount)),
			For:    "300s",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: fmt.Sprintf("AMQOnlineOperatorAvailable"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("AMQ-SLO-1.5: amq-online(enmasse) operator is not available in namespace %s", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("absent(kube_pod_status_ready{condition='true',namespace='%s',pod=~'enmasse-operator-.*'}==1)", r.Config.GetNamespace())),
			For:    "300s",
			Labels: map[string]string{"severity": "critical"},
		}}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "amqonline.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating enmasse PrometheusRule: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKubeStateMetricsEndpointAvailableAlerts(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-endpoint-alerts",
			Namespace: r.Config.GetNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIAMQOnlineNoneAuthServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='none-authservice'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIAMQOnlineAddressSpaceControllerServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='address-space-controller'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIAMQOnlineConsoleServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='console'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIAMQOnlineRegistryCsServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetNamespace())),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIAMQOnlineStandardAuthServiceServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='standard-authservice'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIAMQOnlineEnmasseOperatorMetricsServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='enmasse-operator-metrics'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		}}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "amqonline-endpoint.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating enmasse PrometheusRule: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
