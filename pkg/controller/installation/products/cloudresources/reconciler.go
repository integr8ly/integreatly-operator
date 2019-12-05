package cloudresources

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
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
	defaultSubscriptionName      = "integreatly-cloud-resources"
	manifestPackage              = "integreatly-cloud-resources"
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
	ns := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		// ensure resources are cleaned up before deleting the namespace
		phase, err := r.cleanupResources(ctx, inst, client)
		fmt.Println("cleanupResources, returnned phase", phase)
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}

		// remove the namespace
		phase, err = resources.RemoveNamespace(ctx, inst, client, ns)
		fmt.Println("removeNamespace, returnned phase", phase)
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}
		return v1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, ns, inst, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, client)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace(), ManifestPackage: manifestPackage}, inst.Namespace, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) cleanupResources(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("ensuring cloud resources are cleaned up")

	// ensure postgres instances are cleaned up
	postgresInstances := &crov1.PostgresList{}
	postgresInstanceOpts := []pkgclient.ListOption{
		pkgclient.InNamespace(inst.Namespace),
	}
	err := client.List(ctx, postgresInstances, postgresInstanceOpts...)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if len(postgresInstances.Items) > 0 {
		r.logger.Info("deletion of postgres instances in progress")
		return v1alpha1.PhaseInProgress, nil
	}

	// ensure redis instances are cleaned up
	redisInstances := &crov1.RedisList{}
	redisInstanceOpts := []pkgclient.ListOption{
		pkgclient.InNamespace(inst.Namespace),
	}
	err = client.List(ctx, redisInstances, redisInstanceOpts...)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if len(redisInstances.Items) > 0 {
		r.logger.Info("deletion of redis instances in progress")
		return v1alpha1.PhaseInProgress, nil
	}

	// ensure blob storage instances are cleaned up
	blobStorages := &crov1.BlobStorageList{}
	blobStorageOpts := []pkgclient.ListOption{
		pkgclient.InNamespace(inst.Namespace),
	}
	err = client.List(ctx, blobStorages, blobStorageOpts...)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if len(blobStorages.Items) > 0 {
		r.logger.Info("deletion of blob storage instances in progress")
		return v1alpha1.PhaseInProgress, nil
	}

	// ensure blob storage instances are cleaned up
	smtpCredentialSets := &crov1.SMTPCredentialSetList{}
	smtpOpts := []pkgclient.ListOption{
		pkgclient.InNamespace(inst.Namespace),
	}
	err = client.List(ctx, smtpCredentialSets, smtpOpts...)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	fmt.Println("SMTP Credential sets", smtpCredentialSets)
	if len(smtpCredentialSets.Items) > 0 {
		r.logger.Info("deletion of smtp credential sets in progress")
		return v1alpha1.PhaseInProgress, nil
	}

	// everything has been cleaned up, delete the ns
	return v1alpha1.PhaseCompleted, nil
}
