package metrics

import (
	"context"
	"fmt"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/cluster"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/version"
	prometheusApi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	prometheusConfig "github.com/prometheus/common/config"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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

	RHOAMInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_spec",
			Help: "RHOAM info variables",
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

	// RhoamStateMetric metric exported to telemeter. DO NOT increase cardinality
	RhoamStateMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_state",
			Help: "Capture currently installed/upgrading RHOAM version. This will facilitate a snapshot dashboard to SRE of versions and upgrade status across the fleet.",
		},
		[]string{
			"status",    // "in progress/complete
			"upgrading", // "true/false"
			"version",   // "x.y.z"
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

	// RHOAMProductStatus tracks individual product reconciliation status
	RHOAMProductStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rhoam_product_status",
			Help: "RHOAM individual product reconciliation status",
		},
		[]string{
			"product",
			"stage",
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
				"active - indicating whether RHOAM was installed with custom domain enabled",
		},
		[]string{LabelActive},
	)

	ThreeScalePortals = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "threescale_portals",
			Help: "ThreeScale portals availability. " +
				"Labels indicating portal availability: 1) system-master 2) system-developer 3) system-provider",
		},
		[]string{
			LabelSystemMaster,
			LabelSystemDeveloper,
			LabelSystemProvider,
		},
	)

	TenantsSummary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenants_summary",
			Help: "Summary of APIManagementTenant CRs",
		},
		[]string{
			"tenantName",
			"tenantNamespace",
			"provisioningStatus",
			"lastError",
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
	SloDays              = 7
)

type PortalInfo struct {
	Host        string
	Ingress     string
	PortalName  string
	IsAvailable bool
}

type RhoamState struct {
	Status    integreatlyv1alpha1.StatusPhase
	Upgrading bool
	Version   string
}

// SetInfo exposes operator info metrics with labels from the installation CR
func SetInfo(installation *integreatlyv1alpha1.RHMI) {
	RHOAMInfo.WithLabelValues(installation.Spec.UseClusterStorage,
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

// SetStatus exposes RHOAM_status metric for each stage
func SetStatus(installation *integreatlyv1alpha1.RHMI) {
	RHOAMStatus.Reset()
	if string(installation.Status.Stage) != "" {
		RHOAMStatus.With(prometheus.Labels{"stage": string(installation.Status.Stage)}).Set(float64(1))
	}
}

// SetProductStatus exposes RHOAM product-specific status metrics
func SetProductStatus(installation *integreatlyv1alpha1.RHMI) {
	RHOAMProductStatus.Reset()

	// Check if installation stage exists and has products
	if installationStage, exists := installation.Status.Stages["installation"]; exists {
		for productName, product := range installationStage.Products {
			// Set metric value based on product phase
			// 0 = completed, 1 = in progress/error
			var value float64
			if product.Phase == integreatlyv1alpha1.PhaseCompleted {
				value = 0
			} else {
				value = 1
			}

			RHOAMProductStatus.With(prometheus.Labels{
				"product": string(productName),
				"stage":   string(product.Phase),
			}).Set(value)
		}
	}
}

func SetVersions(stage string, version string, toVersion string, externalID string, firstInstallTimestamp int64) {
	RHOAMVersion.Reset()
	status := resources.InstallationState(version, toVersion)
	RHOAMVersion.WithLabelValues(stage, status, version, toVersion, externalID).Set(float64(firstInstallTimestamp))
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

func SetTenantsSummary(tenants *integreatlyv1alpha1.APIManagementTenantList) {
	TenantsSummary.Reset()
	for _, tenant := range tenants.Items {
		TenantsSummary.With(prometheus.Labels{
			"tenantName":         tenant.Name,
			"tenantNamespace":    tenant.Namespace,
			"provisioningStatus": string(tenant.Status.ProvisioningStatus),
			"lastError":          tenant.Status.LastError,
		}).Set(float64(tenant.CreationTimestamp.Unix()))
	}
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

// GetContainerCPUMetric node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate was renamed in 4.9
func GetContainerCPUMetric(ctx context.Context, serverClient k8sclient.Client, l l.Logger) (string, error) {
	before49, err := cluster.ClusterVersionBefore49(ctx, serverClient, l)
	if err != nil {
		return "", err
	}
	if before49 {
		return "node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate", nil
	} else {
		return "node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate", nil
	}
}

func SetCustomDomain(active bool, value float64) {
	labels := prometheus.Labels{LabelActive: strconv.FormatBool(active)}
	CustomDomain.Reset()
	CustomDomain.With(labels).Set(value)
}

func SetThreeScalePortals(portals map[string]PortalInfo, value float64) {
	labels := prometheus.Labels{
		LabelSystemMaster:    "false",
		LabelSystemDeveloper: "false",
		LabelSystemProvider:  "false",
	}
	for key, portal := range portals {
		labels[key] = strconv.FormatBool(portal.IsAvailable)
	}
	ThreeScalePortals.Reset()
	ThreeScalePortals.With(labels).Set(value)
}

func SetRhoamState(status RhoamState) {
	labels := prometheus.Labels{
		"status":    string(status.Status),
		"upgrading": strconv.FormatBool(status.Upgrading),
		"version":   status.Version,
	}
	RhoamStateMetric.Reset()
	RhoamStateMetric.With(labels).Set(1)
}

func GetRhoamState(cr *integreatlyv1alpha1.RHMI) (RhoamState, error) {
	status := RhoamState{}

	if cr == nil {
		return status, fmt.Errorf("funtion parameter \"cr\" is nil")
	}

	if cr.Status.Stage == integreatlyv1alpha1.CompleteStage {
		status.Status = integreatlyv1alpha1.PhaseCompleted
	} else {
		status.Status = integreatlyv1alpha1.PhaseInProgress
	}

	if cr.Status.Version != "" && cr.Status.ToVersion != "" {
		status.Upgrading = true
	} else {
		status.Upgrading = false
	}

	if cr.Status.Version == "" {
		status.Version = cr.Status.ToVersion
	} else {
		status.Version = cr.Status.Version
	}
	return status, nil
}

type inlineSecret struct {
	text string
}

func (s *inlineSecret) Fetch(ctx context.Context) (string, error) {
	return s.text, nil
}
func (s *inlineSecret) Description() string { return "inline" }
func (s *inlineSecret) Immutable() bool     { return true }

func GetApiClient(route string, token prometheusConfig.Secret) (prometheusv1.API, error) {
	client, err := prometheusApi.NewClient(prometheusApi.Config{
		Address: route,
		RoundTripper: prometheusConfig.NewAuthorizationCredentialsRoundTripper(
			"Authorization",
			&inlineSecret{string(token)},
			prometheusApi.DefaultRoundTripper,
		),
	})
	if err != nil {
		return nil, err
	}

	v1api := prometheusv1.NewAPI(client)
	return v1api, nil
}
