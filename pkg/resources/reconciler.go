package resources

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	productsConfig "github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	oauthv1 "github.com/openshift/api/oauth/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	OwnerLabelKey = v1alpha1.SchemeGroupVersion.Group + "/installation-uid"
)

// This is the base reconciler that all the other reconcilers extend. It handles things like namespace creation, subscription creation etc

type Reconciler struct {
	mpm marketplace.MarketplaceInterface
}

func NewReconciler(mpm marketplace.MarketplaceInterface) *Reconciler {
	return &Reconciler{
		mpm: mpm,
	}
}

func (r *Reconciler) ReconcileOauthClient(ctx context.Context, inst *v1alpha1.Installation, name, secret string, redirectUris []string, apiClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	oauthClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, apiClient, oauthClient, func() error {
		PrepareObject(oauthClient, inst)
		oauthClient.Secret = secret
		oauthClient.RedirectURIs = redirectUris
		oauthClient.GrantMethod = oauthv1.GrantHandlerPrompt
		return nil
	})

	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to %v oauth client: %s", or, oauthClient.Name)
	}

	return v1alpha1.PhaseCompleted, nil
}

func GetNS(ctx context.Context, namespace string, client pkgclient.Client) (*v1.Namespace, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := client.Get(ctx, pkgclient.ObjectKey{Name: ns.Name}, ns)
	if err == nil {
		// workaround for https://github.com/kubernetes/client-go/issues/541
		ns.TypeMeta = metav1.TypeMeta{Kind: "Namespace", APIVersion: v1.SchemeGroupVersion.Version}
	}
	return ns, err
}

func (r *Reconciler) ReconcileNamespace(ctx context.Context, namespace string, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns, err := GetNS(ctx, namespace, client)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not retrieve namespace: %s", ns.Name)
		}
		PrepareObject(ns, inst)
		if err = client.Create(ctx, ns); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create namespace: %s", ns.Name)
		}
		return v1alpha1.PhaseCompleted, nil
	}
	// ns exists so check it is our namespace
	if !IsOwnedBy(ns, inst) && ns.Status.Phase != v1.NamespaceTerminating {
		return v1alpha1.PhaseFailed, errors.New("existing namespace found with name " + ns.Name + " but it is not owned by the integreatly installation and it isn't being deleted")
	}
	if ns.Status.Phase == v1.NamespaceTerminating {
		logrus.Debugf("namespace %s is terminating, maintaining phase to try again on next reconcile", namespace)
		return v1alpha1.PhaseInProgress, nil
	}
	PrepareObject(ns, inst)
	if err := client.Update(ctx, ns); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to update the ns definition ")
	}
	if ns.Status.Phase != v1.NamespaceActive {
		return v1alpha1.PhaseInProgress, nil
	}
	return v1alpha1.PhaseCompleted, nil
}

type finalizerFunc func() (v1alpha1.StatusPhase, error)

func (r *Reconciler) ReconcileFinalizer(ctx context.Context, client pkgclient.Client, inst *v1alpha1.Installation, productName string, finalFunc finalizerFunc) (v1alpha1.StatusPhase, error) {
	finalizer := "finalizer." + productName + ".integreatly.org"
	// Add finalizer if not there
	err := AddFinalizer(ctx, inst, client, finalizer)
	if err != nil {
		logrus.Error(fmt.Sprintf("Error adding finalizer %s to installation", finalizer), err)
		return v1alpha1.PhaseFailed, err
	}

	// Run finalization logic. If it fails, don't remove the finalizer
	// so that we can retry during the next reconciliation
	if inst.GetDeletionTimestamp() != nil {
		if contains(inst.GetFinalizers(), finalizer) {
			phase, err := finalFunc()
			if err != nil || phase != v1alpha1.PhaseCompleted {
				return phase, err
			}

			// Remove the finalizer to allow for deletion of the installation cr
			logrus.Infof("Removing finalizer: %s", finalizer)
			err = RemoveProductFinalizer(ctx, inst, client, productName)
			if err != nil {
				return v1alpha1.PhaseFailed, err
			}
		}
		// Don't continue reconciling the product
		return v1alpha1.PhaseNone, nil
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcilePullSecret(ctx context.Context, namespace, secretName string, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	pullSecretName := DefaultOriginPullSecretName
	if secretName != "" {
		pullSecretName = secretName
	}

	err := CopyDefaultPullSecretToNameSpace(namespace, pullSecretName, inst, client, ctx)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "error creating/updating secret '%s' in namespace: '%s'", pullSecretName, namespace)
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcileSubscription(ctx context.Context, owner ownerutil.Owner, t marketplace.Target, targetNS string, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling subscription %s from channel %s in namespace: %s", t.Pkg, "integreatly", t.Namespace)
	err := r.mpm.InstallOperator(ctx, client, owner, t, []string{targetNS}, operatorsv1alpha1.ApprovalManual)

	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create subscription in namespace: %s", t.Namespace))
	}
	ips, _, err := r.mpm.GetSubscriptionInstallPlans(ctx, client, t.Pkg, t.Namespace)
	if err != nil {

		// this could be the install plan or subscription so need to check if sub nil or not TODO refactor
		if k8serr.IsNotFound(errors.Cause(err)) {
			return v1alpha1.PhaseAwaitingOperator, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve installplan and subscription in namespace: %s", t.Namespace))
	}

	if len(ips.Items) == 0 {
		return v1alpha1.PhaseInProgress, nil
	}

	for _, ip := range ips.Items {
		err = upgradeApproval(ctx, client, &ip)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "error approving installplan for "+t.Pkg)
		}

		//if it's approved but not complete, then it's in progress
		if ip.Status.Phase != operatorsv1alpha1.InstallPlanPhaseComplete && ip.Spec.Approved {
			logrus.Infof("%s install plan is not complete yet ", t.Pkg)
			return v1alpha1.PhaseInProgress, nil
			//if it's not approved by now, then it will not be approved by this version of the integreatly-operator
		} else if !ip.Spec.Approved {
			logrus.Infof("%s has an upgrade installplan above the maximum allowed version", t.Pkg)
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func PrepareObject(ns metav1.Object, install *v1alpha1.Installation) {
	labels := ns.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels["integreatly"] = "true"
	labels["monitoring-key"] = "middleware"
	labels[OwnerLabelKey] = string(install.GetUID())
	monitoringConfig := productsConfig.NewMonitoring(productsConfig.ProductConfig{})
	labels[monitoringConfig.GetLabelSelectorKey()] = monitoringConfig.GetLabelSelector()
	ns.SetLabels(labels)
}

func IsOwnedBy(o metav1.Object, owner *v1alpha1.Installation) bool {
	// TODO change logic to check for our finalizer?
	for k, v := range o.GetLabels() {
		if k == OwnerLabelKey && v == string(owner.UID) {
			return true
		}
	}
	return false
}
