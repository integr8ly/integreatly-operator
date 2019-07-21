package fuse

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "fuse"
	defaultSubscriptionName      = "syndesis"
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
		return nil, errors.Wrap(err, "could not retrieve keycloak codeReadyConfig")
	}

	if fuseConfig.GetNamespace() == "" {
		fuseConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if err = fuseConfig.Validate(); err != nil {
		return nil, errors.Wrap(err, "keycloak codeReadyConfig is not valid")
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

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	reconciledPhase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile namespace for fuse ")
	}

	reconciledPhase, err = r.ReconcileSubscription(ctx, inst, defaultSubscriptionName, r.Config.GetNamespace(), serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile subscription for fuse ")
	}

	reconciledPhase, err = r.reconcileCustomResource(ctx, inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile fuse custom resource ")
	}
	// if we get to the end and no phase set then it is done

	return reconciledPhase, nil
}

func (r *Reconciler) reconcileCustomResource(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	intLimit := -1
	cr := &syn.Syndesis{
		ObjectMeta: v12.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "integreatly",
		},
		TypeMeta: v12.TypeMeta{
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
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
		if errors2.IsNotFound(err) {
			if err := client.Create(ctx, cr); err != nil && !errors2.IsAlreadyExists(err) {
				return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create a syndesis cr when reconciling custom resource")
			}
			return v1alpha1.PhaseInProgress, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get a syndesis cr when reconciling custom resource")
	}
	if cr.Status.Phase != syn.SyndesisPhaseInstalled && cr.Status.Phase != syn.SyndesisPhaseStartupFailed {
		return v1alpha1.PhaseInProgress, nil
	}
	if cr.Status.Phase == syn.SyndesisPhaseStartupFailed {
		return v1alpha1.PhaseFailed, errors.New("syndesis has failed to install " + string(cr.Status.Reason))
	}
	return v1alpha1.PhaseCompleted, nil
}
