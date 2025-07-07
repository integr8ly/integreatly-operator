package resources

import (
	"context"
	"errors"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	projectv1 "github.com/openshift/api/project/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	oauthv1 "github.com/openshift/api/oauth/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	OwnerLabelKey = integreatlyv1alpha1.GroupVersion.Group + "/installation-uid"
)

// This is the base reconciler that all the other reconcilers extend. It handles things like namespace creation, subscription creation etc

type Reconciler struct {
	mpm                marketplace.MarketplaceInterface
	productDeclaration *marketplace.ProductDeclaration
}

func NewReconciler(mpm marketplace.MarketplaceInterface) *Reconciler {
	return &Reconciler{
		mpm: mpm,
	}
}

func (r *Reconciler) ReconcileOauthClient(ctx context.Context, inst *integreatlyv1alpha1.RHMI, client *oauthv1.OAuthClient, apiClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Make sure to use the redirect URIs supplied to the reconcile function and
	// not those that are currently on the client. Copy the uris because arrays
	// are references.
	redirectUris := make([]string, len(client.RedirectURIs))
	copy(redirectUris, client.RedirectURIs)

	// Preserve secret and grant method too
	secret := client.Secret
	grantMethod := client.GrantMethod

	if err := apiClient.Get(ctx, k8sclient.ObjectKey{Name: client.Name}, client); err != nil {
		if k8serr.IsNotFound(err) {
			PrepareObjectLabels(client, inst, true, false, false)

			if err := apiClient.Create(ctx, client); err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create oauth client: %s. %w", client.Name, err)
			}
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get oauth client: %s. %w", client.Name, err)
	}

	PrepareObjectLabels(client, inst, true, false, false)

	client.RedirectURIs = redirectUris
	client.GrantMethod = grantMethod
	client.Secret = secret

	if err := apiClient.Update(ctx, client); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update oauth client: %s. %w", client.Name, err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// GetNS gets the specified corev1.Namespace from the k8s API server
func GetNS(ctx context.Context, namespace string, client k8sclient.Client) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: namespace}, ns)
	if err == nil {
		// workaround for https://github.com/kubernetes/client-go/issues/541
		ns.TypeMeta = metav1.TypeMeta{Kind: "Namespace", APIVersion: metav1.SchemeGroupVersion.Version}
	}
	return ns, err
}

func CreateNSWithProjectRequest(ctx context.Context, namespace string, client k8sclient.Client, inst *integreatlyv1alpha1.RHMI, addRHMIMonitoringLabels, addClusterMonitoringLabel, disableUserAlerting bool) (*corev1.Namespace, error) {
	projectRequest := &projectv1.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	if err := client.Create(ctx, projectRequest); err != nil {
		return nil, fmt.Errorf("could not create %s Namespace: %v", projectRequest.Name, err)
	}

	// when a namespace is created using the ProjectRequest object it drops labels and annotations
	// so we need to retrieve the project as namespace and add them
	ns, err := GetNS(ctx, namespace, client)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve %s namespace: %v", ns.Name, err)
	}

	PrepareObjectLabels(ns, inst, addRHMIMonitoringLabels, addClusterMonitoringLabel, disableUserAlerting)

	if err := client.Update(ctx, ns); err != nil {
		return nil, fmt.Errorf("failed to update the %s namespace definition: %v", ns.Name, err)
	}

	return ns, err
}

func (r *Reconciler) ReconcileNamespace(ctx context.Context, namespace string, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client, log l.Logger) (integreatlyv1alpha1.StatusPhase, error) {
	ns, err := GetNS(ctx, namespace, client)
	if err != nil {
		// Since we are using ProjectRequests and limited permissions,
		// request can return "forbidden" error even when Namespace simply doesn't exist yet
		if !k8serr.IsNotFound(err) && !k8serr.IsForbidden(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve namespace: %s. %w", namespace, err)
		}

		_, err = CreateNSWithProjectRequest(ctx, namespace, client, inst, true, false, true)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create %s namespace: %v", namespace, err)
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	if inst.Spec.PullSecret.Name != "" {
		_, err := r.ReconcilePullSecret(ctx, namespace, inst.Spec.PullSecret.Name, inst, client)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to reconcile %s pull secret", inst.Spec.PullSecret.Name)
		}
	}

	PrepareObjectLabels(ns, inst, true, false, true)

	if err := client.Update(ctx, ns); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update the ns definition: %w", err)
	}

	if ns.Status.Phase == corev1.NamespaceTerminating {
		log.Debugf("namespace terminating, maintaining phase to try again on next reconcile", l.Fields{"ns": namespace})
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if ns.Status.Phase != corev1.NamespaceActive {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

type finalizerFunc func() (integreatlyv1alpha1.StatusPhase, error)

func (r *Reconciler) ReconcileFinalizer(ctx context.Context, client k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productName string, uninstall bool, finalFunc finalizerFunc, log l.Logger) (integreatlyv1alpha1.StatusPhase, error) {
	finalizer := productName + ".integreatly.org" + "/finalizer"

	// Run finalization logic. If it fails, don't remove the finalizer
	// so that we can retry during the next reconciliation
	if uninstall {
		if Contains(inst.GetFinalizers(), finalizer) {
			phase, err := finalFunc()
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			// Remove the finalizer to allow for deletion of the installation cr
			log.Infof("Removing finalizer", l.Fields{"finalizer": finalizer})
			inst.SetFinalizers(Remove(inst.GetFinalizers(), finalizer))
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	// Replace finalizers with the new format finalizers
	err := UpdateFinalizer(ctx, inst, client, productName, finalizer, log)
	if err != nil {
		log.Error(fmt.Sprintf("Error replacing finalizer %s in installation", finalizer), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Add finalizer if not there
	err = AddFinalizer(ctx, inst, client, finalizer, log)
	if err != nil {
		log.Error(fmt.Sprintf("Error adding finalizer %s to installation", finalizer), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcilePullSecret(ctx context.Context, destSecretNamespace, destSecretName string, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	err := CopySecret(ctx, client, inst.Spec.PullSecret.Name, inst.Spec.PullSecret.Namespace, destSecretName, destSecretNamespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating/updating secret '%s' in namespace: '%s': %w", destSecretName, destSecretNamespace, err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcileSubscription(ctx context.Context, target marketplace.Target, operandNS []string, preUpgradeBackupExecutor backup.BackupExecutor, client k8sclient.Client, catalogSourceReconciler marketplace.CatalogSourceReconciler, log l.Logger) (integreatlyv1alpha1.StatusPhase, error) {
	log.Infof("Reconciling subscription", l.Fields{"subscription": target.SubscriptionName, "channel": marketplace.IntegreatlyChannel, "ns": target.Namespace})
	err := r.mpm.InstallOperator(ctx, client, target, operandNS, operatorsv1alpha1.ApprovalManual, catalogSourceReconciler)

	if err != nil && !k8serr.IsAlreadyExists(err) {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create subscription in namespace: %s: %w", target.Namespace, err)
	}
	ip, sub, err := r.mpm.GetSubscriptionInstallPlan(ctx, client, target.SubscriptionName, target.Namespace)
	if err != nil {
		// this could be the install plan or subscription so need to check if sub nil or not TODO refactor
		if k8serr.IsNotFound(err) || k8serr.IsNotFound(errors.Unwrap(err)) {
			return integreatlyv1alpha1.PhaseAwaitingOperator, nil
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve installplan and subscription in namespace: %s: %w", target.Namespace, err)
	}

	if ip == nil {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	err = upgradeApproval(ctx, preUpgradeBackupExecutor, client, ip, log)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error approving installplan for %v: %w", target.SubscriptionName, err)
	}

	// Workaround to re-install product operator if install plan fails due to https://bugzilla.redhat.com/show_bug.cgi?id=1923111
	if ip.Status.Phase == operatorsv1alpha1.InstallPlanPhaseFailed {
		var csv *operatorsv1alpha1.ClusterServiceVersion
		if sub.Status.InstalledCSV != "" {
			csv = &operatorsv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{Namespace: target.Namespace, Name: sub.Status.InstalledCSV},
			}
		}
		return retryInstallation(ctx, client, log, target, csv, sub)
	}

	//if it's approved but not complete, then it's in progress
	if ip.Status.Phase != operatorsv1alpha1.InstallPlanPhaseComplete && ip.Spec.Approved {
		log.Infof("Install plan is not complete yet ", l.Fields{"install plan": target.SubscriptionName})
		return integreatlyv1alpha1.PhaseInProgress, nil
		//if it's not approved by now, then it will not be approved by this version of the integreatly-operator
	} else if !ip.Spec.Approved {
		log.Infof("Upgrade installplan above the maximum allowed version", l.Fields{"install plan": target.SubscriptionName})
	}

	for _, csvName := range ip.Spec.ClusterServiceVersionNames {
		ipCSV := &operatorsv1alpha1.ClusterServiceVersion{}
		if err := client.Get(ctx, k8sclient.ObjectKey{
			Name:      csvName,
			Namespace: target.Namespace,
		}, ipCSV); err != nil {
			if k8serr.IsNotFound(err) {
				log.Infof("Waiting for CSV to be created in cluster after InstallPlan is complete: %s", l.Fields{
					"install plan": target.SubscriptionName,
				})
				return integreatlyv1alpha1.PhaseInProgress, nil
			}
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error retrieving CSV: %v", err)
		}

		if err := validateCSV(ipCSV); err != nil {
			log.Warningf("CSV failed validation. Retrying operator installation", l.Fields{"error": err, "install plan": target.SubscriptionName})
			return retryInstallation(ctx, client, log, target, ipCSV, sub)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) WithProductDeclaration(productDeclaration marketplace.ProductDeclaration) *Reconciler {
	r.productDeclaration = &productDeclaration
	return r
}

func (r *Reconciler) GetProductDeclaration() *marketplace.ProductDeclaration {
	return r.productDeclaration
}

func validateCSV(csv *operatorsv1alpha1.ClusterServiceVersion) error {
	if csv.Spec.InstallStrategy.StrategyName == operatorsv1alpha1.InstallStrategyNameDeployment && len(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs) == 0 {
		return errors.New("no Deployment found in install strategy")
	}

	return nil
}

func retryInstallation(ctx context.Context, client k8sclient.Client, log l.Logger, target marketplace.Target, csv *operatorsv1alpha1.ClusterServiceVersion, sub *operatorsv1alpha1.Subscription) (integreatlyv1alpha1.StatusPhase, error) {
	if csv != nil {
		log.Warningf("Deleting csv for re-install due to failed install plan", l.Fields{"ns": target.Namespace, "install plan": target.SubscriptionName, "csv": csv.Name})
		if err := client.Delete(ctx, csv); err != nil && !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete csv for re-install: %s", err)
		}
	}

	log.Warningf("Deleting subscription for re-install due to failed install plan", l.Fields{"ns": target.Namespace, "install plan": target.SubscriptionName})
	if err := client.Delete(ctx, sub); err != nil && !k8serr.IsNotFound(err) {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete subscription for re-intstall: %s", err)
	}
	return integreatlyv1alpha1.PhaseAwaitingOperator, nil
}

func PrepareObjectLabels(object metav1.Object, install *integreatlyv1alpha1.RHMI, addRHMIMonitoringLabels, addClusterMonitoringLabel, disableUserAlerting bool) {
	labels := object.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if disableUserAlerting {
		labels["openshift.io/user-monitoring"] = "false"
	}
	if addRHMIMonitoringLabels {
		labels["monitoring-key"] = "middleware"
	} else {
		delete(labels, "monitoring-key")
	}
	if addClusterMonitoringLabel {
		labels["openshift.io/cluster-monitoring"] = "true"
	} else {
		delete(labels, "openshift.io/cluster-monitoring")
	}
	labels["integreatly"] = "true"
	labels[OwnerLabelKey] = string(install.GetUID())
	object.SetLabels(labels)
}

func IsOwnedBy(o metav1.Object, owner *integreatlyv1alpha1.RHMI) bool {
	// TODO change logic to check for our finalizer?
	for k, v := range o.GetLabels() {
		if k == OwnerLabelKey && v == string(owner.UID) {
			return true
		}
	}
	return false
}

func (r *Reconciler) ReconcileCsvDeploymentsPriority(ctx context.Context, client k8sclient.Client, csvName, csvNamespace, priorityClassName string) (integreatlyv1alpha1.StatusPhase, error) {
	csv := &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      csvName,
			Namespace: csvNamespace,
		},
	}
	mutateFn := func() error {
		deployments := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
		for i := range deployments {
			deployments[i].Spec.Template.Spec.PriorityClassName = priorityClassName
		}
		return nil
	}
	return k8s.PatchIfExists(ctx, client, mutateFn, csv)
}
