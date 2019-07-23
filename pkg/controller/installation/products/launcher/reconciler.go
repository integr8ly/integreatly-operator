package launcher

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "launcher"
	defaultSubscriptionName="launcher"
)

type Reconciler struct {
	coreClient    kubernetes.Interface
	Config        *config.Launcher
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
}

func NewReconciler(coreClient *kubernetes.Clientset, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadLauncher()
	if err != nil {
		return nil, err
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		coreClient:    coreClient,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
	}, nil
}

func (r *Reconciler) Reconcile(inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ctx := context.TODO()

	phase, err := r.reconcileNamespace(ctx, inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, " failed to reconcile namespace for launcher ")
	}

	phase, err = r.reconcileSubscription(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, " failed to reconcile subscription for launcher ")
	}

	phase, err = r.reconcileGithubOauth(ctx, inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, " failed to reconcile secret for launcher github oauth")
	}

	phase, err = r.reconcileCustomResource(ctx, inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, " failed to reconcile custom resource for launcher")
	}

	phase, err = r.reconcileOauthClient(ctx, inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, " failed to reconcile oauthclient for launcher")
	}

	r.logger.Debug("End of reconcile Phase: ", phase)

	// if we get to the end and no phase set then the reconcile is completed
	if phase == v1alpha1.PhaseNone {
		return v1alpha1.PhaseCompleted, nil
	}

	return phase, nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	nsr := resources.NewNamespaceReconciler(serverClient, r.logger)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}

	// Reconcile namespace
	ns, err := nsr.Reconcile(ctx, ns, inst)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "failed to reconcile launcher namespace "+r.Config.GetNamespace())
	}

	if ns.Status.Phase == v1.NamespaceTerminating {
		r.logger.Debugf("namespace %s is terminating, maintaining phase to try again on next reconcile", r.Config.GetNamespace())
		return v1alpha1.PhaseAwaitingNS, nil
	}

	if ns.Status.Phase != v1.NamespaceActive {
		return v1alpha1.PhaseAwaitingNS, nil
	}

	// all good return no status when ready
	r.logger.Debug("namespace is ready")
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Debugf("creating subscription %s from channel %s in namespace: %s", defaultSubscriptionName, "integreatly", r.Config.GetNamespace())
	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		defaultSubscriptionName,
		marketplace.IntegreatlyChannel,
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not create subscription in namespace: %s", r.Config.GetNamespace()))
	}

	//return v1alpha1.PhaseAwaitingOperator, nil
	return r.handleAwaitingOperator(ctx, serverClient)
}

func (r *Reconciler) reconcileGithubOauth(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Debugf("reconciling secret for launcher github oauth in namespace: %s", r.Config.GetNamespace())

	githubOauthClientId := inst.Spec.GithubOauthClientId
	githubOauthClientSecret := inst.Spec.GithubOauthClientSecret

	// Set placeholder values if client id and secret are not provided
	if githubOauthClientId == "" {
		githubOauthClientId = "github-oauth-client-id"
	}

	if githubOauthClientSecret == "" {
		githubOauthClientSecret = "github-oauth-client-secret"
	}

	launcherGithubOauthSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "launcher-oauth-github",
			Namespace: r.Config.GetNamespace(),
		},
		StringData: map[string]string{
			"clientId": githubOauthClientId,
			"secret": githubOauthClientSecret,
		},
	}

	err := serverClient.Create(ctx, launcherGithubOauthSecret)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("failed to retrieve %s secret in namespace %s", launcherGithubOauthSecret.Name, r.Config.GetNamespace()))
		}

		if err = serverClient.Update(ctx, launcherGithubOauthSecret); err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("failed to update %s secret in namespace %s", launcherGithubOauthSecret.Name, r.Config.GetNamespace()))
		}
	}

	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileCustomResource(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// TODO Case: Create Custom Resource https://gist.github.com/JameelB/ab711ed80e147078e816aaf895ba00b4
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileOauthClient(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// TODO Case: OauthClient (as per https://github.com/fabric8-launcher/launcher-operator#install-the-launcher-via-the-installed-operator)
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) handleAwaitingOperator(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("checking installplan is created for subscription %s in namespace: %s", defaultSubscriptionName, r.Config.GetNamespace())
	ip, _, err := r.mpm.GetSubscriptionInstallPlan(defaultSubscriptionName, r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			r.logger.Debugf(fmt.Sprintf("installplan resource is not found in namespace: %s", r.Config.GetNamespace()))
			return v1alpha1.PhaseAwaitingOperator, nil
		}
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not retrieve installplan in namespace: %s", r.Config.GetNamespace()))
	}

	r.logger.Infof("installplan phase is %s", ip.Status.Phase)
	if ip.Status.Phase != coreosv1alpha1.InstallPlanPhaseComplete {
		r.logger.Infof("launcher install plan is not complete yet")
		return v1alpha1.PhaseAwaitingOperator, nil
	}

	r.logger.Infof("launcher install plan is complete. Installation ready.")
	return v1alpha1.PhaseNone, nil
}