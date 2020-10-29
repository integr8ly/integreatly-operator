package ups

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "ups"
	defaultUpsName               = "ups"
	defaultRoutename             = defaultUpsName + "-unifiedpush-proxy"
	manifestPackage              = "integreatly-unifiedpush"
	tier                         = "production"
)

type Reconciler struct {
	Config        *config.Ups
	ConfigManager config.ConfigReadWriter
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadUps()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve ups config: %w", err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
		configManager.WriteConfig(config)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
		configManager.WriteConfig(config)
	}

	config.SetBlackboxTargetPath("/rest/auth/config/")

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

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unifiedpush-operator",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductUps],
		string(integreatlyv1alpha1.VersionUps),
		string(integreatlyv1alpha1.OperatorVersionUPS),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", defaultUpsName)

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, productNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.UPSSubscriptionName), err)

		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.reconcileHost(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile host", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler().ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	logrus.Infof("%s is successfully reconciled", defaultUpsName)
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling external postgres")
	ns := installation.Namespace

	// setup postgres custom resource
	// this will be used by the cloud resources operator to provision a postgres instance
	postgresName := fmt.Sprintf("%s%s", constants.UPSPostgresPrefix, installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, client, defaultInstallationNamespace, installation.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres request: %w", err)
	}

	// reconcile postgres alerts
	phase, err := resources.ReconcilePostgresAlerts(ctx, client, installation, postgres)
	productName := postgres.Labels["productName"]
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres alerts for %s: %w", productName, err)
	}

	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	// get the secret created by the cloud resources operator
	// containing postgres connection details
	connSec := &corev1.Secret{}
	err = client.Get(ctx, k8sclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, connSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get postgres credential secret: %w", err)
	}

	postgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      postgres.Status.SecretRef.Name,
			Namespace: r.Config.GetNamespace(),
		},
	}

	controllerutil.CreateOrUpdate(ctx, client, postgresSecret, func() error {
		postgresSecret.StringData = map[string]string{
			"POSTGRES_DATABASE":  string(connSec.Data["database"]),
			"POSTGRES_HOST":      string(connSec.Data["host"]),
			"POSTGRES_PORT":      string(connSec.Data["port"]),
			"POSTGRES_USERNAME":  string(connSec.Data["username"]),
			"POSTGRES_PASSWORD":  string(connSec.Data["password"]),
			"POSTGRES_SUPERUSER": "false",
			"POSTGRES_VERSION":   "10",
		}
		return nil
	})

	// Reconcile ups custom resource
	logrus.Info("Reconciling unified push server cr")
	cr := &upsv1alpha1.UnifiedPushServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultUpsName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: upsv1alpha1.UnifiedPushServerSpec{
			ExternalDB:     true,
			DatabaseSecret: postgres.Status.SecretRef.Name,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.ObjectMeta.Name = defaultUpsName
		cr.ObjectMeta.Namespace = r.Config.GetNamespace()
		cr.Spec.ExternalDB = true
		cr.Spec.DatabaseSecret = postgres.Status.SecretRef.Name
		owner.AddIntegreatlyOwnerAnnotations(cr, installation)
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/maintain unifiedpush server custom resource during reconcile: %w", err)
	}

	// Wait till the ups cr status is complete
	if cr.Status.Phase != upsv1alpha1.PhaseReconciling {
		logrus.Info("Waiting for unified push server cr phase to complete")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	logrus.Info("Successfully reconciled unified push server custom resource")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileHost(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Setting host on config to exposed route
	logrus.Info("Setting unified push server config host")
	upsRoute := &routev1.Route{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultRoutename, Namespace: r.Config.GetNamespace()}, upsRoute)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get route for unified push server: %w", err)
	}

	r.Config.SetHost("https://" + upsRoute.Spec.Host)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not update unified push server config: %w", err)
	}

	logrus.Info("Successfully set unified push server host")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-ups", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost() + "/" + r.Config.GetBlackboxTargetPath(),
		Service: "ups-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating ups blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func preUpgradeBackupExecutor(installation *integreatlyv1alpha1.RHMI) backup.BackupExecutor {
	if installation.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}

	return backup.NewAWSBackupExecutor(
		installation.Namespace,
		"ups-postgres-rhmi",
		backup.PostgresSnapshotType,
	)
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.UPSSubscriptionName,
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
		[]string{productNamespace},
		preUpgradeBackupExecutor(inst),
		serverClient,
		catalogSourceReconciler,
	)
}
