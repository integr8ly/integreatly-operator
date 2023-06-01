package mcg

import (
	"context"
	"errors"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/version"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	noobaav1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8spointer "k8s.io/utils/pointer"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultInstallationNamespace  = "mcg"
	noobaaName                    = "noobaa"
	noobaaDefaultBucketClass      = noobaaName + "-default-bucket-class"
	pvpoolStorageSize             = "16Gi"
	dbStorageSize                 = "1Gi"
	defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"
	threescaleBucket              = "3scale-operator-bucket"
	ThreescaleBucketClaim         = threescaleBucket + "-claim"
	S3RouteName                   = "s3"
)

type Reconciler struct {
	Config        *config.MCG
	ConfigManager config.ConfigReadWriter
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	*resources.Reconciler
	recorder record.EventRecorder
	log      l.Logger
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for mcg")
	}

	mcgConfig, err := configManager.ReadMCG()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve mcg config: %w", err)
	}
	if mcgConfig.GetNamespace() == "" {
		mcgConfig.SetNamespace(installation.Spec.NamespacePrefix + DefaultInstallationNamespace)
		if err := configManager.WriteConfig(mcgConfig); err != nil {
			return nil, fmt.Errorf("error writing mcg config : %w", err)
		}
	}
	if mcgConfig.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			mcgConfig.SetOperatorNamespace(mcgConfig.GetNamespace())
		} else {
			mcgConfig.SetOperatorNamespace(mcgConfig.GetNamespace() + "-operator")
		}
	}

	return &Reconciler{
		Config:        mcgConfig,
		ConfigManager: configManager,
		installation:  installation,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
		log:           logger,
	}, nil
}

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	product := installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductMCG]
	return version.VerifyProductAndOperatorVersion(
		product,
		string(integreatlyv1alpha1.VersionMCG),
		string(integreatlyv1alpha1.OperatorVersionMCG),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Reconciling")
	operatorNamespace := r.Config.GetOperatorNamespace()

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := r.cleanupResources(ctx, serverClient, installation)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
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

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, serverClient, operatorNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, operatorNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.MCGSubscriptionName), err)
		return phase, err
	}

	phase, err = r.ReconcileNoobaa(ctx, serverClient)
	r.log.Infof("ReconcileNoobaa", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile NooBaa", err)
		return phase, err
	}

	phase, err = r.ReconcileObjectBucketClaim(ctx, serverClient)
	r.log.Infof("ReconcileObjectBucketClaim", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile ObjectBucketClaim", err)
		return phase, err
	}

	alertsReconciler, err := r.newAlertReconciler(r.log, r.installation.Spec.Type, config.GetOboNamespace(r.installation.Namespace))
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, serverClient); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile mcg alerts", err)
		return phase, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Infof("Installation reconciled successfully", l.Fields{"productStatus": r.Config.GetProductName()})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		SubscriptionName: constants.MCGSubscriptionName,
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
		[]string{productNamespace},
		backup.NewNoopBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}

func (r *Reconciler) ReconcileNoobaa(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	noobaa := &noobaav1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      noobaaName,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}
	key := k8sclient.ObjectKeyFromObject(noobaa)
	err := serverClient.Get(ctx, key, noobaa)
	if err != nil && !k8serr.IsNotFound(err) {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	defaultStorageClass, err := retrieveDefaultStorageClass(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	status, err := controllerutil.CreateOrUpdate(ctx, serverClient, noobaa, func() error {
		noobaa.Spec.CleanupPolicy.AllowNoobaaDeletion = true
		noobaa.Spec.DisableLoadBalancerService = true
		noobaa.Spec.PVPoolDefaultStorageClass = k8spointer.String(defaultStorageClass.Name)
		dbStorageQuantity, err := resource.ParseQuantity(dbStorageSize)
		if err != nil {
			return err
		}
		noobaa.Spec.DBVolumeResources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: dbStorageQuantity,
			},
		}
		pvPoolStorageQuantity, err := resource.ParseQuantity(pvpoolStorageSize)
		if err != nil {
			return err
		}
		noobaa.Spec.DefaultBackingStoreSpec = &noobaav1.BackingStoreSpec{
			PVPool: &noobaav1.PVPoolSpec{
				StorageClass: defaultStorageClass.Name,
				NumVolumes:   1,
				VolumeResources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: pvPoolStorageQuantity,
					},
				},
			},
			Type: noobaav1.StoreTypePVPool,
		}
		noobaa.Spec.Endpoints = &noobaav1.EndpointsSpec{
			MinCount: 1,
			MaxCount: 2,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("250Mi")},
				Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m"), corev1.ResourceMemory: resource.MustParse("300Mi")},
			},
		}

		noobaa.Spec.CoreResources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m"), corev1.ResourceMemory: resource.MustParse("800Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("400m"), corev1.ResourceMemory: resource.MustParse("900Mi")},
		}

		noobaa.Spec.DBResources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("250Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m"), corev1.ResourceMemory: resource.MustParse("300Mi")},
		}
		owner.AddIntegreatlyOwnerAnnotations(noobaa, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if noobaa.Status.Phase != noobaav1.SystemPhaseReady {
		r.log.Infof("NooBaa deployment in progress", l.Fields{"status": noobaa.Status.Phase})
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if noobaa.Status.Services != nil && len(noobaa.Status.Services.ServiceS3.ExternalDNS) > 0 {
		r.Config.SetHost(noobaa.Status.Services.ServiceS3.ExternalDNS[0])
	}

	r.log.Infof("NooBaa: ", l.Fields{"status": status})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcileObjectBucketClaim(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	objbc := &noobaav1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ThreescaleBucketClaim,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	key := k8sclient.ObjectKeyFromObject(objbc)
	err := serverClient.Get(ctx, key, objbc)
	if err != nil && !k8serr.IsNotFound(err) {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	status, err := controllerutil.CreateOrUpdate(ctx, serverClient, objbc, func() error {
		objbc.Spec.GenerateBucketName = threescaleBucket
		objbc.Spec.StorageClassName = r.Config.GetOperatorNamespace() + ".noobaa.io"
		objbc.Spec.AdditionalConfig = map[string]string{
			"bucketclass": noobaaDefaultBucketClass,
		}
		owner.AddIntegreatlyOwnerAnnotations(objbc, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	r.log.Infof("ObjectBucketClaim: ", l.Fields{"status": status})

	if objbc.Status.Phase != obv1.ObjectBucketClaimStatusPhaseBound {
		r.log.Infof("ObjectBucket provisioning in progress", l.Fields{"phase": objbc.Status.Phase})
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func retrieveDefaultStorageClass(ctx context.Context, serverClient k8sclient.Client) (*storagev1.StorageClass, error) {
	storageList := &storagev1.StorageClassList{}
	err := serverClient.List(ctx, storageList)
	if err != nil {
		return nil, err
	}
	var defaultStorageClass *storagev1.StorageClass
	for i := range storageList.Items {
		if storageList.Items[i].Annotations[defaultStorageClassAnnotation] == "true" {
			defaultStorageClass = &storageList.Items[i]
			break
		}
	}
	if defaultStorageClass == nil {
		return nil, errors.New("unable to determine default storage class")
	}

	return defaultStorageClass, nil
}

func (r *Reconciler) cleanupResources(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	opts := &k8sclient.ListOptions{
		Namespace: r.Config.GetOperatorNamespace(),
	}

	noobaaCRD := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "noobaas.noobaa.io",
		},
	}
	crdExists, err := k8s.Exists(ctx, serverClient, noobaaCRD)
	if err != nil {
		r.log.Error("Error checking NooBaa CRD existence: ", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if !crdExists {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	// Ensure buckets are deleted before we remove the noobaa system
	objectBucketClaims := &noobaav1.ObjectBucketClaimList{}
	err = serverClient.List(ctx, objectBucketClaims, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	for i := range objectBucketClaims.Items {
		objclaim := objectBucketClaims.Items[i]
		err = serverClient.Delete(ctx, &objclaim)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Check all object bucket claims have been deleted
	err = serverClient.List(ctx, objectBucketClaims, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(objectBucketClaims.Items) > 0 {
		r.log.Info("ObjectBucketClaim deletion in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Check all object buckets have been deleted
	objectBuckets := &noobaav1.ObjectBucketList{}
	err = serverClient.List(ctx, objectBuckets, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(objectBuckets.Items) > 0 {
		r.log.Info("ObjectBucket deletion in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	noobaas := &noobaav1.NooBaaList{}
	err = serverClient.List(ctx, noobaas, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	for i := range noobaas.Items {
		noobaa := noobaas.Items[i]
		err = serverClient.Delete(ctx, &noobaa)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Check all noobaas have been deleted
	err = serverClient.List(ctx, noobaas, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(noobaas.Items) > 0 {
		r.log.Info("NooBaa deletion in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
