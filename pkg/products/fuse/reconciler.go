package fuse

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/client"
	v1 "k8s.io/api/core/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"github.com/sirupsen/logrus"

	syndesisv1beta1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1beta1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "fuse"
	defaultFusePullSecret        = "syndesis-pull-secret"
	developersGroupName          = "rhmi-developers"
	clusterViewRoleName          = "view"
	manifestPackage              = "integreatly-fuse-online"
	syndesisPrometheusPVC        = "10Gi"
	syndesisPrometheus           = "syndesis-prometheus"
)

// Reconciler reconciles everything needed to install Syndesis/Fuse. The resources that it works
// with are considered secondary resources in the context of the installation controller.
type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	Config        *config.Fuse
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	recorder      record.EventRecorder
}

// NewReconciler instantiates and returns a reference to a new Reconciler.
func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadFuse()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve fuse config: %w", err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}
	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("fuse config is not valid: %w", err)
	}
	config.SetBlackboxTargetPath("/oauth/healthz")

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

// GetPreflightObject returns an object that will be checked in the preflight checks in the main
// Installation controller to ensure there isn't a conflicting Syndesis already installed.
func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis-server",
			Namespace: ns,
		},
	}
}

// Reconcile reads that state of the cluster for fuse and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}

	err = resources.CopyPullSecretToNameSpace(ctx, installation.GetPullSecretSpec(), r.Config.GetNamespace(), defaultFusePullSecret, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s pull secret", defaultFusePullSecret), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.reconcileViewFusePerms(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile view fuse permissions", err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	preUpgradeBackups := preUpgradeBackupExecutor(installation)
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.FuseSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, preUpgradeBackups, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.FuseSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileCloudResources(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile cloud resources", err)
		return phase, err
	}

	phase, err = r.reconcileCustomResource(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile custom resource", err)
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.reconcileKubeStateMetricsEndpointAvailableAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile endpoint available alerts", err)
		return phase, err
	}

	phase, err = r.reconcileKubeStateMetricsOperatorEndpointAvailableAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile operator endpoint available alerts", err)
		return phase, err
	}
	phase, err = r.reconcileKubeStateMetricsFuseAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile fuse ksm alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	logrus.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["MonitoringKey"] = r.Config.GetLabelSelector()
	r.extraParams["Namespace"] = r.Config.GetNamespace()

	templateHelper := monitoring.NewTemplateHelper(r.extraParams)
	resource, err := templateHelper.CreateResource(resourceName)

	if err != nil {
		return nil, fmt.Errorf("createResource failed: %w", err)
	}

	err = serverClient.Create(ctx, resource)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return nil, fmt.Errorf("error creating resource: %w", err)
		}
	}

	return resource, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, template, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Ensures all users in rhmi-developers group have view Fuse permissions
func (r *Reconciler) reconcileViewFusePerms(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Infof("Reconciling view Fuse permissions for %s group on %s namespace", developersGroupName, r.Config.GetNamespace())
	viewFuseRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      developersGroupName + "-fuse-view",
			Namespace: r.Config.GetNamespace(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterViewRoleName,
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, viewFuseRoleBinding, func() error {
		viewFuseRoleBinding.Subjects = []rbacv1.Subject{rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Name:     developersGroupName,
			Kind:     "Group",
		}}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	r.logger.Infof("The %s subjects were: %s", viewFuseRoleBinding.Name, or)
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileCloudResources(ctx context.Context, rhmi *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling cloud resources for Fuse")

	pgName := fmt.Sprintf("%s%s", constants.FusePostgresPrefix, rhmi.Name)
	ns := rhmi.Namespace
	postgres, err := croResources.ReconcilePostgres(ctx, client, defaultInstallationNamespace, rhmi.Spec.Type, croResources.TierProduction, pgName, ns, pgName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, rhmi)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres instance for fuse: %w", err)
	}

	// create prometheus failed rule
	_, err = resources.CreatePostgresResourceStatusPhaseFailedAlert(ctx, client, rhmi, postgres)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres failure alert: %w", err)
	}

	// postgres provisioning is still in progress
	if postgres.Status.Phase != croTypes.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingCloudResources, nil
	}

	// create prometheus pending rule
	_, err = resources.CreatePostgresResourceStatusPhasePendingAlert(ctx, client, rhmi, postgres)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres pending alert: %w", err)
	}

	// create the prometheus availability rule
	if _, err = resources.CreatePostgresAvailabilityAlert(ctx, client, rhmi, postgres); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus alert for rhsso: %w", err)
	}

	// create the prometheus connectivity rule
	if _, err = resources.CreatePostgresConnectivityAlert(ctx, client, rhmi, postgres); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus connectivity alert for rhsso : %s", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// reconcileCustomResource ensures that the fuse custom resource exists
func (r *Reconciler) reconcileCustomResource(ctx context.Context, rhmi *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling fuse custom resource")

	pgName := fmt.Sprintf("%s%s", constants.FusePostgresPrefix, rhmi.Name)
	// get the credential secret
	postgresSec := &v1.Secret{}
	if err := client.Get(ctx, k8sclient.ObjectKey{Name: pgName, Namespace: rhmi.Namespace}, postgresSec); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get postgres credential secret for fuse: %w", err)
	}

	// create the syndesis external database secret
	synExternalDatabaseSec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis-global-config",
			Namespace: r.Config.GetNamespace(),
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, synExternalDatabaseSec, func() error {
		if synExternalDatabaseSec.Data == nil {
			synExternalDatabaseSec.Data = map[string][]byte{}
		}
		synExternalDatabaseSec.Data["POSTGRESQL_PASSWORD"] = postgresSec.Data["password"]
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile fuse external database secret: %w", err)
	}

	//Reconcile PVC for syndesis-prometheus
	pvccr := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      syndesisPrometheus,
			Namespace: r.Config.GetNamespace(),
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, client, pvccr, func() error {
		if len(pvccr.Spec.Resources.Requests) == 0 {
			pvccr.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
			pvccr.Spec.Resources.Requests = make(v1.ResourceList)
			pvccr.Spec.Resources.Requests[v1.ResourceStorage] = resource.MustParse(syndesisPrometheusPVC)
		} else {
			pvccr.Spec.Resources.Requests[v1.ResourceStorage] = resource.MustParse(syndesisPrometheusPVC)
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update syndesis-promtheus PVC custom resource: %w", err)
	}
	if opRes != controllerutil.OperationResultNone {
		r.logger.Infof("operation result of creating/updating syndesis-prometheus PVC CR was %v", opRes)
	}

	cr := &syndesisv1beta1.Syndesis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "integreatly",
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		threescaleHost := ""
		threescaleConfig, err := r.ConfigManager.ReadThreeScale()
		// ignore errors in case 3Scale is not installed yet
		if err == nil {
			threescaleHost = threescaleConfig.GetHost()
		}
		cr.Spec = syndesisv1beta1.SyndesisSpec{
			Components: syndesisv1beta1.ComponentsSpec{
				Database: syndesisv1beta1.DatabaseConfiguration{
					User:          string(postgresSec.Data["username"]),
					Name:          string(postgresSec.Data["database"]),
					ExternalDbURL: fmt.Sprintf("postgresql://%s:%s", string(postgresSec.Data["host"]), string(postgresSec.Data["port"])),
				},
				Server: syndesisv1beta1.ServerConfiguration{
					Features: syndesisv1beta1.ServerFeatures{
						ManagementUrlFor3scale: threescaleHost,
						IntegrationLimit:       0,
					},
				},
			},
			Addons: syndesisv1beta1.AddonsSpec{
				Jaeger: syndesisv1beta1.JaegerConfiguration{
					// enabled being false still creates some resources
					Enabled:      false,
					OperatorOnly: true,
					ClientOnly:   true,
				},
				Todo: syndesisv1beta1.AddonSpec{
					Enabled: false,
				},
			},
		}
		owner.AddIntegreatlyOwnerAnnotations(cr, rhmi)
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update a Syndesis(Fuse) custom resource: %w", err)
	}

	if cr.Status.Phase == syndesisv1beta1.SyndesisPhaseStartupFailed {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to install fuse custom resource: %s", cr.Status.Reason)
	}

	if cr.Status.Phase != syndesisv1beta1.SyndesisPhaseInstalled {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis",
			Namespace: r.Config.GetNamespace(),
		},
	}

	if err := client.Get(ctx, k8sclient.ObjectKey{Name: route.Name, Namespace: route.Namespace}, route); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not read syndesis route for fuse: %w", err)
	}

	var url string
	if route.Spec.TLS != nil {
		url = fmt.Sprintf("https://" + route.Spec.Host)
	} else {
		url = fmt.Sprintf("http://" + route.Spec.Host)
	}
	if r.Config.GetHost() != url {
		r.Config.SetHost(url)
		r.ConfigManager.WriteConfig(r.Config)
	}

	// if there are no errors, the phase is complete
	return integreatlyv1alpha1.PhaseCompleted, nil
}
func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-syndesis", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost() + r.Config.GetBlackboxTargetPath(),
		Service: "syndesis-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating syndesis blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
func preUpgradeBackupExecutor(rhmi *integreatlyv1alpha1.RHMI) backup.BackupExecutor {
	pgName := fmt.Sprintf("%s%s", constants.FusePostgresPrefix, rhmi.Name)
	if rhmi.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}

	return backup.NewAWSBackupExecutor(
		rhmi.Namespace,
		pgName,
		backup.PostgresSnapshotType,
	)
}
