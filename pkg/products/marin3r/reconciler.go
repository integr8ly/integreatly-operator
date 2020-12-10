package marin3r

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/pkg/errors"

	"strconv"

	prometheus "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/products/grafana"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	marin3r "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "marin3r"
	manifestPackage              = "integreatly-marin3r"
	statsdHost                   = "prom-statsd-exporter"
	statsdPort                   = 9125
	metricsPort                  = 9102
	discoveryServiceName         = "instance"
	externalRedisSecretName      = "redis"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager   config.ConfigReadWriter
	Config          *config.Marin3r
	RateLimitConfig *marin3rconfig.RateLimitConfig
	AlertsConfig    map[string]*marin3rconfig.AlertConfig
	installation    *integreatlyv1alpha1.RHMI
	mpm             marketplace.MarketplaceInterface
	log             l.Logger
	recorder        record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductMarin3r],
		string(integreatlyv1alpha1.VersionMarin3r),
		string(integreatlyv1alpha1.OperatorVersionMarin3r),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger) (*Reconciler, error) {
	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	config, err := configManager.ReadMarin3r()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve threescale config: %w", err)
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(ns)
		configManager.WriteConfig(config)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start marin3r reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		threescaleConfig, err := r.ConfigManager.ReadThreeScale()
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "could not read 3scale config from marin3r reconciler")
		}

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
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	rateLimitConfig, err := marin3rconfig.GetRateLimitConfig(ctx, client, r.installation.Namespace)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to obtain rate limit config", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	r.RateLimitConfig = rateLimitConfig

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

	phase, err = r.reconcileSubscription(ctx, client, operatorNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CloudResourceSubscriptionName), err)
		return phase, err
	}

	r.log.Info("about to start reconciling the discovery service")
	phase, err = r.reconcileDiscoveryService(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile DiscoveryService cr"), err)
		return phase, err
	}

	phase, err = r.reconcileRedis(ctx, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		return phase, nil
	}

	phase, err = r.reconcilePromStatsdExporter(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile Prometheus StatsD exporter cr"), err)
		return phase, err
	}
	statsdConfig := StatsdConfig{
		Host: statsdHost,
		Port: strconv.Itoa(statsdPort),
	}

	phase, err = r.reconcilePromStatsdExporterService(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile Prometheus StatsD exporter service"), err)
		return phase, err
	}

	phase, err = NewRateLimitServiceReconciler(r.RateLimitConfig, installation, productNamespace, externalRedisSecretName).
		WithStatsdConfig(statsdConfig).
		ReconcileRateLimitService(ctx, client)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rate limit service", err)
		return phase, err
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		return phase, nil
	}

	phase, err = r.reconcileServiceMonitor(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile Prometheus service monitor"), err)
		return phase, err
	}

	alertsReconciler := r.newAlertReconciler(r.log)
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile Marin3r alerts", err)
		return phase, err
	}

	// Reconcile API usage alerts
	phase, err = r.reconcileAlerts(ctx, client, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	rejectedRequestsAlertReconciler, err := r.newRejectedRequestsAlertsReconciler(r.log)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to instantiate rejected requests alert reconciler", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase, err := rejectedRequestsAlertReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rejected requests alert", err)
		return phase, err
	}

	if phase, err := r.newSoftLimitAlertsReconciler(r.log).ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile soft limit alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Installation successful")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAlerts(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {

	granafaConsoleURL, err := grafana.GetGrafanaConsoleURL(ctx, client, installation)
	if err != nil {
		if productsStage, ok := installation.Status.Stages[v1alpha1.ProductsStage]; ok {
			if productsStage.Products != nil {
				grafanaProduct, grafanaProductExists := productsStage.Products[v1alpha1.ProductGrafana]
				// Ignore the Forbidden and NotFound errors if Grafana is not installed yet
				if !grafanaProductExists ||
					(grafanaProduct.Status != v1alpha1.PhaseCompleted &&
						(k8serr.IsForbidden(err) || k8serr.IsNotFound(err))) {

					r.log.Info("Failed to get Grafana console URL. Awaiting completion of Grafana installation")
					return integreatlyv1alpha1.PhaseInProgress, nil
				}
			}
		}
		r.log.Error("failed to get Grafana console URL", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	grafanaDashboardURL := fmt.Sprintf("%s/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting", granafaConsoleURL)
	alertReconciler, err := r.newAlertsReconciler(grafanaDashboardURL)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err := alertReconciler.ReconcileAlerts(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileRedis(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Creating backend redis instance in marine3r reconcile")

	ns := r.installation.Namespace

	redisName := fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, r.installation.Name)
	rateLimitRedis, err := croUtil.ReconcileRedis(ctx, client, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, redisName, ns, redisName, ns, func(cr metav1.Object) error {
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

	enabledNamespaces := []string{threescaleConfig.GetNamespace()}
	discoveryService := &marin3r.DiscoveryService{
		ObjectMeta: metav1.ObjectMeta{
			Name: discoveryServiceName,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, discoveryService, func() error {
		discoveryService.Spec.DiscoveryServiceNamespace = productNamespace
		discoveryService.Spec.EnabledNamespaces = enabledNamespaces
		discoveryService.Spec.Image = fmt.Sprintf("quay.io/3scale/marin3r:v%s", integreatlyv1alpha1.VersionMarin3r)
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
	discoveryService := &marin3r.DiscoveryService{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name: discoveryServiceName,
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
		Pkg:       constants.Marin3rSubscriptionName,
		Namespace: operatorNamespace,
		Channel:   marketplace.IntegreatlyChannel,
	}
	catalogSourceReconciler := marketplace.NewConfigMapCatalogSourceReconciler(
		manifestPackage,
		serverClient,
		operatorNamespace,
		marketplace.CatalogSourceName,
	)
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

func (r *Reconciler) reconcilePromStatsdExporter(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start reconcilePromStatsdExporter for marin3r")

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statsdHost,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if deployment.Labels == nil {
			deployment.Labels = map[string]string{}
		}
		deployment.Labels["app"] = statsdHost

		var replicas int32 = 1
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": statsdHost,
			},
		}
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": statsdHost,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  statsdHost,
						Image: "quay.io/integreatly/statsd-exporter:v0.18.0",
						Ports: []corev1.ContainerPort{
							{
								Name:          "prom-statsd",
								ContainerPort: statsdPort,
							},
							{
								Name:          "metrics",
								ContainerPort: metricsPort,
							},
						},
					},
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

func (r *Reconciler) reconcilePromStatsdExporterService(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start reconcilePromStatsdExporterService for marin3r")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statsdHost,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}

		service.Labels["app"] = statsdHost
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "prom-statsd",
				Protocol:   corev1.ProtocolTCP,
				Port:       statsdPort,
				TargetPort: intstr.FromInt(statsdPort),
			},
			{
				Name:       "metrics",
				Protocol:   corev1.ProtocolTCP,
				Port:       metricsPort,
				TargetPort: intstr.FromInt(metricsPort),
			},
		}
		service.Spec.Selector = map[string]string{
			"app": statsdHost,
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileServiceMonitor(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start reconcileServiceMonitor for marin3r")

	serviceMonitor := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statsdHost,
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
					Port: "metrics",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": statsdHost,
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
