package apicurio

import (
	"context"
	"fmt"
	apicurio "github.com/integr8ly/integreatly-operator/pkg/apis/apicur/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "apicurio"
	defaultSubscriptionName      = "integreatly-apicurio"
	manifestPackage              = "integreatly-apicurio"
	apicurioName                 = "apicurio"
)

type Reconciler struct {
	Config        *config.Apicurio
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	logger        *logrus.Entry
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.Installation
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	apicurioConfig, err := configManager.ReadApicurio()

	if err != nil {
		return nil, err
	}

	if apicurioConfig.GetNamespace() == "" {
		apicurioConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	return &Reconciler{
		Config:        apicurioConfig,
		extraParams:   make(map[string]string),
		ConfigManager: configManager,
		logger:        logger,
		installation:  installation,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.Installation, product *integreatlyv1alpha1.InstallationProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {

		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		_, err = resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}

		//if both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		_, nsErr := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(nsErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to write config in apicurio reconciler", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}
	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	r.logger.Info("Reconciling Apicurio components")
	apicurioCR := &apicurio.Apicurito{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apicurioName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, apicurioCR, func() error {
		// Specify 2 pods to provide HA
		apicurioCR.Spec.Size = 2
		apicurioCR.Spec.Image = "registry.redhat.io/fuse7/fuse-apicurito:1.5"
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update apicurio custom resource")
	}
	r.logger.Infof("The operation result for apicurio %s was %s", apicurioCR.Name, or)

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	logrus.Infof("%s is successfully reconciled", apicurioName)

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Debug("checking amq streams pods are running")

	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := client.List(ctx, pods, listOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to check apicurio installation: %w", err)
	}

	//expecting 2 pods in total
	if len(pods.Items) < 2 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
checkPodStatus:
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == corev1.ContainersReady {
				if cnd.Status != corev1.ConditionStatus("True") {
					return integreatlyv1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}

	r.logger.Infof("all apicurio pods ready, returning complete")
	return integreatlyv1alpha1.PhaseCompleted, nil
}
