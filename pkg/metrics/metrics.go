package metrics

import (
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/prometheus/client_golang/prometheus"
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

	RHMIVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhmi_version",
			Help: "RHMI versions",
		},
		[]string{
			"stage",
			"version",
			"to_version",
		},
	)

	RHMIStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhmi_status",
			Help: "RHMI status of an installation",
		},
		[]string{
			"stage",
		},
	)
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

// SetRHMIStatus exposes rhmi_status metric for each stage
func SetRHMIStatus(installation *integreatlyv1alpha1.RHMI) {
	RHMIStatus.Reset()
	if string(installation.Status.Stage) != "" {
		RHMIStatus.With(prometheus.Labels{"stage": string(installation.Status.Stage)}).Set(float64(1))
	}
}

func SetRhmiVersions(stage string, version string, toVersion string, firstInstallTimestamp int64) {
	RHMIVersion.Reset()
	RHMIVersion.WithLabelValues(stage, version, toVersion).Set(float64(firstInstallTimestamp))
}
