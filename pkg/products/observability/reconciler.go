package observability

import (
	"context"
	"fmt"
	observability "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoringcommon"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/version"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "observability"

	// alert manager configuration
	alertManagerRouteName = "kafka-alertmanager"
	configMapNoInit       = "observability-operator-no-init"
	observabilityName     = "observability-stack"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager config.ConfigReadWriter
	Config        *config.Observability
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	extraParams   map[string]string
	recorder      record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductObservability],
		string(integreatlyv1alpha1.VersionObservability),
		string(integreatlyv1alpha1.OperatorVersionObservability),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	config, err := configManager.ReadObservability()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve observability config: %w", err)
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(ns)
		err := configManager.WriteConfig(config)
		if err != nil {
			return nil, err
		}
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, _ quota.ProductConfig) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Observability reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if productNamespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, productNamespace, client)
		if !k8serr.IsNotFound(err) {
			// Mark OO CR for deletion.
			phase, err := r.deleteObservabilityCR(ctx, client, installation, productNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = resources.RemoveNamespace(ctx, installation, client, productNamespace, r.log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		// Check if operatorNamespace is still present before trying to delete it resources
		_, err = resources.GetNS(ctx, operatorNamespace, client)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, client, operatorNamespace, r.log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		// If both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, operatorNamespace, client)
		_, productNSErr := resources.GetNS(ctx, productNamespace, client)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(productNSErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	}, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileConfigMap(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s configmap which is required to disable observability operator initilisting it's own cr", configMapNoInit), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, client, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.ObservabilitySubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, client, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to create components", err)
		return phase, err
	}

	phase, err = monitoringcommon.ReconcileAlertManagerSecrets(ctx, client, r.installation, r.Config.GetNamespace(), alertManagerRouteName)
	r.log.Infof("ReconcileAlertManagerConfigSecret", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("failed to reconcile alert manager config secret " + err.Error())
		}
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alert manager config secret", err)
		return phase, err
	}

	if string(r.Config.GetProductVersion()) != string(integreatlyv1alpha1.VersionObservability) {
		r.Config.SetProductVersion(string(integreatlyv1alpha1.VersionObservability))
		err := r.ConfigManager.WriteConfig(r.Config)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Create a ConfigMap in the operator namespace to prevent observability CR from being created in the operator ns.
	cfgMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapNoInit,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, serverClient, cfgMap, func() error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if op == controllerutil.OperationResultUpdated || op == controllerutil.OperationResultCreated {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}
	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {

	oo := &observability.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: productNamespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, serverClient, oo, func() error {
		disabled := true
		oo.Spec = observability.ObservabilitySpec{
			ConfigurationSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"monitoring-key": r.Config.GetLabelSelector(),
				},
				MatchExpressions: nil,
			},
			SelfContained: &observability.SelfContained{
				DisableRepoSync:         &disabled,
				DisableObservatorium:    &disabled,
				DisablePagerDuty:        &disabled,
				DisableDeadmansSnitch:   &disabled,
				DisableBlackboxExporter: nil,
				SelfSignedCerts:         nil,
				FederatedMetrics:        nil,
				PodMonitorLabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				PodMonitorNamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				ServiceMonitorLabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				ServiceMonitorNamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				RuleLabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				RuleNamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				ProbeLabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				ProbeNamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"monitoring-key": r.Config.GetLabelSelector(),
					},
				},
				AlertManagerConfigSecret: config.AlertManagerConfigSecretName,
			},
			ResyncPeriod: "1h",
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	if op == controllerutil.OperationResultUpdated || op == controllerutil.OperationResultCreated {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: oo.Name, Namespace: oo.Namespace}, oo)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if oo.Status.StageStatus == observability.ResultFailed {
		return integreatlyv1alpha1.PhaseFailed, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling subscription")

	target := marketplace.Target{
		SubscriptionName: constants.ObservabilitySubscriptionName,
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
		r.preUpgradeBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	return backup.NewNoopBackupExecutor()
}

func (r *Reconciler) deleteObservabilityCR(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, targetNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	// If the installation is NOT marked for deletion, return without deleting observability CR
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	oo := &observability.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: targetNamespace,
		},
	}

	// Get the observability CR; return if not found
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oo.Name, Namespace: oo.Namespace}, oo)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Mark the observability CR for deletion
	err = serverClient.Delete(ctx, oo)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}
