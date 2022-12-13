package mcg

import (
	"context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/version"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "mcg"
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
		mcgConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
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
		// TODO: cleanup resources
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
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

	if r.installation.GetDeletionTimestamp() == nil {
	}

	//TODO:
	// - Reconcile NooBaa CR
	// - Reconcile ObjectBucketClaim CR
	// - Retrieve credentials once provisioned
	// - Create 3scale secret with credentials

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
