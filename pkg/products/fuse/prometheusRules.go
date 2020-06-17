package fuse

import (
	"context"
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileKubeStateMetricsEndpointAvailableAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-endpoint-alerts",
			Namespace: r.Config.GetNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIFuseOnlineBrokerAmqTcpServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='broker-amq-tcp'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIFuseOnlineSyndesisMetaServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='syndesis-meta'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIFuseOnlineSyndesisOauthproxyServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='syndesis-oauthproxy'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIFuseOnlineSyndesisPrometheusServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='syndesis-prometheus'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIFuseOnlineSyndesisServerServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='syndesis-server'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIFuseOnlineSyndesisUiServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='syndesis-ui'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		}}

	or, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "fuse-online-endpoint.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating fuse PrometheusRule: %w", err)
	}

	if or != controllerutil.OperationResultNone {
		r.logger.Infof("The operation result for fuse %s was %s", rule.Name, or)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKubeStateMetricsOperatorEndpointAvailableAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-endpoint-alerts",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIFuseOnlineOperatorRhmiRegistryCsServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIFuseOnlineOperatorSyndesisOperatorMetricsServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='syndesis-operator-metrics', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		}}

	or, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "fuse-online-operator-endpoint.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating fuse-online operator PrometheusRule: %w", err)
	}

	if or != controllerutil.OperationResultNone {
		r.logger.Infof("The operation result for fuse operator %s was %s", rule.Name, or)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
func (r *Reconciler) reconcileKubeStateMetricsFuseAlerts(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-fuse-online-alerts",
			Namespace: r.Config.GetNamespace(),
		},
	}

	var namespace = r.Config.GetNamespace()

	rules := []monitoringv1.Rule{
		{
			Alert: "FuseOnlineSyndesisServerInstanceDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlAlertsAndTroubleshooting,
				"message": "Fuse Online Syndesis Server instance {{ $labels.pod }} in namespace {{ $labels.namespace }} is down.",
			},
			Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%s'} * on (pod, namespace) kube_pod_labels{label_deploymentconfig='syndesis-server'})) or sum(kube_pod_status_ready{condition='true',namespace='%s'} * on (pod, namespace) kube_pod_labels{label_deploymentconfig='syndesis-server'}) < 1", namespace, namespace)),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "FuseOnlineSyndesisUIInstanceDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlAlertsAndTroubleshooting,
				"message": " Fuse Online Syndesis UI instance {{ $labels.pod }} in namespace {{ $labels.namespace }} is down.",
			},
			Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%s'} * on (pod, namespace) kube_pod_labels{label_deploymentconfig='syndesis-ui'})) or sum(kube_pod_status_ready{condition='true',namespace='%s'} * on (pod, namespace) kube_pod_labels{label_deploymentconfig='syndesis-ui'}) < 1", namespace, namespace)),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "general.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})

	if err != nil {
		r.logger.Errorf("Phase: %s reconcilePrometheusAlerts", err)

		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating backup PrometheusRule: %w", err)
	}
	if opRes != controllerutil.OperationResultNone {
		r.logger.Infof("The operation result for fuse %s was %s", rule.Name, opRes)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}
