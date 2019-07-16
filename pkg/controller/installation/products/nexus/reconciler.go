package nexus

import (
	"context"

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

func (r *Reconciler) Reconcile(inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	ctx := context.TODO()

	switch v1alpha1.StatusPhase(phase) {
	case v1alpha1.PhaseNone:
		return r.handleNoPhase(ctx, serverClient)
	case v1alpha1.PhaseAwaitingNS:
		return r.reconcileNamespace(ctx, serverClient)
	case v1alpha1.PhaseCreatingSubscription:
		return r.reconcileSubscription()
	case v1alpha1.PhaseCreatingComponents:
		return r.reconcileCustomResource(ctx, inst, serverClient)
	case v1alpha1.PhaseAwaitingOperator:
		return r.handleAwaitingOperator()
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase()
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		return v1alpha1.PhaseFailed, errors.New("installation of nexus failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("unknown phase for nexus: " + string(phase))
	}
}

func (r *Reconciler) handleNoPhase(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      r.Config.GetNamespace(),
		},
	}

	err := client.Create(ctx, ns)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}

	err := client.Get(ctx, pkgclient.ObjectKey{Name: r.Config.GetNamespace()}, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if ns.Status.Phase == v1.NamespaceTerminating {
		r.logger.Debugf("namespace %s is terminating, maintaining phase to try again on next reconcile", r.Config.GetNamespace())
		return v1alpha1.PhaseAwaitingNS, nil
	}
	if ns.Status.Phase == v1.NamespaceActive {
		r.logger.Infof("creating subscription for Nexus")
		return v1alpha1.PhaseCreatingSubscription, nil
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) reconcileSubscription() (v1alpha1.StatusPhase, error) {
	r.logger.Debugf("creating subscription %s from channel %s in namespace: %s", defaultSubscriptionName, marketplace.IntegreatlyChannel, r.Config.GetNamespace())

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

	return v1alpha1.PhaseAwaitingOperator, nil
}

func (r *Reconciler) handleAwaitingOperator() (v1alpha1.StatusPhase, error) {
	r.logger.Infof("checking installplan is created for subscription %s in namespace: %s", defaultSubscriptionName, r.Config.GetNamespace())

	ip, err := r.mpm.GetSubscriptionInstallPlan("nexus", r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			r.logger.Infof("no installplan created yet")
			return v1alpha1.PhaseAwaitingOperator, nil
		}

		r.logger.Infof("error getting nexus subscription installplan: %s", err)
		return v1alpha1.PhaseFailed, err
	}

	r.logger.Infof("nexus installplan phase is %s", ip.Status.Phase)
	if ip.Status.Phase != coreosv1alpha1.InstallPlanPhaseComplete {
		return v1alpha1.PhaseAwaitingOperator, nil
	}

	r.logger.Infof("nexus installplan is complete. Installation ready")
	return v1alpha1.PhaseCreatingComponents, nil
}

func (r *Reconciler) reconcileCustomResource(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("creating nexus custom resource")

	ref := metav1.NewControllerRef(install, v1alpha1.SchemaGroupVersionKind)
	cr := &nexus.Nexus{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*ref,
			},
			Namespace: r.Config.GetNamespace(),
			Name:      defaultSubscriptionName,
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
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
		if k8serr.IsNotFound(err) {
			if err := client.Create(ctx, cr); err != nil && !k8serr.IsAlreadyExists(err) {
				return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create a nexus custom resource")
			}

			r.logger.Info("created custom resource for nexus")
			return v1alpha1.PhaseInProgress, nil
		}

		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create a nexus custom resource")
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	pods, err := r.coreClient.CoreV1().Pods(r.Config.GetNamespace()).List(metav1.ListOptions{})
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
					r.logger.Infof("pod not ready, returning in progress: %+v", cnd.Status)
					return v1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}

	r.logger.Infof("all pods ready, nexus complete")
	return v1alpha1.PhaseCompleted, nil
}
