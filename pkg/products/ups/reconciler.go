package ups

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	monitoringv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	routev1 "github.com/openshift/api/route/v1"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "ups"
	defaultUpsName               = "ups"
	defaultSubscriptionName      = "integreatly-unifiedpush"
	defaultRoutename             = defaultUpsName + "-unifiedpush-proxy"
	manifestPackage              = "integreatly-unifiedpush"
	tier                         = "production"
)

type Reconciler struct {
	Config        *config.Ups
	ConfigManager config.ConfigReadWriter
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

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", defaultUpsName)

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
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	ns := r.Config.GetNamespace()

	phase, err = r.ReconcileNamespace(ctx, ns, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", ns), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", ns), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Namespace: r.Config.GetOperatorNamespace(), Channel: marketplace.IntegreatlyChannel, ManifestPackage: manifestPackage}, []string{ns}, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)

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
	postgresName := fmt.Sprintf("ups-postgres-%s", installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, client, defaultInstallationNamespace, installation.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres request: %w", err)
	}

	// wait for the postgres instance to reconcile
	if postgres.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	alertNamespace := postgres.Namespace
	alertResourceID := postgres.Name
	alertProductName := postgres.Labels["productName"]
	alertName := alertProductName + "RDSInstanceUnavailable"

	if err = r.CreateRDSAvailabilityAlert(ctx, postgres, client, alertName, alertNamespace, alertResourceID, alertProductName); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create alert: %s for Postgresql: %s error: %w", alertName, alertResourceID, err)
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

	err = client.Get(ctx, k8sclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr)
	if err != nil {
		// If the error is not an isNotFound error
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// Otherwise create the cr
		owner.AddIntegreatlyOwnerAnnotations(cr, installation)
		if err := client.Create(ctx, cr); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create unifiedpush server custom resource during reconcile: %w", err)
		}
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

// CreateRDSAvailabilityAlert Call this when we create the RDS, to create a
// PrometheusRule alert to watch for the availability of the RDS instance
func (r *Reconciler) CreateRDSAvailabilityAlert(ctx context.Context, cr *v1alpha1.Postgres, serverClient k8sclient.Client,
	alertName string, alertNamespace string, alertResourceID string, alertProductName string,
) error {
	ruleName := fmt.Sprintf("availability-rule-%s", alertResourceID)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(cro_aws_rds_available{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			cr.Namespace, alertResourceID, alertProductName),
	)
	alertDescription := fmt.Sprintf("The product: %s, RDS instance: %s, in namespace: %s is unavailable", alertProductName, alertResourceID, alertNamespace)
	labels := map[string]string{
		"severity":    "critical",
		"productName": cr.Labels["productName"],
	}

	// CreatePrometheusRule(ruleName string, namespace string, alertRuleName string,
	//	description string, alertExp intstr.IntOrString, labels map[string]string)
	pr, err := croResources.CreatePrometheusRule(ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
	if err != nil {
		return err
	}

	// Unless it already exists, call the kubernetes api and create this PrometheusRule
	// Replace this with CreateOrUpdate if we can figure it out
	err = serverClient.Create(ctx, pr)
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return errorUtil.Wrap(err, fmt.Sprintf("exception calling Create prometheusrule: %s", ruleName))
		}
	}
	logrus.Infof(fmt.Sprintf("PrometheusRule: %s reconciled successfully.", pr.Name))
	return nil
}

// DeleteRDSAvailabilityAlert Call this when we delete the RDS instance,
// The PrometheusRule alert will also be deleted.
func (r *Reconciler) DeleteRDSAvailabilityAlert(ctx context.Context, serverClient k8sclient.Client, namespace string, instanceID string) error {
	// query the kubernetes api to find the object we're looking for
	ruleName := fmt.Sprintf("availability-rule-%s", instanceID)

	pr := &prometheusv1.PrometheusRule{}
	selector := client.ObjectKey{
		Namespace: namespace,
		Name:      ruleName,
	}

	if err := serverClient.Get(ctx, selector, pr); err != nil {
		return errorUtil.Wrapf(err, "exception calling DeleteRDSAvailabilityAlert: %s", ruleName)
	}

	// call delete on that object
	if err := serverClient.Delete(ctx, pr); err != nil {
		return errorUtil.Wrapf(err, "exception calling DeleteRDSAvailabilityAlert: %s", ruleName)
	}
	logrus.Infof(fmt.Sprintf("PrometheusRule: %s reconciled successfully.", pr.Name))

	return nil
}

// CreateElastiCacheAvailabilityAlert Call this when we create the ElastiCache, to create a
// PrometheusRule alert to watch for the availability of the ElastiCache instance
func (r *Reconciler) CreateElastiCacheAvailabilityAlert(ctx context.Context, cr *v1alpha1.Redis, serverClient k8sclient.Client,
	alertName string, alertNamespace string, alertResourceID string, alertProductName string,
) error {
	ruleName := fmt.Sprintf("availability-rule-%s", alertResourceID)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(cro_aws_elasticache_available{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			cr.Namespace, alertResourceID, alertProductName),
	)
	alertDescription := fmt.Sprintf("The product: %s, ElastiCache instance: %s, in namespace: %s is unavailable", alertProductName, alertResourceID, alertNamespace)
	labels := map[string]string{
		"severity":    "critical",
		"productName": cr.Labels["productName"],
	}

	// CreatePrometheusRule(ruleName string, namespace string, alertRuleName string,
	//	description string, alertExp intstr.IntOrString, labels map[string]string)
	pr, err := croResources.CreatePrometheusRule(ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
	if err != nil {
		return err
	}

	// Unless it already exists, call the kubernetes api and create this PrometheusRule
	// Replace this with CreateOrUpdate if we can figure it out
	err = serverClient.Create(ctx, pr)
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return errorUtil.Wrap(err, fmt.Sprintf("exception calling Create prometheusrule: %s", ruleName))
		}
	}
	logrus.Infof(fmt.Sprintf("PrometheusRule: %s reconciled successfully.", pr.Name))
	return nil
}

// DeleteElastiCacheAvailabilityAlert We call this when we delete an ElastiCache instance,
// it removes the prometheusrule alert which watches for the availability of the instance.
func (r *Reconciler) DeleteElastiCacheAvailabilityAlert(ctx context.Context, serverClient k8sclient.Client, namespace string, instanceID string) error {
	// query the kubernetes api to find the object we're looking for
	ruleName := fmt.Sprintf("availability-rule-%s", instanceID)

	pr := &prometheusv1.PrometheusRule{}
	selector := client.ObjectKey{
		Namespace: namespace,
		Name:      ruleName,
	}

	if err := serverClient.Get(ctx, selector, pr); err != nil {
		return errorUtil.Wrapf(err, "exception calling DeleteRDSAvailabilityAlert: %s", ruleName)
	}

	// call delete on that object
	if err := serverClient.Delete(ctx, pr); err != nil {
		return errorUtil.Wrapf(err, "exception calling DeleteRDSAvailabilityAlert: %s", ruleName)
	}
	logrus.Infof(fmt.Sprintf("PrometheusRule: %s reconciled successfully.", pr.Name))

	return nil
}
