package marin3r

import (
	"context"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	corev1 "k8s.io/api/core/v1"
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
	manifestPackage              = "integreatly-marin3r"
	serverSecretName             = "marin3r-server-cert-instance"
	caSecretName                 = "marin3r-ca-cert-instance"
	secretDataCertKey            = "tls.crt"
	secretDataKeyKey             = "tls.key"

	discoveryServiceName = "instance"

	externalRedisSecretName = "redis"
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

	phase, err = r.reconcileRedis(ctx, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		return phase, nil
	}

	phase, err = NewRateLimitServiceReconciler(productNamespace, externalRedisSecretName).
		ReconcileRateLimitService(ctx, client)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rate limit service", err)
		return phase, err
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		return phase, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileRedis(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Creating backend redis instance in marine3r reconcile")

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

	phase, err := resources.ReconcileRedisAlerts(ctx, client, r.installation, rateLimitRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile redis alerts: %w", err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	// create Redis Cpu Usage High alert
	err = resources.CreateRedisCpuUsageAlerts(ctx, client, r.installation, rateLimitRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create rate limit redis prometheus Cpu usage high alerts for threescale: %s", err)
	}

	return phase, nil
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

	return backup.NewAWSBackupExecutor(
		r.installation.Namespace,
		externalRedisSecretName,
		backup.RedisSnapshotType,
	)
}
