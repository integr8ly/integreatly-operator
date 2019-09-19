package nexus

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"

	nexus "github.com/integr8ly/integreatly-operator/pkg/apis/gpte/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "nexus"
	defaultSubscriptionName      = "integreatly-nexus"
	resourceName                 = "nexus"
)

type Reconciler struct {
	Config        *config.Nexus
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadNexus()
	if err != nil {
		return nil, errors.Wrap(err, "could not read nexus config")
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if err = config.Validate(); err != nil {
		return nil, errors.Wrap(err, "nexus config is not valid")
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

// Reconcile reads that state of the cluster for nexus and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	version, err := resources.NewVersion(v1alpha1.OperatorVersionNexus)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for nexus")
	}
	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, client, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileCustomResource(ctx, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.handleProgress(ctx, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileCustomResource(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Debug("reconciling nexus custom resource")

	cr := &nexus.Nexus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      resourceName,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: nexus.SchemeGroupVersion.String(),
			Kind:       nexus.NexusKind,
		},
		Spec: nexus.NexusSpec{
			NexusVolumeSize:    "10Gi",
			NexusSSL:           true,
			NexusImageTag:      "latest",
			NexusCPURequest:    1,
			NexusCPULimit:      2,
			NexusMemoryRequest: "2Gi",
			NexusMemoryLimit:   "2Gi",
		},
	}

	// attempt to create the custom resource
	if err := client.Create(ctx, cr); err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get or create a nexus custom resource")
	}

	// if there are no errors, the phase is complete
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgress(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Debug("checking nexus pods are running")

	pods := &corev1.PodList{}
	err := client.List(ctx, &pkgclient.ListOptions{Namespace: r.Config.GetNamespace()}, pods)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to list pods in nexus namespace")
	}

	// expecting 2 pods in total
	if len(pods.Items) < 2 {
		return v1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
checkPodStatus:
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == v1.ContainersReady {
				if cnd.Status != v1.ConditionStatus("True") {
					return v1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}

	r.logger.Infof("all pods ready, nexus complete")
	return v1alpha1.PhaseCompleted, nil
}
