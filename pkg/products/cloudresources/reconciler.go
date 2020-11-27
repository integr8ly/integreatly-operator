package cloudresources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	crov1alpha1Types "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	installation  *integreatlyv1alpha1.RHMI
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
		installation:  installation,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	product := installation.Status.Stages[integreatlyv1alpha1.CloudResourcesStage].Products[integreatlyv1alpha1.ProductCloudResources]
	return version.VerifyProductAndOperatorVersion(
		product,
		string(integreatlyv1alpha1.VersionCloudResources),
		string(integreatlyv1alpha1.OperatorVersionCloudResources),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, operatorNamespace, client)
		if !k8serr.IsNotFound(err) {

			phase, err := r.removeSnapshots(ctx, installation, client)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			// overrides cro default deletion strategy to delete resources snapshots
			phase, err = r.createDeletionStrategy(ctx, installation, client)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			// ensure resources are cleaned up before deleting the namespace
			phase, err = r.cleanupResources(ctx, installation, client)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			// remove the namespace
			phase, err = resources.RemoveNamespace(ctx, installation, client, operatorNamespace)
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

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	if err := r.reconcileCIDRValue(ctx, client); err != nil {
		phase := integreatlyv1alpha1.PhaseFailed
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile CIDR value", err)
		return phase, err
	}

	// In this case due to cloudresources reconciler is always installed in the
	// same namespace as the operatorNamespace we pass operatorNamespace as the
	// productNamepace too
	phase, err = r.reconcileSubscription(ctx, client, installation, operatorNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CloudResourceSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileBackupsStorage(ctx, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.newAlertsReconciler().ReconcileAlerts(ctx, client)
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

func (r *Reconciler) removeSnapshots(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	logrus.Infof("Removing postgres and redis snapshots")

	pgSnaps := &crov1alpha1.PostgresSnapshotList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err := client.List(ctx, pgSnaps, listOpts...)
	if err != nil {
		logrus.Error("Failed to list postgres snapshots")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list postgres snapshots: %w", err)
	}

	for _, pgSnap := range pgSnaps.Items {
		logrus.Infof("Deleting postgres snapshot %s", pgSnap.Name)
		if err := client.Delete(ctx, &pgSnap); err != nil {
			logrus.Infof("Failed to delete postgres snapshot %s", pgSnap.Name)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	redisSnaps := &crov1alpha1.RedisSnapshotList{}
	listOpts = []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err = client.List(ctx, redisSnaps, listOpts...)
	if err != nil {
		logrus.Error("Failed to list redis snapshots")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list redis snapshots: %w", err)
	}

	for _, redisSnap := range redisSnaps.Items {
		logrus.Infof("Deleting redis snapshot %s", redisSnap.Name)
		if err := client.Delete(ctx, &redisSnap); err != nil {
			logrus.Infof("Failed to delete redis snapshot %s", redisSnap.Name)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	logrus.Infof("Finished postgres and redis snapshots removal")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createDeletionStrategy(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	if strings.ToLower(installation.Spec.UseClusterStorage) == "false" {
		croStrategyConfig := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cloud-resources-aws-strategies",
				Namespace: installation.Namespace,
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, croStrategyConfig, func() error {
			forceBucketDeletion := true
			skipFinalSnapshot := true
			finalSnapshotIdentifier := ""
			tier := "production"
			resources := map[string]interface{}{
				"blobstorage": aws.S3DeleteStrat{
					ForceBucketDeletion: &forceBucketDeletion,
				},
				"postgres": rds.DeleteDBClusterInput{
					SkipFinalSnapshot: &skipFinalSnapshot,
				},
				"redis": elasticache.DeleteCacheClusterInput{
					FinalSnapshotIdentifier: &finalSnapshotIdentifier,
				},
			}
			for resource, deleteStrategy := range resources {
				err := overrideStrategyConfig(resource, tier, croStrategyConfig, deleteStrategy)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

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

	if len(postgresInstances.Items) > 0 {
		r.logger.Info("deletion of postgres instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if len(redisInstances.Items) > 0 {
		r.logger.Info("deletion of redis instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if len(blobStorages.Items) > 0 {
		r.logger.Info("deletion of blob storage instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// everything has been cleaned up, delete the ns
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBackupsStorage(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	if r.installation.Spec.Type != string(integreatlyv1alpha1.InstallationTypeManaged) {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

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

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.CloudResourceSubscriptionName,
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
		[]string{inst.Namespace}, // TODO why is this this value and not productNamespace?
		backup.NewNoopBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
	)
}

func overrideStrategyConfig(resourceType string, tier string, croStrategyConfig *corev1.ConfigMap, deleteStrategy interface{}) error {
	resource := croStrategyConfig.Data[resourceType]
	strategyConfig := map[string]*aws.StrategyConfig{}
	if err := json.Unmarshal([]byte(resource), &strategyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal strategy mapping for resource type %s %w", resourceType, err)
	}

	deleteStrategyJSON, err := json.Marshal(deleteStrategy)
	if err != nil {
		return err
	}

	strategyConfig[tier].DeleteStrategy = json.RawMessage(deleteStrategyJSON)

	strategyConfigJSON, err := json.Marshal(strategyConfig)
	if err != nil {
		return err
	}
	croStrategyConfig.Data[resourceType] = string(strategyConfigJSON)

	return nil
}

// reconcileCIDRValue sets the CIDR value in the ConfigMap from the addon
// parameter. If the value has already been set, or if the secret is not found,
// it does nothing
func (r *Reconciler) reconcileCIDRValue(ctx context.Context, client k8sclient.Client) error {
	cidrValue, ok, err := addon.GetStringParameter(ctx, client, r.installation.Namespace, "cidr-range")
	if err != nil {
		return err
	}

	//!ok means the param wasn't found so we want to default rather than return
	//but don't do it until the installation object is more than a minute old in case the secret is slow to create
	if !ok || cidrValue == "" && r.installation.ObjectMeta.CreationTimestamp.Time.Before(time.Now().Add(-(1*time.Minute))) {
		cidrValue = "10.1.0.0/16"
	}

	cfgMap := &corev1.ConfigMap{}

	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      "cloud-resources-aws-strategies",
		Namespace: r.installation.Namespace,
	}, cfgMap); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	var network struct {
		Production struct {
			CreateStrategy struct {
				CidrBlock string `json:"CidrBlock"`
			} `json:"createStrategy"`
		} `json:"production"`
	}

	data, ok := cfgMap.Data["_network"]
	if ok {
		if err := json.Unmarshal([]byte(data), &network); err != nil {
			return err
		}

		if network.Production.CreateStrategy.CidrBlock != "" {
			return nil
		}
	}

	network.Production.CreateStrategy.CidrBlock = cidrValue
	networkJSON, err := json.Marshal(network)
	if err != nil {
		return err
	}

	cfgMap.Data["_network"] = string(networkJSON)

	return client.Patch(ctx, cfgMap, k8sclient.Merge)
}
