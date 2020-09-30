package marin3r

import (
	"context"
	"fmt"
	marin3r "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/version"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	logrus.Infof("after function is finished to reconciling the discovery service")

	phase, err = r.reconcileSecrets(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile secrets"), err)
		return phase, err
	}

	phase, err = r.reconcileAlerts(ctx, client, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	// if the phase is not complete but there's no error, then return the phase
	// this could happen when trying to reconcile the secrets as there is a request to get the service that would
	// be created as a result of the previous reconcileDiscoverService
	// return to allow the service time to be created. on subsequent reconciles the reconculesecrets should reconcile as complete
	// or failed if there's an error
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAlerts(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {

	// TODO: Get the rateLimitUnit and rateLimitRequestsPerUnit from here
	//apiVersion: v1
	//kind: ConfigMap
	//metadata:
	//name: ratelimit-config
	//namespace: marin3r
	//labels:
	//app: ratelimit
	//	part-of: 3scale-saas
	//data:
	//	kuard.yaml: |
	//	domain: kuard
	//	descriptors:
	//	- key: generic_key
	//	value: slowpath
	//	rate_limit:
	//	unit: minute
	//	requests_per_unit: 1

	alertReconciler, err := r.newAlertsReconciler("second", 20000)
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

func (r *Reconciler) reconcileDiscoveryService(ctx context.Context, client k8sclient.Client, productNamespace string, namespacePrefix string) (integreatlyv1alpha1.StatusPhase, error) {
	enabledNamespaces := []string{
		namespacePrefix + "3scale",
	}

	discoveryService := &marin3r.DiscoveryService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "instance",
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
		[]string{productNamespace},
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

func (r *Reconciler) reconcileSecrets(ctx context.Context, client k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	service := &corev1.Service{}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: "marin3r-instance", Namespace: productNamespace}, service)
	if err != nil {
		if k8serr.IsNotFound(err) {
			logrus.Infof("didn't find the service in %s with name marin3r-instance", productNamespace)
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}
	// add annotations to the service to trigger creation of the secret
	annotations := service.GetAnnotations()
	if service.Annotations == nil {
		annotations = map[string]string{}
	}
	annotations["service.beta.openshift.io/serving-cert-secret-name"] = serverSecretName
	service.SetAnnotations(annotations)

	err = client.Update(ctx, service)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	//service should be updated now with the annotations
	//wait for secret to be created
	serverSecret := &corev1.Secret{}
	err = client.Get(ctx, types.NamespacedName{Name: serverSecretName, Namespace: productNamespace}, serverSecret)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, serverSecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(serverSecret, r.installation)
		return nil
	})

	// get the secret data from the server secret
	crt, ok := serverSecret.Data[secretDataCertKey]
	if !ok {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Secret does not contain key %s", secretDataCertKey)
	}
	key, ok := serverSecret.Data[secretDataKeyKey]
	if !ok {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Secret does not contain key %s", secretDataKeyKey)
	}
	secretData := map[string][]byte{}
	// assign the same crt and key to the second secret required by marin3r instance
	secretData[secretDataCertKey] = crt
	secretData[secretDataKeyKey] = key
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caSecretName,
			Namespace: productNamespace,
		},
		Data: secretData,
		Type: "kubernetes.io/tls",
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, caSecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(caSecret, r.installation)
		return nil
	})
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			logrus.Infof("error creating or updating %s secret", caSecret.Name)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
