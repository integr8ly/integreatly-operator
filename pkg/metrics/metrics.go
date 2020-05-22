package metrics

import (
	"fmt"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/prometheus/client_golang/prometheus"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Custom metrics
var (
	OperatorVersion = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "integreatly_version_info",
			Help: "Integreatly operator information",
			ConstLabels: prometheus.Labels{
				"operator_version": version.Version,
				"version":          version.IntegreatlyVersion,
			},
		},
	)

	RHMIStatusAvailable = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rhmi_status_available",
			Help: "RHMI status available",
		},
	)

	RHMIInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhmi_spec",
			Help: "RHMI info variables",
		},
		[]string{
			"use_cluster_storage",
			"master_url",
			"installation_type",
			"operator_name",
			"namespace",
			"namespace_prefix",
			"operators_in_product_namespace",
			"routing_subdomain",
			"self_signed_certs",
		},
	)

	RHMIStatus *prometheus.GaugeVec = nil

	rhmiStatusLabels = []string{
		"operator_name",
		"namespace",
		"last_error",
		"preflight_message",
		"preflight_status",
		"stage",
	}
)

// SetRHMIInfo exposes rhmi info metrics with labels from the installation CR
func SetRHMIInfo(installation *integreatlyv1alpha1.RHMI) {
	RHMIInfo.WithLabelValues(installation.Spec.UseClusterStorage,
		installation.Spec.MasterURL,
		installation.Spec.Type,
		installation.GetName(),
		installation.GetNamespace(),
		installation.Spec.NamespacePrefix,
		fmt.Sprintf("%t", installation.Spec.OperatorsInProductNamespace),
		installation.Spec.RoutingSubdomain,
		fmt.Sprintf("%t", installation.Spec.SelfSignedCerts),
	)
}

func SetRHMIStatus(installation *integreatlyv1alpha1.RHMI) {

	isCompleted := float64(0)
	if installation.Status.Stage == "complete" {
		isCompleted = float64(1)
	}

	// creates the metric labels with values
	labelsWithValue := make(map[string]string, len(rhmiStatusLabels))
	for _, label := range rhmiStatusLabels {
		labelsWithValue[label] = ""
	}

	// sets value from rhmi installation to labels
	labelsWithValue["operator_name"] = installation.GetName()
	labelsWithValue["namespace"] = installation.GetNamespace()
	labelsWithValue["last_error"] = installation.Status.LastError
	labelsWithValue["preflight_message"] = installation.Status.PreflightMessage
	labelsWithValue["preflight_status"] = string(installation.Status.PreflightStatus)
	labelsWithValue["stage"] = string(installation.Status.Stage)

	for _, stage := range installation.Status.Stages {
		for _, product := range stage.Products {
			labelsWithValue[SanitizeForPrometheusLabel(product.Name)] = string(product.Status)
		}
	}

	if RHMIStatus != nil {
		RHMIStatus.Reset()
		RHMIStatus.With(labelsWithValue).Set(isCompleted)
	}
}

func ExposeRHMIStatusMetric(stages []integreatlyv1alpha1.RHMIStageStatus) {

	if RHMIStatus == nil {
		for _, stage := range stages {
			for _, product := range stage.Products {
				rhmiStatusLabels = append(rhmiStatusLabels, SanitizeForPrometheusLabel(product.Name))
			}
		}

		RHMIStatus = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rhmi_status",
				Help: "RHMI status for when installation completes",
			},
			rhmiStatusLabels,
		)
		customMetrics.Registry.MustRegister(RHMIStatus)
	}
}

func SanitizeForPrometheusLabel(productName integreatlyv1alpha1.ProductName) string {
	if productName == integreatlyv1alpha1.Product3Scale {
		productName = "threescale"
	}
	return fmt.Sprintf("%s_status", strings.ReplaceAll(string(productName), "-", "_"))
}
