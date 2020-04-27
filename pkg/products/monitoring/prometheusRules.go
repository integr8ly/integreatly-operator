package monitoring

import (
	"context"
	"fmt"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"k8s.io/apimachinery/pkg/util/intstr"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"






	
)
func (r *Reconciler) reconcilePrometheusRule(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backup-monitoring-alerts",
			Namespace: "redhat-rhmi-middleware-monitoring-operator",
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: fmt.Sprintf("JobRunningTimeExceeded"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf(" Job {{ $labels.namespace }} / {{ $labels.job  }} has been running for longer than 300 seconds"),
				},	
			Expr: intstr.FromString(fmt.Sprintf("time() - (max(kube_job_status_active * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name) * ON(job_name) GROUP_RIGHT() max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name, namespace, label_cronjob_name) > 0) > 300 ")),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
			
		},
		{
			Alert: fmt.Sprintf("JobRunningTimeExceeded"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf(" Job {{ $labels.namespace }} / {{ $labels.job  }} has been running for longer than 600 seconds"),
				},	
			Expr: intstr.FromString(fmt.Sprintf("time() - (max(kube_job_status_active * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name) * ON(job_name) GROUP_RIGHT() max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name, namespace, label_cronjob_name) > 0) > 600 ")),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
			
		},
		{
			Alert: fmt.Sprintf("CronJobSuspended"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf(" CronJob {{ $labels.namespace  }}/{{ $labels.cronjob }} is suspended"),
				},	
			Expr: intstr.FromString(fmt.Sprintf("kube_cronjob_labels{ label_monitoring_key='middleware' } * ON (cronjob) GROUP_RIGHT() kube_cronjob_spec_suspend > 0")),
			For:    "60s",
			Labels: map[string]string{"severity": "critical"},
			
		},
		{
			Alert: fmt.Sprintf("CronJobNotRunInThreshold"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf(" CronJob {{ $labels.namespace }}/{{ $labels.label_cronjob_name }} has not started a Job in 25 hours"),
				},	
			Expr: intstr.FromString(fmt.Sprintf("(time() - (max( kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'} ) BY (job_name, label_cronjob_name) == ON(label_cronjob_name) GROUP_LEFT() max( kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'} ) BY (label_cronjob_name))) > 60*60*25")),
			Labels: map[string]string{"severity": "critical"},	
		},
		{
			Alert: fmt.Sprintf("CronJobsFailed"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("Job {{ $labels.namespace  }}/{{  $labels.job  }} has failed"),
				},	
			Expr: intstr.FromString(fmt.Sprintf("clamp_max(max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'} ) BY (job_name, label_cronjob_name, namespace) == ON(label_cronjob_name) GROUP_LEFT() max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (label_cronjob_name), 1) * ON(job_name) GROUP_LEFT() kube_job_status_failed > 0")),
			For: "5m",
			Labels: map[string]string{"severity": "critical"},	
		}}
		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, rule, func() error {
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
		logrus.Infof("Phase: %s reconcilePrometheusAlerts", err)

		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating backup PrometheusRule: %w", err)
		
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
