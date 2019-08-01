package fuse

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "fuse"
	defaultSubscriptionName      = "integreatly-syndesis"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	Config        *config.Fuse
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	fuseConfig, err := configManager.ReadFuse()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve fuse config")
	}

	if fuseConfig.GetNamespace() == "" {
		fuseConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if err = fuseConfig.Validate(); err != nil {
		return nil, errors.Wrap(err, "fuse config is not valid")
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        fuseConfig,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

// Reconcile reads that state of the cluster for fuse and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	phase, err = r.reconcileCustomResource(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	logrus.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

// reconcileCustomResource ensures that the fuse custom resource exists
func (r *Reconciler) reconcileCustomResource(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling fuse custom resource")

	intLimit := -1
	cr := &syn.Syndesis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "integreatly",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Syndesis",
			APIVersion: syn.SchemeGroupVersion.String(),
		},
		Spec: syn.SyndesisSpec{
			Integration: syn.IntegrationSpec{
				Limit: &intLimit,
			},
			Components: syn.ComponentsSpec{
				Server: syn.ServerConfiguration{
					Features: syn.ServerFeatures{
						ExposeVia3Scale: true,
					},
				},
			},
		},
	}
	ownerutil.EnsureOwner(cr, install)

	// attempt to create the custom resource
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
		if k8serr.IsNotFound(err) {
			if err := client.Create(ctx, cr); err != nil && !k8serr.IsAlreadyExists(err) {
				return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create a syndesis cr when reconciling custom resource")
			}
			return v1alpha1.PhaseInProgress, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get a syndesis cr when reconciling custom resource")
	}

	if cr.Status.Phase == syn.SyndesisPhaseStartupFailed {
		return v1alpha1.PhaseFailed, errors.New(fmt.Sprintf("failed to install fuse custom resource: %s", cr.Status.Reason))
	}

	if cr.Status.Phase != syn.SyndesisPhaseInstalled {
		return v1alpha1.PhaseInProgress, nil
	}

	// if there are no errors, the phase is complete
	return v1alpha1.PhaseCompleted, nil
}
