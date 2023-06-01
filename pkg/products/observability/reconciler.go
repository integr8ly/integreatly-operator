package observability

import (
	"context"
	"fmt"
	"regexp"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoringcommon"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/version"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	observability "github.com/redhat-developer/observability-operator/v4/api/v1"
	v1 "k8s.io/api/core/v1"
	flowcontrolv1alpha1 "k8s.io/api/flowcontrol/v1alpha1"
	rbac "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "observability"

	configMapNoInit              = "observability-operator-no-init"
	observabilityName            = "observability-stack"
	OpenshiftMonitoringNamespace = "openshift-monitoring"

	blackboxExporterPrefix                    = "blackbox-exporter"
	blackboxExporterAPIGroup                  = "rbac.authorization.k8s.io"
	clusterMonitoringPrometheusServiceAccount = "prometheus-k8s"
	clusterMonitoringNamespace                = "openshift-monitoring"
	serviceMonitorRoleBindingName             = "rhmi-prometheus-k8s"
	serviceMonitorRoleRefAPIGroup             = "rbac.authorization.k8s.io"
	serviceMonitorRoleRefName                 = "rhmi-prometheus-k8s"

	clonedServiceMonitorLabelKey   = "integreatly.org/cloned-servicemonitor"
	clonedServiceMonitorLabelValue = "true"
	labelSelector                  = "monitoring-key=middleware"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager config.ConfigReadWriter
	Config        *config.Observability
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	extraParams   map[string]string
	recorder      record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductObservability],
		string(integreatlyv1alpha1.VersionObservability),
		string(integreatlyv1alpha1.OperatorVersionObservability),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {

	ns := GetDefaultNamespace(installation.Spec.NamespacePrefix)
	productConfig, err := configManager.ReadObservability()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve observability config: %w", err)
	}

	productConfig.SetNamespacePrefix(installation.Spec.NamespacePrefix)
	productConfig.SetNamespace(ns)

	if installation.Spec.OperatorsInProductNamespace {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace())
	} else {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace() + "-operator")
	}

	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, err
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        productConfig,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, _ quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {

	r.log.Info("Start Observability reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if productNamespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, productNamespace, client)
		if !k8serr.IsNotFound(err) {
			// Mark OO CR for deletion.
			phase, err := r.deleteObservabilityCR(ctx, client, installation, productNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = resources.RemoveNamespace(ctx, installation, client, productNamespace, r.log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		// Check if operatorNamespace is still present before trying to delete it resources
		_, err = resources.GetNS(ctx, operatorNamespace, client)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, client, operatorNamespace, r.log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		// Delete ClusterRole and ClusterRoleBinding that were created for the blackbox exporter
		err = r.removeRoleandRoleBindingForBlackbox(ctx, client)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// If both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, operatorNamespace, client)
		_, productNSErr := resources.GetNS(ctx, productNamespace, client)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(productNSErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	}, r.log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, client, operatorNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, client, productNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileConfigMap(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s configmap which is required to disable observability operator initilisting it's own cr", configMapNoInit), err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxExporter(ctx, client, r.Config)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox exporter", err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, client, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.ObservabilitySubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, client, productNamespace, r.installation.Spec.NamespacePrefix)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to create components", err)
		return phase, err
	}

	phase, err = monitoringcommon.ReconcileAlertManagerSecrets(ctx, client, r.installation, r.Config.GetNamespace(), r.Config.GetAlertManagerRouteName())
	r.log.Infof("ReconcileAlertManagerConfigSecret", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("failed to reconcile alert manager config secret " + err.Error())
		}
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alert manager config secret", err)
		return phase, err
	}

	if string(r.Config.GetProductVersion()) != string(integreatlyv1alpha1.VersionObservability) {
		r.Config.SetProductVersion(string(integreatlyv1alpha1.VersionObservability))
		err := r.ConfigManager.WriteConfig(r.Config)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	phase, err = r.reconcileDashboards(ctx, client)
	r.log.Infof("reconcileDashboards", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("Failure reconciling dashboards " + err.Error())
		}
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile dashboards", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler(r.log, r.installation.Spec.Type, config.GetOboNamespace(r.installation.Namespace)).ReconcileAlerts(ctx, client)
	r.log.Infof("reconcilePrometheusRule", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	// creates an alert to check for the presents of sendgrid smtp secret
	phase, err = resources.CreateSmtpSecretExists(ctx, client, installation)
	r.log.Infof("CreateSmtpSecretExistsRule", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile SendgridSmtpSecretExists alert", err)
		return phase, err
	}

	// creates an alert to check for the presents of DeadMansSnitch secret
	phase, err = resources.CreateDeadMansSnitchSecretExists(ctx, client, installation)
	r.log.Infof("create DeadMansSnitch secret alerting rule", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile DeadMansSnitchSecretExists alert", err)
		return phase, err
	}

	// creates an alert to check for the presents of addon-managed-api-service-parameters secret
	phase, err = resources.CreateAddonManagedApiServiceParametersExists(ctx, client, installation)
	r.log.Infof("create addon-managed-api-service-parameters secret alerting rule", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile AddonManagedApiServiceParametersExists alert", err)
		return phase, err
	}

	phase, err = r.reconcileMonitoring(ctx, client)
	r.log.Infof("reconcileMonitoring", l.Fields{"status": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("failed to reconcile: " + err.Error())
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile: ", err)
		}
		return phase, err
	}

	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Create a ConfigMap in the operator namespace to prevent observability CR from being created in the operator ns.
	cfgMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapNoInit,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, serverClient, cfgMap, func() error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if op == controllerutil.OperationResultUpdated || op == controllerutil.OperationResultCreated {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}
	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient k8sclient.Client, productNamespace string, nsPrefix string) (integreatlyv1alpha1.StatusPhase, error) {

	oo := &observability.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: productNamespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, serverClient, oo, func() error {
		overrideSelectors := true
		disabled := true

		oo.Spec.AlertManagerDefaultName = r.Config.GetAlertManagerOverride()
		oo.Spec.GrafanaDefaultName = r.Config.GetGrafanaOverride()
		oo.Spec.PrometheusDefaultName = r.Config.GetPrometheusOverride()

		oo.Spec.ConfigurationSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"monitoring-key": r.Config.GetLabelSelector(),
			},
			MatchExpressions: nil,
		}

		oo.Spec.Storage = &observability.Storage{
			PrometheusStorageSpec: &prometheus.StorageSpec{
				VolumeClaimTemplate: prometheus.EmbeddedPersistentVolumeClaim{
					Spec: v1.PersistentVolumeClaimSpec{
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								"storage": resource.MustParse(r.Config.GetPrometheusStorageRequest()),
							},
						},
					},
				},
			},
			AlertManagerStorageSpec: &prometheus.StorageSpec{
				VolumeClaimTemplate: prometheus.EmbeddedPersistentVolumeClaim{
					Spec: v1.PersistentVolumeClaimSpec{
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								"storage": resource.MustParse(r.Config.GetAlertManagerStorageRequest()),
							},
						},
					},
				},
			},
		}

		oo.Spec.Retention = r.Config.GetPrometheusRetention()

		if oo.Spec.SelfContained == nil {
			oo.Spec.SelfContained = &observability.SelfContained{}
		}

		bearerToken := r.getBlackboxExporterServiceAccountToken(ctx, serverClient, productNamespace)
		if bearerToken != "" {
			oo.Spec.SelfContained.BlackboxBearerTokenSecret = bearerToken
		}
		oo.Spec.SelfContained.OverrideSelectors = &overrideSelectors
		oo.Spec.SelfContained.DisableRepoSync = &disabled
		oo.Spec.SelfContained.DisableObservatorium = &disabled
		oo.Spec.SelfContained.DisablePagerDuty = &disabled
		oo.Spec.SelfContained.DisableDeadmansSnitch = &disabled
		oo.Spec.SelfContained.DisableBlackboxExporter = nil
		oo.Spec.SelfContained.FederatedMetrics = []string{
			"'kubelet_volume_stats_used_bytes{endpoint=\"https-metrics\",namespace=~\"" + nsPrefix + ".*\"}'",
			"'kubelet_volume_stats_available_bytes{endpoint=\"https-metrics\",namespace=~\"" + nsPrefix + ".*\"}'",
			"'kubelet_volume_stats_capacity_bytes{endpoint=\"https-metrics\",namespace=~\"" + nsPrefix + ".*\"}'",
			"'haproxy_backend_http_responses_total{route=~\"^keycloak.*\",exported_namespace=~\"" + nsPrefix + ".*sso$\"}'",
			"'{ service=\"kube-state-metrics\" }'",
			"'{ service=\"node-exporter\" }'",
			"'{ __name__=~\"node_namespace_pod_container:.*\" }'",
			"'{ __name__=~\"node:.*\" }'",
			"'{ __name__=~\"instance:.*\" }'",
			"'{ __name__=~\"container_memory_.*\" }'",
			"'{ __name__=~\":node_memory_.*\" }'",
			"'{ __name__=~\"csv_.*\" }'",
		}
		oo.Spec.SelfContained.PodMonitorLabelSelector = nil
		oo.Spec.SelfContained.PodMonitorNamespaceSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "monitoring-key",
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						r.Config.GetLabelSelector(),
					},
				},
			},
		}
		oo.Spec.SelfContained.ServiceMonitorLabelSelector = nil
		oo.Spec.SelfContained.ServiceMonitorNamespaceSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "monitoring-key",
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						r.Config.GetLabelSelector(),
					},
				},
			},
		}
		oo.Spec.SelfContained.RuleLabelSelector = nil
		oo.Spec.SelfContained.RuleNamespaceSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "monitoring-key",
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						r.Config.GetLabelSelector(),
					},
				},
			},
		}
		oo.Spec.SelfContained.ProbeLabelSelector = nil
		oo.Spec.SelfContained.ProbeNamespaceSelector = nil
		oo.Spec.SelfContained.GrafanaDashboardLabelSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      r.Config.GetLabelSelectorKey(),
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						r.Config.GetLabelSelector(),
					},
				},
			},
		}
		oo.Spec.SelfContained.AlertManagerConfigSecret = config.AlertManagerConfigSecretName
		oo.Spec.SelfContained.PrometheusVersion = r.Config.GetPrometheusVersion()
		oo.Spec.SelfContained.AlertManagerVersion = r.Config.GetAlertManagerVersion()
		oo.Spec.SelfContained.AlertManagerResourceRequirement = r.Config.GetAlertManagerResourceRequirements()
		oo.Spec.SelfContained.GrafanaResourceRequirement = r.Config.GetGrafanaResourceRequirements()
		oo.Spec.SelfContained.PrometheusResourceRequirement = r.Config.GetPrometheusResourceRequirements()
		oo.Spec.SelfContained.PrometheusOperatorResourceRequirement = r.Config.GetPrometheusOperatorResourceRequirements()
		oo.Spec.SelfContained.DisableLogging = &disabled
		oo.Spec.ResyncPeriod = "1h"

		return nil
	},
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if op == controllerutil.OperationResultCreated {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: oo.Name, Namespace: oo.Namespace}, oo)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if oo.Status.StageStatus == observability.ResultFailed {
		return integreatlyv1alpha1.PhaseFailed, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling subscription")

	target := marketplace.Target{
		SubscriptionName: constants.ObservabilitySubscriptionName,
		Namespace:        operatorNamespace,
	}

	catalogSourceReconciler, err := r.GetProductDeclaration().PrepareTarget(
		r.log,
		serverClient,
		marketplace.CatalogSourceName,
		&target,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{productNamespace},
		r.preUpgradeBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	return backup.NewNoopBackupExecutor()
}

func (r *Reconciler) deleteObservabilityCR(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, targetNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	// If the installation is NOT marked for deletion, return without deleting observability CR
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	isHiveManaged, err := addon.OperatorIsHiveManaged(ctx, serverClient, inst)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if isHiveManaged {
		// proceed after the dms secret is deleted by the deadmanssnitch-operator to prevent alert false positive
		dmsSecret, err := monitoringcommon.GetDMSSecret(ctx, serverClient, *inst)
		if err != nil && !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("unexpected error retrieving dead man's snitch secret: %w", err)
		}
		if dmsSecret != "" {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("dead man's snitch secret is still present, requeing")
		}
	}

	oo := &observability.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: targetNamespace,
		},
	}

	// Get the observability CR; return if not found
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: oo.Name, Namespace: oo.Namespace}, oo)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Mark the observability CR for deletion
	err = serverClient.Delete(ctx, oo)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) reconcileDashboards(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	for _, dashboard := range r.Config.GetDashboards(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {
		err := r.reconcileGrafanaDashboards(ctx, serverClient, dashboard)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update grafana dashboard %s: %w", dashboard, err)
		}
		r.log.Infof("Reconcile successful", l.Fields{"grafanaDashboard": dashboard})
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileGrafanaDashboards(ctx context.Context, serverClient k8sclient.Client, dashboard string) (err error) {

	//clusterVersion
	containerCpuMetric, err := metrics.GetContainerCPUMetric(ctx, serverClient, r.log)
	if err != nil {
		return err
	}

	grafanaDB := &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboard,
			Namespace: r.Config.GetNamespace(),
		},
	}
	specJSON, _, err := monitoringcommon.GetSpecDetailsForDashboard(dashboard, r.installation, containerCpuMetric)
	if err != nil {
		return err
	}

	pluginList := monitoringcommon.GetPluginsForGrafanaDashboard(dashboard)

	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, grafanaDB, func() error {
		grafanaDB.Labels = map[string]string{
			"monitoring-key": r.Config.GetLabelSelector(),
		}
		grafanaDB.Spec = grafanav1alpha1.GrafanaDashboardSpec{
			Json: specJSON,
		}
		if len(pluginList) > 0 {
			grafanaDB.Spec.Plugins = pluginList
		}
		return nil
	})
	if err != nil {
		return err
	}
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result", l.Fields{"grafanaDashboard": grafanaDB.Name, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileMonitoring(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	//Get list of service monitors in the namespace that has
	//label "integreatly.org/cloned-servicemonitor" set to "true"
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
		k8sclient.MatchingLabels(getClonedServiceMonitorLabel()),
	}

	//Get list of service monitors in the observability namespace
	monSermonMap, err := r.getServiceMonitors(ctx, client, listOpts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	//Get the list of namespaces with the given label selector "monitoring-key=middleware"
	namespaces, err := r.getMWMonitoredNamespaces(ctx, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name != r.Config.GetNamespace() {
			//Get list of service monitors in each name space
			listOpts := []k8sclient.ListOption{
				k8sclient.InNamespace(ns.Name),
			}
			serviceMonitorsMap, err := r.getServiceMonitors(ctx, client, listOpts)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}

		copyOut:
			for _, sm := range serviceMonitorsMap {

				// don't copy the one for redhat-rhoam-middleware-monitoring-operator as that namespace is removed now
				// delete it from the cluster
				// consider parameterising this to rhoam

				for _, s := range sm.Spec.NamespaceSelector.MatchNames {
					if s == fmt.Sprintf("%smiddleware-monitoring-operator", r.Config.GetNamespacePrefix()) {
						err = r.removeServiceMonitor(ctx, client, sm.Namespace, sm.Name)
						if err != nil {
							return integreatlyv1alpha1.PhaseFailed, err
						}
						continue copyOut
					}
				}

				//Create a copy of service monitors in the observability namespace
				//Create the corresponding rolebindings at each of the service namespace
				key := sm.Namespace + `-` + sm.Name
				delete(monSermonMap, key) // Servicemonitor exists, remove it from the local map

				err := r.reconcileServiceMonitor(ctx, client, sm)
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, err
				}

				err = r.reconcileRoleBindingsForServiceMonitor(ctx, client, key)
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, err
				}
			}
		}
	}

	//Clean-up the stale service monitors and rolebindings if any
	if len(monSermonMap) > 0 {
	cleanUpOut:
		for _, sm := range monSermonMap {
			//Remove servicemonitor
			// on upgrade don't copy the one for redhat-rhoam-middleware-monitoring-operator as that namespace is removed
			// those service monitors were created by AMO to self monitor
			// the can be removed in the case of RHOAM
			for _, s := range sm.Spec.NamespaceSelector.MatchNames {
				if s == fmt.Sprintf("%smiddleware-monitoring-operator", r.Config.GetNamespacePrefix()) {
					err = r.removeServiceMonitor(ctx, client, sm.Namespace, sm.Name)
					if err != nil {
						return integreatlyv1alpha1.PhaseFailed, err
					}
					continue cleanUpOut
				}
			}

			err = r.removeServiceMonitor(ctx, client, sm.Namespace, sm.Name)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}
			//Remove rolebindings
			for _, namespace := range sm.Spec.NamespaceSelector.MatchNames {
				err := r.removeRoleandRoleBindingForServiceMonitor(ctx, client, namespace, serviceMonitorRoleRefName, serviceMonitorRoleBindingName)
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, err
				}
			}
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, err
}

func (r *Reconciler) reconcileBlackboxExporter(ctx context.Context, client k8sclient.Client, cfg *config.Observability) (integreatlyv1alpha1.StatusPhase, error) {
	// Create blackbox-exporter-service-account
	blackboxExporterServiceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      blackboxExporterPrefix,
			Namespace: cfg.GetNamespace(),
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, blackboxExporterServiceAccount, func() error {
		return nil
	})
	if err != nil {
		r.log.Error("Unable to create blackbox-exporter ServiceAccount", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create ClusterRole
	apiGroups := []string{""}
	resources := []string{"namespaces"}
	verbs := []string{"get"}
	err = r.reconcileClusterRole(ctx, client, blackboxExporterPrefix, apiGroups, resources, verbs)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create ClusterRoleBinding
	err = r.reconcileClusterRoleBinding(ctx, client, blackboxExporterPrefix, rbac.ServiceAccountKind, blackboxExporterPrefix, cfg.GetNamespace(), blackboxExporterPrefix, blackboxExporterAPIGroup)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getBlackboxExporterServiceAccountToken(ctx context.Context, client k8sclient.Client, namespace string) string {
	// Get the blackbox-exporter ServiceAccount
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      blackboxExporterPrefix,
			Namespace: namespace,
		},
	}
	if err := client.Get(ctx, k8sclient.ObjectKey{Name: sa.Name, Namespace: sa.Namespace}, sa); err != nil {
		r.log.Error("Unable to get blackbox-exporter ServiceAccount", err)
		return ""
	}

	// Get secret containing bearer token from the blackbox-exporter ServiceAccount
	secretName := ""
	for _, secret := range sa.Secrets {
		if res, err := regexp.MatchString(fmt.Sprintf("%s-token", blackboxExporterPrefix), secret.Name); err != nil {
			r.log.Error("Failed to get secret containing bearer token from blackbox-exporter ServiceAccount", err)
			return ""
		} else if res {
			secretName = secret.Name
		}
	}
	return secretName
}

func (r *Reconciler) removeRoleandRoleBindingForBlackbox(ctx context.Context, serverClient k8sclient.Client) (err error) {
	//Get the ClusterRoleBinding
	clusterRoleBinding := &rbac.ClusterRoleBinding{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: blackboxExporterPrefix}, clusterRoleBinding)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the ClusterRoleBinding if serverClient was able to get it
	if err == nil {
		err = serverClient.Delete(ctx, clusterRoleBinding)
		if err != nil && !k8serr.IsNotFound(err) {
			return err
		}
	}

	//Get the ClusterRole
	clusterRole := &rbac.ClusterRole{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: blackboxExporterPrefix}, clusterRole)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the ClusterRole if serverClient was able to get it
	if err == nil {
		err = serverClient.Delete(ctx, clusterRole)
		if err != nil && !k8serr.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (r *Reconciler) getServiceMonitors(ctx context.Context,
	serverClient k8sclient.Client,
	listOpts []k8sclient.ListOption) (serviceMonitorsMap map[string]*prometheus.ServiceMonitor, err error) {

	if len(listOpts) == 0 {
		return serviceMonitorsMap, fmt.Errorf("list options is empty")
	}
	serviceMonitors := &prometheus.ServiceMonitorList{}
	err = serverClient.List(ctx, serviceMonitors, listOpts...)
	if err != nil {
		return serviceMonitorsMap, err
	}
	serviceMonitorsMap = make(map[string]*prometheus.ServiceMonitor)
	for _, sm := range serviceMonitors.Items {
		serviceMonitorsMap[sm.Name] = sm
	}
	return serviceMonitorsMap, err
}

func (r *Reconciler) getMWMonitoredNamespaces(ctx context.Context,
	serverClient k8sclient.Client) (namespaces *v1.NamespaceList, err error) {
	ls, err := labels.Parse(labelSelector)
	if err != nil {
		return namespaces, err
	}
	opts := &k8sclient.ListOptions{
		LabelSelector: ls,
	}
	//Get the list of namespaces with the given label selector
	namespaces = &v1.NamespaceList{}
	err = serverClient.List(ctx, namespaces, opts)
	return namespaces, err
}

func (r *Reconciler) removeServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, namespace, name string) (err error) {
	sm := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	//Delete the servicemonitor
	err = serverClient.Delete(ctx, sm)
	if err != nil && k8serr.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *Reconciler) reconcileServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, serviceMonitor *prometheus.ServiceMonitor) (err error) {

	if serviceMonitor.Spec.NamespaceSelector.Any {
		r.log.Warningf("servicemonitor cannot be copied to namespace. Namespace selector has been set to any",
			l.Fields{"serviceMonitor": serviceMonitor.Name, "ns": r.Config.GetNamespace()})
		return nil
	}
	sm := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceMonitor.Namespace + `-` + serviceMonitor.Name,
			Namespace: r.Config.GetNamespace(),
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, sm, func() error {
		// Check if the servicemonitor has no  namespace selectors defined,
		// if not add the namespace
		sm.Spec = serviceMonitor.Spec
		if len(sm.Spec.NamespaceSelector.MatchNames) == 0 {
			sm.Spec.NamespaceSelector.MatchNames = []string{serviceMonitor.Namespace}
		}
		//Add all the original labels and append cloned servicemonitor label
		sm.Labels = serviceMonitor.Labels
		if len(sm.Labels) == 0 {
			sm.Labels = make(map[string]string)
		}
		sm.Labels[clonedServiceMonitorLabelKey] = clonedServiceMonitorLabelValue
		return nil
	})
	if err != nil {
		return err
	}
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result", l.Fields{"serviceMonitor": sm.Name, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileRoleBindingsForServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, serviceMonitorName string) (err error) {
	//Get the service monitor - that was created/updated
	sermon := &prometheus.ServiceMonitor{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: serviceMonitorName, Namespace: r.Config.GetNamespace()}, sermon)
	if err != nil {
		return err
	}
	//Create role binding for each of the namespace label selectors
	apiGroups := []string{""}
	resources := []string{
		"services",
		"endpoints",
		"pods",
	}
	verbs := []string{
		"get",
		"list",
		"watch",
	}
	for _, namespace := range sermon.Spec.NamespaceSelector.MatchNames {
		err := r.reconcileRole(ctx, serverClient, serviceMonitorRoleRefName, namespace, apiGroups, resources, verbs)
		if err != nil {
			return err
		}
		err = r.reconcileRoleBinding(ctx, serverClient, serviceMonitorRoleBindingName, namespace, rbac.ServiceAccountKind, clusterMonitoringPrometheusServiceAccount, clusterMonitoringNamespace, serviceMonitorRoleRefName, serviceMonitorRoleRefAPIGroup)
		if err != nil {
			return err
		}
	}
	return err
}

func (r *Reconciler) removeRoleandRoleBindingForServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, namespace, roleName, rbName string) (err error) {

	// Check if the namespace has service monitors
	// if so do not delete the rolebinding
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	serviceMonitorsMap, err := r.getServiceMonitors(ctx, serverClient, listOpts)
	if err != nil {
		return err
	}

	if len(serviceMonitorsMap) > 0 {
		return nil
	}

	//Get the role
	role := &rbac.Role{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: roleName, Namespace: namespace}, role)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the role
	err = serverClient.Delete(ctx, role)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Get the rolebinding
	rb := &rbac.RoleBinding{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: rbName, Namespace: namespace}, rb)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the rolebinding
	err = serverClient.Delete(ctx, rb)
	if err != nil && k8serr.IsNotFound(err) {
		return nil
	}
	return err
}

func GetDefaultNamespace(installationPrefix string) string {
	return installationPrefix + defaultInstallationNamespace
}

func getClonedServiceMonitorLabel() map[string]string {
	return map[string]string{
		clonedServiceMonitorLabelKey: clonedServiceMonitorLabelValue,
	}
}

func (r *Reconciler) reconcileRole(ctx context.Context, serverClient k8sclient.Client, name string, namespace string,
	apiGroups []string, resources []string, verbs []string) (err error) {

	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, role, func() error {
		role.Rules = []rbac.PolicyRule{
			{
				APIGroups: apiGroups,
				Resources: resources,
				Verbs:     verbs,
			},
		}
		return nil
	})
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result", l.Fields{"role": name, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileRoleBinding(ctx context.Context, serverClient k8sclient.Client, bindingName string,
	bindingNamespace string, subjectKind flowcontrolv1alpha1.SubjectKind, subjectName string, subjectNamespace string, roleName string, roleApiGroup string) (err error) {

	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bindingName,
			Namespace: bindingNamespace,
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, roleBinding, func() error {
		roleBinding.Subjects = []rbac.Subject{
			{
				Kind:      string(subjectKind),
				Name:      subjectName,
				Namespace: subjectNamespace,
			},
		}
		roleBinding.RoleRef = rbac.RoleRef{
			APIGroup: roleApiGroup,
			Kind:     "Role",
			Name:     roleName,
		}
		return nil
	})
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result", l.Fields{"roleBinding": bindingName, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileClusterRole(ctx context.Context, serverClient k8sclient.Client, name string,
	apiGroups []string, resources []string, verbs []string) (err error) {

	clusterRole := &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, clusterRole, func() error {
		clusterRole.Rules = []rbac.PolicyRule{
			{
				APIGroups: apiGroups,
				Resources: resources,
				Verbs:     verbs,
			},
		}
		return nil
	})
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result", l.Fields{"clusterRole": name, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileClusterRoleBinding(ctx context.Context, serverClient k8sclient.Client, bindingName string,
	subjectKind flowcontrolv1alpha1.SubjectKind, subjectName string, subjectNamespace string, roleName string, roleApiGroup string) (err error) {

	clusterRoleBinding := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, clusterRoleBinding, func() error {
		clusterRoleBinding.Subjects = []rbac.Subject{
			{
				Kind:      string(subjectKind),
				Name:      subjectName,
				Namespace: subjectNamespace,
			},
		}
		clusterRoleBinding.RoleRef = rbac.RoleRef{
			APIGroup: roleApiGroup,
			Kind:     "ClusterRole",
			Name:     roleName,
		}

		return nil
	})
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result", l.Fields{"roleBinding": bindingName, "result": opRes})
	}
	return err
}
