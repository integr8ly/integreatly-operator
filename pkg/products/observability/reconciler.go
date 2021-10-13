package observability

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoringcommon"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/version"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	observability "github.com/redhat-developer/observability-operator/v3/api/v1"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "observability"

	configMapNoInit              = "observability-operator-no-init"
	observabilityName            = "observability-stack"
	defaultProbeModule           = "http_2xx"
	OpenshiftMonitoringNamespace = "openshift-monitoring"
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

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductObservability],
		string(integreatlyv1alpha1.VersionObservability),
		string(integreatlyv1alpha1.OperatorVersionObservability),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {

	ns := GetDefaultNamespace(installation.Spec.NamespacePrefix)
	config, err := configManager.ReadObservability()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve observability config: %w", err)
	}

	config.SetNamespacePrefix(installation.Spec.NamespacePrefix)

	if config.GetNamespace() == "" {
		config.SetNamespace(ns)
		err := configManager.WriteConfig(config)
		if err != nil {
			return nil, err
		}
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
		err := configManager.WriteConfig(config)
		if err != nil {
			return nil, err
		}
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
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

	phase, err = r.reconcileConfigMap(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s configmap which is required to disable observability operator initilisting it's own cr", configMapNoInit), err)
		return phase, err
	}

	monitoringConfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.log.Info("check AMO is uninstalled before progressing with Observability Installation")
	amo := &v1alpha1.ApplicationMonitoringList{}

	err = client.List(ctx, amo, &k8sclient.ListOptions{
		Namespace: monitoringConfg.GetOperatorNamespace(),
	})
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to check AMO is uninstalled"), err)
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

	phase, err = r.newAlertsReconciler(r.log, r.installation.Spec.Type).ReconcileAlerts(ctx, client)
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

	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func CreatePrometheusProbe(ctx context.Context, client k8sclient.Client, inst *integreatlyv1alpha1.RHMI, cfg *config.Observability, name string, module string, targets prometheus.ProbeTargetStaticConfig) (integreatlyv1alpha1.StatusPhase, error) {
	if cfg.GetNamespace() == "" {
		// Retry later
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if len(targets.Targets) == 0 {
		// Retry later if the URL(s) is not yet known
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// The default policy is to require a 2xx http return code
	if module == "" {
		module = defaultProbeModule
	}

	// Prepare the probe
	probe := &prometheus.Probe{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cfg.GetNamespace(),
		},
	}
	owner.AddIntegreatlyOwnerAnnotations(probe, inst)
	_, err := controllerutil.CreateOrUpdate(ctx, client, probe, func() error {
		probe.Labels = map[string]string{
			cfg.GetLabelSelectorKey(): cfg.GetLabelSelector(),
		}
		probe.Spec = prometheus.ProbeSpec{
			JobName: "blackbox",
			ProberSpec: prometheus.ProberSpec{
				URL:    "127.0.0.1:9115",
				Scheme: "http",
				Path:   "/probe",
			},
			Module: module,
			Targets: prometheus.ProbeTargets{
				StaticConfig: &targets,
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

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

		if &oo.Spec == nil {
			oo.Spec = observability.ObservabilitySpec{}
		}

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
		}

		oo.Spec.Retention = r.Config.GetPrometheusRetention()

		if oo.Spec.SelfContained == nil {
			oo.Spec.SelfContained = &observability.SelfContained{}
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

	oo := &observability.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: targetNamespace,
		},
	}

	// Get the observability CR; return if not found
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oo.Name, Namespace: oo.Namespace}, oo)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Mark the observability CR for deletion
	err = serverClient.Delete(ctx, oo)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
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

	grafanaDB := &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboard,
			Namespace: r.Config.GetNamespace(),
		},
	}
	specJSON, _, err := monitoringcommon.GetSpecDetailsForDashboard(dashboard, r.installation)
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

func GetDefaultNamespace(installationPrefix string) string {
	return installationPrefix + defaultInstallationNamespace
}
