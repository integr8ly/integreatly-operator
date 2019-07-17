package nexus

import (
	"context"
	"fmt"

	nexus "github.com/integr8ly/integreatly-operator/pkg/apis/gpte/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "nexus"
	defaultSubscriptionName      = "nexus"
	resourceName                 = "nexus"
)

type Reconciler struct {
	coreClient    kubernetes.Interface
	Config        *config.Nexus
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
}

func NewReconciler(coreClient kubernetes.Interface, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
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
		coreClient:    coreClient,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
	}, nil
}

func (r *Reconciler) Reconcile(in *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ctx := context.TODO()

	phase, err := r.reconcileNamespace(ctx, serverClient)
	if err != nil && phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileSubscription()
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileOperator()
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileCustomResource(ctx, in, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.handleProgress()
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	logrus.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Debug("reconciling namespace")

	namespace := r.Config.GetNamespace()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      namespace,
		},
	}

	// attempt to create the namespace
	err := client.Create(ctx, ns)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create namespace %s", namespace)
	}

	// if the namespace is active, complete the phase
	if ns.Status.Phase == v1.NamespaceActive {
		return v1alpha1.PhaseCompleted, nil
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) reconcileSubscription() (v1alpha1.StatusPhase, error) {
	r.logger.Debug("reconciling subscription")

	// create the subscription
	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		defaultSubscriptionName,
		marketplace.IntegreatlyChannel,
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create subscription in namespace %s", r.Config.GetNamespace())
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileOperator() (v1alpha1.StatusPhase, error) {
	r.logger.Debug("reconciling installplan")

	// get the installplan for the subscription
	ip, err := r.mpm.GetSubscriptionInstallPlan("nexus", r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			r.logger.Infof("no installplan created yet")
		}

		r.logger.Infof("error getting nexus subscription installplan: %s", err)
		return v1alpha1.PhaseFailed, err
	}

	// if the installplan phase is complete, complete the phase
	r.logger.Infof("nexus installplan phase is %s", ip.Status.Phase)
	if ip != nil && ip.Status.Phase == coreosv1alpha1.InstallPlanPhaseComplete {
		return v1alpha1.PhaseCompleted, nil
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) reconcileCustomResource(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
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

func (r *Reconciler) handleProgress() (v1alpha1.StatusPhase, error) {
	r.logger.Debug("checking nexus pod is running")

	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", resourceName),
	}

	pods, err := r.coreClient.CoreV1().Pods(r.Config.GetNamespace()).List(listOptions)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to list pods in nexus namespace")
	}

	// expecting 1 pod in total
	if len(pods.Items) < 1 {
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
