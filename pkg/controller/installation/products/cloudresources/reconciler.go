package cloudresources

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "cloud-resources"
	defaultSubscriptionName = "cloud-resources"
)

type Reconciler struct {
	Config        *config.CloudResources
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadCloudResources()
	if err != nil {
		return nil, errors.Wrap(err, "could not read cloud resources config")
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	logger := logrus.WithFields(logrus.Fields{"product": config.GetProductName()})

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	version, err := resources.NewVersion(v1alpha1.OperatorVersionCloudResources)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for cloud resources operator")
	}

	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, client, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}
