package observability // Package observability TODO MGDAPI-5833 : this package can be removed

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/obo"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	observability "github.com/redhat-developer/observability-operator/v4/api/v1"
	rbac "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "observability"

	observabilityName            = "observability-stack"
	OpenshiftMonitoringNamespace = "openshift-monitoring"

	blackboxExporterPrefix = "blackbox-exporter"
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

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return true
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {

	ns := GetDefaultNamespace(installation.Spec.NamespacePrefix)
	productConfig, err := configManager.ReadObservability()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve observability config: %w", err)
	}

	productConfig.SetNamespacePrefix(installation.Spec.NamespacePrefix)
	productConfig.SetNamespace(ns)

	if installation.Spec.OperatorsInProductNamespace {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace())
	} else {
		productConfig.SetOperatorNamespace(productConfig.GetNamespace() + "-operator")
	}

	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, err
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

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, _ quota.ProductConfig, _ bool) (integreatlyv1alpha1.StatusPhase, error) {

	r.log.Info("Start Observability reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), true, func() (integreatlyv1alpha1.StatusPhase, error) {
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
		// Delete ClusterRole and ClusterRoleBinding that were created for the blackbox exporter
		err = r.removeRoleandRoleBindingForBlackbox(ctx, client)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// If both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, operatorNamespace, client)
		_, productNSErr := resources.GetNS(ctx, productNamespace, client)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(productNSErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	}, r.log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) deleteObservabilityCR(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, targetNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	// If the installation is NOT marked for deletion, return without deleting observability CR
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	isHiveManaged, err := addon.OperatorIsHiveManaged(ctx, serverClient, inst)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if isHiveManaged {
		// proceed after the dms secret is deleted by the deadmanssnitch-operator to prevent alert false positive
		dmsSecret, err := obo.GetDMSSecret(ctx, serverClient, *inst)
		if err != nil && !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("unexpected error retrieving dead man's snitch secret: %w", err)
		}
		if dmsSecret != "" {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("dead man's snitch secret is still present, requeing")
		}
	}

	oo := &observability.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: targetNamespace,
		},
	}

	// Get the observability CR; return if not found
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: oo.Name, Namespace: oo.Namespace}, oo)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Mark the observability CR for deletion
	err = serverClient.Delete(ctx, oo)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) removeRoleandRoleBindingForBlackbox(ctx context.Context, serverClient k8sclient.Client) (err error) {
	//Get the ClusterRoleBinding
	clusterRoleBinding := &rbac.ClusterRoleBinding{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: blackboxExporterPrefix}, clusterRoleBinding)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the ClusterRoleBinding if serverClient was able to get it
	if err == nil {
		err = serverClient.Delete(ctx, clusterRoleBinding)
		if err != nil && !k8serr.IsNotFound(err) {
			return err
		}
	}

	//Get the ClusterRole
	clusterRole := &rbac.ClusterRole{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: blackboxExporterPrefix}, clusterRole)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the ClusterRole if serverClient was able to get it
	if err == nil {
		err = serverClient.Delete(ctx, clusterRole)
		if err != nil && !k8serr.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func GetDefaultNamespace(installationPrefix string) string {
	return installationPrefix + defaultInstallationNamespace
}
