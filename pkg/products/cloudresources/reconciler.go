package cloudresources

import (
	"context"
	"fmt"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	"github.com/sirupsen/logrus"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	crov1alpha1Types "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
)

const (
	defaultInstallationNamespace = "cloud-resources"
	manifestPackage              = "integreatly-cloud-resources"
)

type Reconciler struct {
	Config        *config.CloudResources
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadCloudResources()
	if err != nil {
		return nil, fmt.Errorf("could not read cloud resources config: %w", err)
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
	logger := logrus.WithFields(logrus.Fields{"product": config.GetProductName()})

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
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	ns := r.Config.GetOperatorNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, ns, client)
		if !k8serr.IsNotFound(err) {
			// ensure resources are cleaned up before deleting the namespace
			phase, err := r.cleanupResources(ctx, installation, client)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			// remove the namespace
			phase, err = resources.RemoveNamespace(ctx, installation, client, ns)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, ns, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", ns), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, client)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", ns), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.CloudResourceSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{installation.Namespace}, backup.NewNoopBackupExecutor(), client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CloudResourceSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileBackupsStorage(ctx, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileKubeStateMetricsOperatorEndpointAvailableAlerts(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile operator endpoint available alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not write cloud resources config: %w", err)
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.CloudResourcesStage, r.Config.GetProductName())
	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) cleanupResources(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("ensuring cloud resources are cleaned up")

	// ensure postgres instances are cleaned up
	postgresInstances := &crov1alpha1.PostgresList{}
	postgresInstanceOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err := client.List(ctx, postgresInstances, postgresInstanceOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list postgres instances: %w", err)
	}
	for _, pgInst := range postgresInstances.Items {
		if err := client.Delete(ctx, &pgInst); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}
	if len(postgresInstances.Items) > 0 {
		r.logger.Info("deletion of postgres instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// ensure redis instances are cleaned up
	redisInstances := &crov1alpha1.RedisList{}
	redisInstanceOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err = client.List(ctx, redisInstances, redisInstanceOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list redis instances: %w", err)
	}
	for _, redisInst := range redisInstances.Items {
		if err := client.Delete(ctx, &redisInst); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}
	if len(redisInstances.Items) > 0 {
		r.logger.Info("deletion of redis instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// ensure blob storage instances are cleaned up
	blobStorages := &crov1alpha1.BlobStorageList{}
	blobStorageOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err = client.List(ctx, blobStorages, blobStorageOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list blobStorage instances: %w", err)
	}
	for _, bsInst := range blobStorages.Items {
		if err := client.Delete(ctx, &bsInst); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}
	if len(blobStorages.Items) > 0 {
		r.logger.Info("deletion of blob storage instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// everything has been cleaned up, delete the ns
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBackupsStorage(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	blobStorageName := fmt.Sprintf("%s%s", constants.BackupsBlobStoragePrefix, installation.Name)
	blobStorage, err := croUtil.ReconcileBlobStorage(ctx, client, defaultInstallationNamespace, installation.Spec.Type, croUtil.TierProduction, blobStorageName, installation.Namespace, r.ConfigManager.GetBackupsSecretName(), installation.Namespace, func(cr metav1.Object) error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile blob storage request: %w", err)
	}

	// wait for the blob storage cr to reconcile
	if blobStorage.Status.Phase != crov1alpha1Types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
