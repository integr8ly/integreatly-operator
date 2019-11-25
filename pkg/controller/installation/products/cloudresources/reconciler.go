package cloudresources

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "cloud-resources"
	defaultSubscriptionName      = "integreatly-cloud-resources"
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
		phase, err := resources.RemoveNamespace(ctx, inst, client, ns)
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

	version, err := resources.NewVersion(v1alpha1.OperatorVersionCloudResources)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for cloud resource operator")
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: ns}, inst.Namespace, client, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileS3BlobStorage(ctx, inst, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileS3BlobStorage(ctx context.Context, installation *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	s3BucketSecretName := "s3-bucket"
	s3CredentialsSecretName := "s3-credentials"
	tier := "production"
	operatorNamespace := installation.GetNamespace()
	blobStorage, err := croUtil.ReconcileBlobStorage(
		ctx,
		serverClient,
		installation.Spec.Type,
		tier,
		s3BucketSecretName,
		operatorNamespace,
		s3CredentialsSecretName,
		operatorNamespace,
		func(cr metav1.Object) error {
			resources.AddOwner(cr, installation)
			return nil
		},
	)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile blob storage request")
	}

	// wait for the blob storage cr to reconcile
	if blobStorage.Status.Phase != types.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	return v1alpha1.PhaseCompleted, nil
}
