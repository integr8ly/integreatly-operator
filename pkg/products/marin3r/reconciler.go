package marin3r

import (
	"context"
	"fmt"
	prometheus "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	marin3r "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/version"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "marin3r"

	manifestPackage   = "integreatly-marin3r"
	serverSecretName  = "marin3r-server-cert-instance"
	caSecretName      = "marin3r-ca-cert-instance"
	secretDataCertKey = "tls.crt"
	secretDataKeyKey  = "tls.key"

	discoveryServiceName = "instance"

	externalRedisSecretName = "redis"

	statsdHost = "prom-statsd-exporter"
	statsdPort = "9125"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager config.ConfigReadWriter
	Config        *config.Marin3r
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	recorder      record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductMarin3r],
		string(integreatlyv1alpha1.VersionMarin3r),
		"",
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
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

	logger := logrus.NewEntry(logrus.StandardLogger())
	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  installation,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Start marin3r reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, client, productNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, client, operatorNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		if err := r.deleteDiscoveryService(ctx, client); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete discovery service: %v", err)
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, client, operatorNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CloudResourceSubscriptionName), err)
		return phase, err
	}

	logrus.Infof("about to start reconciling the discovery service")
	phase, err = r.reconcileDiscoveryService(ctx, client, productNamespace, installation.Spec.NamespacePrefix)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile DiscoveryService cr"), err)
		return phase, err
	}

	phase, err = r.reconcilePromStatsdExporter(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile Prometheus StatsD exporter cr"), err)
		return phase, err
	}
	statsdConfig := &StatsdConfig{
		Host: statsdHost,
		Port: statsdPort,
	}

	phase, err = r.reconcilePromStatsdExporterService(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile Prometheus StatsD exporter service"), err)
		return phase, err
	}

	phase, err = NewRateLimitServiceReconciler(productNamespace, externalRedisSecretName, statsdConfig).ReconcileRateLimitService(ctx, client)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
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

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileDiscoveryService(ctx context.Context, client k8sclient.Client, productNamespace string, namespacePrefix string) (integreatlyv1alpha1.StatusPhase, error) {
	enabledNamespaces := []string{
		namespacePrefix + "3scale",
	}

	discoveryService := &marin3r.DiscoveryService{
		ObjectMeta: metav1.ObjectMeta{
			Name: discoveryServiceName,
		},
		Spec: marin3r.DiscoveryServiceSpec{
			DiscoveryServiceNamespace: productNamespace,
			EnabledNamespaces:         enabledNamespaces,
			Image:                     "quay.io/3scale/marin3r:v0.5.1",
		},
	}

	err := client.Create(ctx, discoveryService)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating resource: %w", err)
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
	logrus.Infof("Start reconcileSubscription for marin3r")
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
	)
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	if r.installation.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}
	//todo add backup for redis once it's added to the reconciler
	return backup.NewNoopBackupExecutor()
}

func (r *Reconciler) reconcilePromStatsdExporter(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Start reconcilePromStatsdExporter for marin3r")

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prom-statsd-exporter",
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if deployment.Labels == nil {
			deployment.Labels = map[string]string{}
		}
		deployment.Labels["app"] = "prom-statsd-exporter"

		var replicas int32 = 1
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "prom-statsd-exporter",
			},
		}
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "prom-statsd-exporter",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "prom-statsd-exporter",
						Image: "prom/statsd-exporter:v0.18.0",
						Ports: []corev1.ContainerPort{
							{
								Name:          "prom-statsd",
								ContainerPort: 9125,
							},
							{
								Name:          "metrics",
								ContainerPort: 9102,
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
	logrus.Infof("Start reconcilePromStatsdExporterService for marin3r")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prom-statsd-exporter",
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}

		service.Labels["app"] = "prom-statsd-exporter"
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "prom-statsd",
				Protocol:   corev1.ProtocolTCP,
				Port:       9125,
				TargetPort: intstr.FromInt(9125),
			},
			{
				Name:       "metrics",
				Protocol:   corev1.ProtocolTCP,
				Port:       9102,
				TargetPort: intstr.FromInt(9102),
			},
		}
		service.Spec.Selector = map[string]string{
			"app": "prom-statsd-exporter",
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileServiceMonitor(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Start reconcileServiceMonitor for marin3r")

	serviceMonitor := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prom-statsd-exporter",
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
					"app": "prom-statsd-exporter ",
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
