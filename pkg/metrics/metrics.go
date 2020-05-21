package metrics

import (
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
)

func SetRhmiVersions(stage string, version string, toVersion string) {
	RHMIVersion.WithLabelValues(stage, version, toVersion).Add(1)
}
