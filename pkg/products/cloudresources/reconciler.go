package cloudresources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/resources/cluster"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	croStrat "github.com/integr8ly/cloud-resource-operator/pkg/client/types"
	croProviders "github.com/integr8ly/cloud-resource-operator/pkg/providers"
	croAWS "github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
)

const (
	defaultInstallationNamespace = "cloud-resources"
	MaintenanceDay               = "maintenance-day"
	MaintenanceHour              = "maintenance-hour"
	DefaultMaintenanceDay        = time.Thursday
	DefaultMaintenanceHour       = 2
	cidrRangeKeyAws              = "cidr-range"
)

var redisServiceUpdatesToInstall = []string{"elasticache-20210615-002", "elasticache-redis-6-2-6-update-20230109", "elasticache-20230315-001", "elasticache-redis-6-2-update"}

// this timestamp is 2022-01-15-00:00:01
var postgresServiceUpdateTimestamp = []string{"1642204801"}

type Reconciler struct {
	Config        *config.CloudResources
	ConfigManager config.ConfigReadWriter
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for cloud resources")
	}

	productConfig, err := configManager.ReadCloudResources()
	if err != nil {
		return nil, fmt.Errorf("could not read cloud resources config: %w", err)
	}

	productConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)

	if installation.Spec.OperatorsInProductNamespace {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace())
	} else {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace() + "-operator")
	}

	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, fmt.Errorf("error writing cloudresources config : %w", err)
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

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	product := installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductCloudResources]
	return version.VerifyProductAndOperatorVersion(
		product,
		string(integreatlyv1alpha1.VersionCloudResources),
		string(integreatlyv1alpha1.OperatorVersionCloudResources),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, _ quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
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

			phase, err = k8s.EnsureObjectDeleted(ctx, client, &operatorsv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Name:      constants.CloudResourceSubscriptionName,
					Namespace: operatorNamespace,
				},
			})
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
			phase, err = k8s.EnsureObjectDeleted(ctx, client, &operatorsv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("cloud-resources.v%s", integreatlyv1alpha1.OperatorVersionCloudResources),
					Namespace: operatorNamespace,
				},
			})
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
			isHiveManaged, err := addon.OperatorIsHiveManaged(ctx, client, installation)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}
			if !isHiveManaged {
				phase, err = resources.RemoveNamespace(ctx, installation, client, operatorNamespace, r.log)
				if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
					return phase, err
				}
			}
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

	if r.Config.GetStrategiesConfigMapName() == "" {
		err := r.setPlatformStrategyName(ctx, client)
		if err != nil {
			phase := integreatlyv1alpha1.PhaseFailed
			events.HandleError(r.recorder, installation, phase, "Failed to determine strategy name from platform", err)
			return phase, err
		}
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, client, operatorNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", operatorNamespace), err)
		return phase, err
	}

	// Check if STS Cluster, get STS role ARN addon parameter and pass ARN to Secret in CRO namespace
	isSTS, err := sts.IsClusterSTS(ctx, client, r.log)
	if err != nil {
		r.log.Error("Error checking STS mode", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if isSTS {
		phase, err = r.checkStsCredentialsPresent(client, operatorNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to create STS secret", err)
			return phase, err
		}
	}

	phase, err = r.reconcileCloudResourceStrategies(ctx, client)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile Cloud Resource strategies", err)
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

	phase, err = r.ReconcileCsvDeploymentsPriority(
		ctx,
		client,
		fmt.Sprintf("cloud-resources.v%s", integreatlyv1alpha1.OperatorVersionCloudResources),
		r.Config.GetOperatorNamespace(),
		r.installation.Spec.PriorityClassName,
	)
	// We need to be 100% sure that we do not get past this stage if the operator version isn't the desired version
	// For now, using the above logic as temp workaround.
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile cloud-resources csv deployments priority class name", err)
		return phase, err
	}

	phase, err = r.addServiceUpdates(ctx, client, croProviders.RedisResourceType, redisServiceUpdatesToInstall)
	if err != nil {
		phase := integreatlyv1alpha1.PhaseFailed
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile redis service updates", err)
		return phase, err
	}
	if phase == integreatlyv1alpha1.PhaseInProgress {
		return phase, nil
	}

	phase, err = r.addServiceUpdates(ctx, client, croProviders.PostgresResourceType, postgresServiceUpdateTimestamp)
	if err != nil {
		phase := integreatlyv1alpha1.PhaseFailed
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile postgres service updates", err)
		return phase, err
	}

	if phase == integreatlyv1alpha1.PhaseInProgress {
		return phase, nil
	}

	alertsReconciler, err := r.newAlertsReconciler(ctx, client, r.log, r.installation.Spec.Type, config.GetOboNamespace(r.installation.Namespace))
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to get new alerts reconciler", err)
		r.log.Error("Error getting cloud resources alerts reconciler", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = alertsReconciler.ReconcileAlerts(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile operator endpoint available alerts", err)
		return phase, err
	}
	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not write cloud resources config: %w", err)
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Infof("Reconcile successful", l.Fields{"productStatus": r.Config.GetProductName()})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) removeSnapshots(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	r.log.Info("Removing postgres and redis snapshots")

	postgresSnapshotCRD := &apiextensionv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "postgressnapshots.integreatly.org",
		},
	}
	crdExists, err := k8s.Exists(ctx, client, postgresSnapshotCRD)
	if err != nil {
		r.log.Error("Error checking Postgres Snapshot CRD existence: ", err)
		return integreatlyv1alpha1.PhaseFailed, err
	} else if !crdExists {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	pgSnaps := &crov1alpha1.PostgresSnapshotList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err = client.List(ctx, pgSnaps, listOpts...)
	if err != nil {
		r.log.Error("Failed to list postgres snapshots", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list postgres snapshots: %w", err)
	}

	for i := range pgSnaps.Items {
		pgSnap := pgSnaps.Items[i]
		r.log.Infof("Deleting postgres snapshot", l.Fields{"name": pgSnap.Name})
		if err := client.Delete(ctx, &pgSnap); err != nil {
			r.log.Infof("Failed to delete postgres snapshot", l.Fields{"name": pgSnap.Name})
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	redisSnaps := &crov1alpha1.RedisSnapshotList{}
	listOpts = []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err = client.List(ctx, redisSnaps, listOpts...)
	if err != nil {
		r.log.Error("Failed to list redis snapshots", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list redis snapshots: %w", err)
	}

	for i := range redisSnaps.Items {
		redisSnap := redisSnaps.Items[i]
		r.log.Infof("Deleting redis snapshot", l.Fields{"name": redisSnap.Name})
		if err := client.Delete(ctx, &redisSnap); err != nil {
			r.log.Infof("Failed to delete redis snapshot", l.Fields{"name": redisSnap.Name})
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	r.log.Info("Finished postgres and redis snapshots removal")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createDeletionStrategy(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	if strings.ToLower(installation.Spec.UseClusterStorage) == "false" {
		if r.Config.GetStrategiesConfigMapName() == croAWS.DefaultConfigMapName {
			croStrategyConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.Config.GetStrategiesConfigMapName(),
					Namespace: installation.Namespace,
				},
			}
			_, err := controllerutil.CreateOrUpdate(ctx, serverClient, croStrategyConfig, func() error {
				forceBucketDeletion := true
				resourcesConfig := map[string]interface{}{
					"blobstorage": croAWS.S3DeleteStrat{
						ForceBucketDeletion: &forceBucketDeletion,
					},
				}

				if resources.IsSkipFinalDBSnapshots(installation) {
					r.log.Info("RHMI CR is annotated with skip_final_db_snapshots=true so CRO will skip creating Postgres/Redis snapshots")

					skipFinalSnapshot := true
					finalSnapshotIdentifier := ""

					resourcesConfig["postgres"] = rds.DeleteDBClusterInput{
						SkipFinalSnapshot: &skipFinalSnapshot,
					}
					resourcesConfig["redis"] = elasticache.DeleteCacheClusterInput{
						FinalSnapshotIdentifier: &finalSnapshotIdentifier,
					}
				}

				for resource, deleteStrategy := range resourcesConfig {
					err := overrideStrategyConfig(resource, croStrategyConfig, deleteStrategy)
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
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) cleanupResources(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("ensuring cloud resources are cleaned up")

	postgresInstancesCRD := &apiextensionv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "postgres.integreatly.org",
		},
	}
	crdExists, err := k8s.Exists(ctx, client, postgresInstancesCRD)
	if err != nil {
		r.log.Error("Error checking Postgres CRD existence: ", err)
		return integreatlyv1alpha1.PhaseFailed, err
	} else if !crdExists {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	// ensure postgres instances are cleaned up
	postgresInstances := &crov1alpha1.PostgresList{}
	postgresInstanceOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(installation.Namespace),
	}
	err = client.List(ctx, postgresInstances, postgresInstanceOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list postgres instances: %w", err)
	}
	for i := range postgresInstances.Items {
		pgInst := postgresInstances.Items[i]
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
	for i := range redisInstances.Items {
		redisInst := redisInstances.Items[i]
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
	for i := range blobStorages.Items {
		bsInst := blobStorages.Items[i]
		if err := client.Delete(ctx, &bsInst); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	if len(postgresInstances.Items) > 0 {
		r.log.Info("deletion of postgres instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if len(redisInstances.Items) > 0 {
		r.log.Info("deletion of redis instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if len(blobStorages.Items) > 0 {
		r.log.Info("deletion of blob storage instances in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// everything has been cleaned up, delete the ns
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		SubscriptionName: constants.CloudResourceSubscriptionName,
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
		[]string{inst.Namespace}, // TODO why is this this value and not productNamespace?
		backup.NewNoopBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}

func overrideStrategyConfig(resourceType string, croStrategyConfig *corev1.ConfigMap, deleteStrategy interface{}) error {
	resource := croStrategyConfig.Data[resourceType]
	strategyConfig := map[string]*croAWS.StrategyConfig{}
	if err := json.Unmarshal([]byte(resource), &strategyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal strategy mapping for resource type %s %w", resourceType, err)
	}

	for tier := range strategyConfig {
		deleteStrategyJSON, err := json.Marshal(deleteStrategy)
		if err != nil {
			return err
		}
		strategyConfig[tier].DeleteStrategy = json.RawMessage(deleteStrategyJSON)
	}

	strategyConfigJSON, err := json.Marshal(strategyConfig)
	if err != nil {
		return err
	}

	croStrategyConfig.Data[resourceType] = string(strategyConfigJSON)

	return nil
}

func (r *Reconciler) addServiceUpdates(ctx context.Context, client k8sclient.Client, resourceType croProviders.ResourceType, updates []string) (integreatlyv1alpha1.StatusPhase, error) {
	if r.Config.GetStrategiesConfigMapName() == croAWS.DefaultConfigMapName {
		cfgMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.Config.GetStrategiesConfigMapName(),
				Namespace: r.installation.Namespace,
			},
		}

		op, err := controllerutil.CreateOrUpdate(ctx, client, cfgMap, func() error {

			var rawStrategy map[string]*croAWS.StrategyConfig
			if err := json.Unmarshal([]byte(cfgMap.Data[string(resourceType)]), &rawStrategy); err != nil {
				return err
			}

			var updateConfig []string
			if err := json.Unmarshal(rawStrategy[croUtil.TierProduction].ServiceUpdates, &updateConfig); err != nil {
				return err
			}

			updateConfig = updates
			updatesMarshalled, err := json.Marshal(updateConfig)
			if err != nil {
				return err
			}

			rawStrategy[croUtil.TierProduction].ServiceUpdates = updatesMarshalled

			marshalledStrategy, err := json.Marshal(rawStrategy)
			if err != nil {
				return err
			}
			cfgMap.Data[string(resourceType)] = string(marshalledStrategy)

			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		if op == controllerutil.OperationResultUpdated {
			return integreatlyv1alpha1.PhaseInProgress, nil
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// reconcileCIDRValue sets the CIDR value in the ConfigMap from the addon
// parameter. If the value has already been set, or if the secret is not found,
// it does nothing
func (r *Reconciler) reconcileCIDRValue(ctx context.Context, client k8sclient.Client) error {
	cidrValueID := ""
	platformType, err := cluster.GetPlatformType(ctx, client)
	if err != nil {
		return fmt.Errorf("unable to retrieve platform type %v", err)
	}
	switch platformType {
	case configv1.AWSPlatformType:
		cidrValueID = cidrRangeKeyAws
	default:
		return fmt.Errorf("unsupported platform type %s", platformType)
	}

	cidrValue, ok, err := addon.GetStringParameter(ctx, client, r.installation.Namespace, cidrValueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve cidr range value %v", err)
	}
	if !ok || cidrValue == "" && r.installation.ObjectMeta.CreationTimestamp.Time.Before(time.Now().Add(-(1*time.Minute))) {
		cidrValue = ""
	}
	cfgMap := &corev1.ConfigMap{}

	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      r.Config.GetStrategiesConfigMapName(),
		Namespace: r.installation.Namespace,
	}, cfgMap); err != nil {
		return err
	}

	type TierCreateStrategy struct {
		CreateStrategy struct {
			CidrBlock string `json:"CidrBlock"`
		} `json:"createStrategy"`
	}

	network := map[string]*TierCreateStrategy{}

	data, ok := cfgMap.Data["_network"]
	if ok {
		if err := json.Unmarshal([]byte(data), &network); err != nil {
			return err
		}

		// If its already set do not override
		if network != nil && network[croUtil.TierProduction] != nil && network[croUtil.TierProduction].CreateStrategy.CidrBlock != "" {
			return nil
		}
	}

	if network == nil {
		network = map[string]*TierCreateStrategy{}
	}

	if network[croUtil.TierProduction] == nil {
		r.log.Info("Add production network aws strategy")
		network[croUtil.TierProduction] = &TierCreateStrategy{}
		network[croUtil.TierProduction].CreateStrategy.CidrBlock = ""
	}

	for key := range network {
		network[key].CreateStrategy.CidrBlock = cidrValue
	}

	networkJSON, err := json.Marshal(network)
	if err != nil {
		return err
	}

	cfgMap.Data["_network"] = string(networkJSON)

	return client.Patch(ctx, cfgMap, k8sclient.Merge)
}

// createSTSARNSecret create the STS arn secret - should be already validated in preflight checks
func (r *Reconciler) checkStsCredentialsPresent(client k8sclient.Client, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	stsCredentials := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: operatorNamespace, Name: sts.CredsSecretName}, stsCredentials)

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get %s secret in %s namespace", sts.CredsSecretName, operatorNamespace)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// reconcileCloudResourceStrategies
// reconcile cro strategy config map, RHMI operator does not care what infrastructure the cluster is running in
// as we support different cloud providers this CRO Reconcile Function will ensure the correct infrastructure strategies are provisioned
//
// this function was part of the rhmiconfig controller, which has sense been removed.
func (r *Reconciler) reconcileCloudResourceStrategies(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling cloud resource maintenance strategies")

	maintenanceDay, _, err := addon.GetStringParameter(ctx, client, r.ConfigManager.GetOperatorNamespace(), MaintenanceDay)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failure to get maintenance day parameter: %v", err)
	}

	var day time.Weekday
	if maintenanceDay != "" {
		parsedDay, err := strconv.ParseInt(maintenanceDay, 0, 64)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failure to parse maintenance day parameter: %v", err)
		}
		day = time.Weekday(parsedDay)
	} else {
		day = DefaultMaintenanceDay
	}

	maintenanceHour, _, err := addon.GetStringParameter(ctx, client, r.ConfigManager.GetOperatorNamespace(), MaintenanceHour)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failure to get maintenance hour parameter: %v", err)
	}

	var hour int
	if maintenanceHour != "" {
		parsedHour, err := strconv.ParseInt(maintenanceHour, 0, 64)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failure to parse maintenance hour parameter: %v", err)
		}
		hour = int(parsedHour)
	} else {
		hour = DefaultMaintenanceHour
	}

	timeConfig := croStrat.NewStrategyTimeConfig(3, 01, day, hour, 00)

	err = croUtil.ReconcileStrategyMaps(ctx, client, timeConfig, croUtil.TierProduction, r.ConfigManager.GetOperatorNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failure to reconcile strategy map: %v", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) setPlatformStrategyName(ctx context.Context, client k8sclient.Client) error {
	platformType, err := cluster.GetPlatformType(ctx, client)
	if err != nil {
		return err
	}
	switch platformType {
	case configv1.AWSPlatformType:
		r.Config.SetStrategiesConfigMapName(croAWS.DefaultConfigMapName)
	default:
		return fmt.Errorf("unsupported platform type %s", platformType)
	}
	return nil
}
