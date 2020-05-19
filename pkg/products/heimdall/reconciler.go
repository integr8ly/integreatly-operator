package heimdall

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"github.com/sirupsen/logrus"

	monitorv1alpha1 "github.com/integr8ly/heimdall/pkg/apis/imagemonitor/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "heimdall"
	defaultHeimdallName          = "rhmi-heimdall"
	manifestPackage              = "integreatly-heimdall"
)

type Reconciler struct {
	Config        *config.Heimdall
	ConfigManager config.ConfigReadWriter
	installation  *integreatlyv1alpha1.RHMI
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadHeimdall()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve heimdall config: %w", err)
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
			Name:      "heimdall",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	// handle any finalizers
	phase, err := r.ReconcileFinalizer(ctx, serverClient, r.installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// clean up product namespace
		phase, err := resources.RemoveNamespace(ctx, r.installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		// clean up operator namespace
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

	// reconcile operator namespace
	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), r.installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	// reconcile product namespace
	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), r.installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}

	// fetch namespace for subscription
	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// reconcile subscription
	preUpgradeBackupsExecutor := backup.NewNoopBackupExecutor()
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.CodeReadySubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, preUpgradeBackupsExecutor, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CodeReadySubscriptionName), err)
		return phase, err
	}

	// reconcile imagemonitor cr
	phase, err = r.reconcileImageMonitor(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile imagemonitor", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("checking that imagemonitor custom resource is marked as available")

	imageMonitor := &monitorv1alpha1.ImageMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultHeimdallName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: defaultHeimdallName, Namespace: r.Config.GetNamespace()}, imageMonitor)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve imagemonitor: %w", err)
	}

	//if imageMonitor.Status.Reports != "Available" {
	//	return integreatlyv1alpha1.PhaseInProgress, nil
	//}

	// need to check pods as heimdall operator seems to lack an installation status value
	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err = client.List(ctx, pods, listOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to check get pods in heimdall installation: %w", err)
	}

	//expecting 8 pods in total
	if len(pods.Items) < 8 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
checkPodStatus:
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == corev1.ContainersReady {
				if cnd.Status != corev1.ConditionStatus("True") {
					return integreatlyv1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}
	r.logger.Infof("all pods ready, returning complete")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileImageMonitor(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	hdConfig, err := r.ConfigManager.ReadHeimdall()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve heimdall config: %w", err)
	}
	r.logger.Infof("creating required custom resources in namespace: %s", r.Config.GetNamespace())

	heimdall, err := r.createImageMonitor(ctx, hdConfig, client)

	if heimdall == nil {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) createImageMonitor(ctx context.Context, heimdall *config.Heimdall, client k8sclient.Client) (*monitorv1alpha1.ImageMonitor, error) {

	imageMonitor := &monitorv1alpha1.ImageMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultHeimdallName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, imageMonitor, func() error {
		imageMonitor.Name = defaultHeimdallName
		imageMonitor.Namespace = r.Config.GetNamespace()
		imageMonitor.APIVersion = fmt.Sprintf(
			"%s/%s",
			monitorv1alpha1.SchemeGroupVersion.Group,
			monitorv1alpha1.SchemeGroupVersion.Version,
		)
		imageMonitor.Kind = "ImageMonitor"
		imageMonitor.Spec.ExcludePattern = ""

		owner.AddIntegreatlyOwnerAnnotations(imageMonitor, r.installation)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create image monitor resource: %w", err)
	}

	return imageMonitor, nil
}
