package metrics

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/version"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// Custom metrics
var (
	OperatorVersion = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "integreatly_version_info",
			Help: "Integreatly operator information",
			ConstLabels: prometheus.Labels{
				"operator_version": version.GetVersion(),
				"version":          version.GetVersion(),
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

	RHOAMVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_version",
			Help: "RHOAM versions",
		},
		[]string{
			"stage",
			"status",
			"version",
			"to_version",
			"externalID",
		},
	)

	RHOAMStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_status",
			Help: "RHOAM status of an installation",
		},
		[]string{
			"stage",
		},
	)

	RHOAMAlertsSummary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_alerts_summary",
			Help: "RHOAM alerts summary, excludes DeadManSwitch",
		},
		[]string{
			"alert",
			"severity",
			"state",
			"externalID",
		},
	)

	RHOAMCluster = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_cluster",
			Help: "Provides Cluster information for the RHOAM installation",
		},
		[]string{
			"type",
			"externalID",
			"version",
		},
	)

	ThreeScaleUserAction = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "threescale_user_action",
			Help: "Status of user CRUD action in 3scale",
		},
		[]string{
			"username",
			"action",
		},
	)

	Quota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_quota",
			Help: "Status of the current quota config",
		},
		[]string{
			"quota",
			"toQuota",
		},
	)

	CustomDomain = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_custom_domain",
			Help: "Custom Domain Status. " +
				"active - indicating whether RHOAM was installed with custom domain enabled " +
				"Labels indicating portal availability: 1) system-master 2) system-developer 3) system-provider",
		},
		[]string{
			LabelActive,
			LabelSystemMaster,
			LabelSystemDeveloper,
			LabelSystemProvider,
		},
	)

	TotalNumTenants = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "total_num_tenants",
			Help: "Total number of tenants (APIManagementTenant CRs) on the cluster",
		},
	)

	NumReconciledTenants = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "num_reconciled_tenants",
			Help: "Number of reconciled tenants (APIManagementTenant CRs) on the cluster",
		},
	)

	NumFailedTenants = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "num_failed_tenants",
			Help: "Number of tenants (APIManagementTenant CRs) on the cluster that didn't reconcile",
		},
	)

	NoActivated3ScaleTenantAccount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "no_activated_3scale_tenant_account",
			Help: "Users/Tenants who do not have an activated 3Scale account",
		},
		[]string{
			"username",
		},
	)

	InstallationControllerReconcileDelayed = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "installation_controller_reconcile_delayed",
			Help: "Measures if the last reconcile of the installation controller is delayed",
		},
	)
)

const (
	LabelActive          = "active"
	LabelSystemMaster    = "system_master"
	LabelSystemDeveloper = "system_developer"
	LabelSystemProvider  = "system_provider"
)

type PortalInfo struct {
	Host       string
	PortalName string
	Status     int
}

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

	RHOAMStatus.Reset()
	if string(installation.Status.Stage) != "" {
		RHOAMStatus.With(prometheus.Labels{"stage": string(installation.Status.Stage)}).Set(float64(1))
	}
}

func SetRhmiVersions(stage string, version string, toVersion string, externalID string, firstInstallTimestamp int64) {
	RHMIVersion.Reset()
	RHMIVersion.WithLabelValues(stage, version, toVersion).Set(float64(firstInstallTimestamp))

	RHOAMVersion.Reset()
	status := resources.InstallationState(version, toVersion)
	RHOAMVersion.WithLabelValues(stage, status, version, toVersion, externalID).Set(float64(firstInstallTimestamp))
}

func SetRHOAMAlertsSummary(alerts resources.AlertMetrics, externalID string) {
	RHOAMAlertsSummary.Reset()
	for key, value := range alerts {
		RHOAMAlertsSummary.With(prometheus.Labels{
			"alert":      string(key.Name),
			"severity":   string(key.Severity),
			"state":      string(key.State),
			"externalID": externalID,
		}).Set(float64(value))
	}
}

func SetRHOAMCluster(cluster string, externalID string, version string, value int64) {
	RHOAMCluster.Reset()
	RHOAMCluster.With(prometheus.Labels{
		"type":       cluster,
		"externalID": externalID,
		"version":    version,
	}).Set(float64(value))
}

func SetThreeScaleUserAction(httpStatus int, username, action string) {
	ThreeScaleUserAction.WithLabelValues(username, action).Set(float64(httpStatus))
}

func ResetThreeScaleUserAction() {
	ThreeScaleUserAction.Reset()
}

func SetTotalNumTenants(numTenants int) {
	TotalNumTenants.Set(float64(numTenants))
}

func SetNumReconciledTenants(numTenants int) {
	NumReconciledTenants.Set(float64(numTenants))
}

func SetNumFailedTenants(numTenants int) {
	NumFailedTenants.Set(float64(numTenants))
}

func ResetNoActivated3ScaleTenantAccount() {
	NoActivated3ScaleTenantAccount.Reset()
}

func SetNoActivated3ScaleTenantAccount(username string) {
	NoActivated3ScaleTenantAccount.WithLabelValues(username).Set(float64(1))
}

func SetQuota(quota string, toQuota string) {
	Quota.Reset()
	Quota.WithLabelValues(quota, toQuota).Set(float64(1))
}

// node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate was renamed in 4.9
func GetContainerCPUMetric(ctx context.Context, serverClient k8sclient.Client, l l.Logger) (string, error) {
	before49, err := resources.ClusterVersionBefore49(ctx, serverClient, l)
	if err != nil {
		return "", err
	}
	if before49 {
		return "node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate", nil
	} else {
		return "node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate", nil
	}
}

func SetCustomDomain(active bool, portals map[string]PortalInfo, value float64) {
	labels := prometheus.Labels{
		LabelActive:          strconv.FormatBool(active),
		LabelSystemMaster:    "false",
		LabelSystemDeveloper: "false",
		LabelSystemProvider:  "false",
	}
	if portals != nil {
		for key, portal := range portals {
			labels[key] = strconv.FormatBool(portal.Status == http.StatusOK)
		}
	}
	CustomDomain.Reset()
	CustomDomain.With(labels).Set(value)
}
