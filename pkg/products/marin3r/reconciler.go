package marin3r

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/products/grafana"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/pkg/errors"

	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	marin3roperator "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
	"github.com/integr8ly/integreatly-operator/version"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "marin3r"
	discoveryServiceName         = "instance"
	externalRedisSecretName      = "redis"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager   config.ConfigReadWriter
	Config          *config.Marin3r
	RateLimitConfig marin3rconfig.RateLimitConfig
	AlertsConfig    map[string]*marin3rconfig.AlertConfig
	installation    *integreatlyv1alpha1.RHMI
	mpm             marketplace.MarketplaceInterface
	log             l.Logger
	recorder        record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductMarin3r],
		string(integreatlyv1alpha1.VersionMarin3r),
		string(integreatlyv1alpha1.OperatorVersionMarin3r),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for marin3r")
	}

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	productConfig, err := configManager.ReadMarin3r()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve threescale config: %w", err)
	}

	productConfig.SetNamespace(ns)
	if installation.Spec.OperatorsInProductNamespace {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace())
	} else {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace() + "-operator")
	}

	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, fmt.Errorf("error writing marin3r config : %w", err)
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

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start marin3r reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	threescaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "could not read 3scale config from marin3r reconciler")
	}

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		enabledNamespaces := []string{threescaleConfig.GetNamespace()}
		phase, err := ratelimit.DeleteEnvoyConfigsInNamespaces(ctx, client, enabledNamespaces...)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		if err := r.deleteDiscoveryService(ctx, client); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete discovery service: %v", err)
		}

		phase, err = resources.RemoveNamespace(ctx, installation, client, productNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, client, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}

	r.RateLimitConfig = productConfig.GetRateLimitConfig()

	alertsConfig, err := marin3rconfig.GetAlertConfig(ctx, client, r.installation.Namespace)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to obtain rate limit alerts config", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	r.AlertsConfig = alertsConfig

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

	phase, err = r.reconcileSubscription(ctx, client, operatorNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CloudResourceSubscriptionName), err)
		return phase, err
	}

	r.log.Info("about to start reconciling the discovery service")
	phase, err = r.reconcileDiscoveryService(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile DiscoveryService cr", err)
		return phase, err
	}

	phase, err = r.ReconcileCsvDeploymentsPriority(
		ctx,
		client,
		fmt.Sprintf("marin3r.v%s", integreatlyv1alpha1.OperatorVersionMarin3r),
		r.Config.GetOperatorNamespace(),
		r.installation.Spec.PriorityClassName,
	)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile marin3r csv deployments priority class name", err)
		return phase, err
	}

	// Wait for RHSSO postgres to be completed
	phase, err = resources.WaitForRHSSOPostgresToBeComplete(client, installation.Name, r.ConfigManager.GetOperatorNamespace())
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Waiting for RHSSO postgres to be completed", err)
		return phase, err
	}

	phase, err = r.reconcileRedis(ctx, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		return phase, nil
	}

	phase, err = NewRateLimitServiceReconciler(r.RateLimitConfig, installation, productNamespace, externalRedisSecretName, resources.NewPodExecutor(r.log), r.ConfigManager).
		ReconcileRateLimitService(ctx, client, productConfig)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rate limit service", err)
		return phase, err
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	phase, err = r.reconcileServiceMonitor(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile Prometheus service monitor", err)
		return phase, err
	}

	alertsReconciler := r.newAlertReconciler(r.log, r.installation.Spec.Type, config.GetOboNamespace(r.installation.Namespace))
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile Marin3r alerts", err)
		return phase, err
	}

	// Reconcile API usage alerts
	phase, err = r.reconcileAlerts(ctx, client, installation, config.GetOboNamespace(r.installation.Namespace))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	rejectedRequestsAlertReconciler, err := r.newRejectedRequestsAlertsReconciler(r.log, r.installation.Spec.Type, config.GetOboNamespace(r.installation.Namespace))
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to instantiate rejected requests alert reconciler", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase, err := rejectedRequestsAlertReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rejected requests alert", err)
		return phase, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Installation successful")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAlerts(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	if !resources.IsInProw(installation) {
		grafanaConsoleURL, err := grafana.GetGrafanaConsoleURL(ctx, client, installation)
		if err != nil {
			if installStage, ok := installation.Status.Stages[integreatlyv1alpha1.InstallStage]; ok {
				if installStage.Products != nil {
					grafanaProduct, grafanaProductExists := installStage.Products[integreatlyv1alpha1.ProductGrafana]
					// Ignore the Forbidden and NotFound errors if Grafana is not installed yet
					if !grafanaProductExists ||
						(grafanaProduct.Phase != integreatlyv1alpha1.PhaseCompleted &&
							(k8serr.IsForbidden(err) || k8serr.IsNotFound(err))) {

						r.log.Info("Failed to get Grafana console URL. Awaiting completion of Grafana installation")
						return integreatlyv1alpha1.PhaseInProgress, nil
					}
				}
			}
			r.log.Error("failed to get Grafana console URL", err)
			return integreatlyv1alpha1.PhaseFailed, err
		}

		grafanaDashboardURL := fmt.Sprintf("%s/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting", grafanaConsoleURL)
		alertReconciler, err := r.newAlertsReconciler(grafanaDashboardURL, namespace)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		phase, err := alertReconciler.ReconcileAlerts(ctx, client)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
			return phase, err
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileRedis(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Creating backend redis instance in marin3r reconcile")

	ns := r.installation.Namespace

	redisName := fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, r.installation.Name)
	rateLimitRedis, err := croUtil.ReconcileRedis(ctx, client, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, redisName, ns, redisName, ns, "", false, false, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile backend redis request: %w", err)
	}

	// wait for the redis cr to reconcile
	if rateLimitRedis.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret created by the cloud resources operator
	// containing system redis connection details
	systemCredSec := &corev1.Secret{}
	err = client.Get(ctx, k8sclient.ObjectKey{Name: rateLimitRedis.Status.SecretRef.Name, Namespace: rateLimitRedis.Status.SecretRef.Namespace}, systemCredSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get system redis credential secret: %w", err)
	}

	// create system redis external connection secret needed for the 3scale apimanager
	redisSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalRedisSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, redisSecret, func() error {
		uri := systemCredSec.Data["uri"]
		port := systemCredSec.Data["port"]

		conn := fmt.Sprintf("%s:%s", uri, port)
		redisSecret.Data["URL"] = []byte(conn)

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed create or update redis secret: %w", err)
	}

	phase, err := resources.ReconcileRedisAlerts(ctx, client, r.installation, rateLimitRedis, r.log)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile redis alerts: %w", err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	// create Redis Cpu Usage High alert
	err = resources.CreateRedisCpuUsageAlerts(ctx, client, r.installation, rateLimitRedis, r.log)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create rate limit redis prometheus Cpu usage high alerts for threescale: %s", err)
	}

	return phase, nil
}

func (r *Reconciler) reconcileDiscoveryService(ctx context.Context, client k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	threescaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "could not read 3scale config from marin3r reconciler")
	}

	discoveryService := &marin3roperator.DiscoveryService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      discoveryServiceName,
			Namespace: threescaleConfig.GetNamespace(),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, discoveryService, func() error {
		image := fmt.Sprintf("quay.io/3scale/marin3r:v%s", integreatlyv1alpha1.VersionMarin3r)
		discoveryService.Spec.Image = &image
		discoveryService.Spec.PodPriorityClass = &r.installation.Spec.PriorityClassName
		return nil
	})
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reconciling resource: %w", err)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) deleteDiscoveryService(ctx context.Context, client k8sclient.Client) error {
	threescaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return errors.Wrap(err, "could not read 3scale config from marin3r reconciler")
	}

	discoveryService := &marin3roperator.DiscoveryService{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      discoveryServiceName,
		Namespace: threescaleConfig.GetNamespace(),
	}, discoveryService); err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		}

		return err
	}

	return client.Delete(ctx, discoveryService)
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		SubscriptionName: constants.Marin3rSubscriptionName,
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
		[]string{},
		r.preUpgradeBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	if r.installation.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}

	return backup.NewAWSBackupExecutor(
		r.installation.Namespace,
		fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, r.installation.Name),
		backup.RedisSnapshotType,
	)
}

func (r *Reconciler) reconcileServiceMonitor(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start reconcileServiceMonitor for marin3r")

	serviceMonitor := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quota.RateLimitName,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, serviceMonitor, func() error {
		serviceMonitor.Labels = map[string]string{
			"monitoring-key": "middleware",
		}
		serviceMonitor.Spec = prometheus.ServiceMonitorSpec{
			Endpoints: []prometheus.Endpoint{
				{
					BearerTokenSecret: corev1.SecretKeySelector{
						Key: "",
					},
					Port: "http",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": quota.RateLimitName,
				},
			},
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
