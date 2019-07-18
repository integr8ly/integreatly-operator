package codeready

import (
	"context"
	"errors"
	"fmt"
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	keycloakv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
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
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "codeready-workspaces"
	defaultClientName            = "che-client"
	defaultCheClusterName        = "integreatly-cluster"
	defaultSubscriptionName      = "codeready-workspaces"
)

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadCodeReady()
	if err != nil {
		return nil, pkgerr.Wrap(err, "could not retrieve che config")
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	kcConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, pkgerr.Wrap(err, "could not retrieve keycloak config")
	}
	if err = kcConfig.Validate(); err != nil {
		return nil, pkgerr.Wrap(err, "keycloak config is not valid")
	}

	return &Reconciler{
		ConfigManager:  configManager,
		Config:         config,
		KeycloakConfig: kcConfig,
		mpm:            mpm,
	}, nil
}

type Reconciler struct {
	Config         *config.CodeReady
	KeycloakConfig *config.RHSSO
	ConfigManager  config.ConfigReadWriter
	mpm            marketplace.MarketplaceInterface
}

func (r *Reconciler) Reconcile(inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling codeready phase: %s", inst.Status.ProductStatus[r.Config.GetProductName()])
	ctx := context.TODO()
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	switch v1alpha1.StatusPhase(phase) {
	case v1alpha1.PhaseNone, v1alpha1.PhaseAwaitingNS:
		return r.reconcileNamespace(ctx, inst, serverClient)
	case v1alpha1.PhaseCreatingSubscription:
		return r.reconcileSubscription(ctx)
	case v1alpha1.PhaseCreatingComponents:
		return r.reconcileCheCluster(ctx, inst, serverClient)
	case v1alpha1.PhaseAwaitingOperator:
		return r.handleAwaitingOperator(ctx)
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase(ctx, serverClient)
	case v1alpha1.PhaseCompleted, v1alpha1.PhaseFailed:
		return r.handleReconcile(ctx, inst, serverClient)
	default:
		return r.handleReconcile(ctx, inst, serverClient)
	}
}

func (r *Reconciler) handleReconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.reconcileNamespace(ctx, inst, serverClient)
	if err != nil {
		return phase, pkgerr.Wrap(err, "could not reconcile namespace")
	}
	phase, err = r.reconcileCheCluster(ctx, inst, serverClient)
	if err != nil {
		return phase, pkgerr.Wrap(err, "could not reconcile checluster")
	}
	phase, err = r.reconcileSubscription(ctx)
	if err != nil {
		return phase, pkgerr.Wrap(err, "could not reconcile subscription")
	}
	phase, err = r.reconcileKeycloakClient(ctx, serverClient)
	if err != nil {
		return phase, pkgerr.Wrap(err, "could not reconcile keycloakrealm")
	}
	phase, err = r.reconcileBackups(ctx, inst, serverClient)
	if err != nil {
		return phase, pkgerr.Wrap(err, "could not reconcile backups")
	}
	return phase, nil
}

func (r *Reconciler) reconcileBackups(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace:  r.Config.GetNamespace(),
		Name:       "codeready-backups",
		Components: []string{"postgres", "codeready_pv"},
	}
	err := resources.ReconcileBackup(ctx, serverClient, inst, backupConfig)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed to create backups for codeready")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	nsr := resources.NewNamespaceReconciler(serverClient)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}
	ns, err := nsr.Reconcile(ctx, ns, inst)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseCreatingSubscription, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context) (v1alpha1.StatusPhase, error) {
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

	return v1alpha1.PhaseAwaitingOperator, nil
}

func (r *Reconciler) reconcileCheCluster(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Debugf("creating required custom resources in namespace: %s", r.Config.GetNamespace())
	kcRealm := &keycloakv1.KeycloakRealm{}
	key := pkgclient.ObjectKey{Name: r.KeycloakConfig.GetRealm(), Namespace: r.KeycloakConfig.GetNamespace()}
	err := serverClient.Get(ctx, key, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(
			err,
			fmt.Sprintf("could not retrieve: %+v", key),
		)
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not retrieve checluster custom resource in namespace: %s", r.Config.GetNamespace()))
		}
		if err = r.createCheCluster(ctx, r.KeycloakConfig, kcRealm, inst, serverClient); err != nil {
			if k8serr.IsAlreadyExists(err) {
				logrus.Debugf("checluster custom resource already exists in namespace: %s", r.Config.GetNamespace())
				return v1alpha1.PhaseInProgress, nil
			}
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not create checluster custom resource in namespace: %s", r.Config.GetNamespace()))
		}
		return v1alpha1.PhaseInProgress, err
	}
	if cheCluster.Spec.Auth.ExternalKeycloak &&
		!cheCluster.Spec.Auth.OpenShiftOauth &&
		cheCluster.Spec.Auth.KeycloakURL == r.KeycloakConfig.GetURL() &&
		cheCluster.Spec.Auth.KeycloakRealm == r.KeycloakConfig.GetRealm() &&
		cheCluster.Spec.Auth.KeycloakClientId == defaultClientName {
		logrus.Debug("skipping checluster custom resource update as all values are correct")
		return v1alpha1.PhaseInProgress, nil
	}
	cheCluster.Spec.Auth.ExternalKeycloak = true
	cheCluster.Spec.Auth.OpenShiftOauth = false
	cheCluster.Spec.Auth.KeycloakURL = r.KeycloakConfig.GetURL()
	cheCluster.Spec.Auth.KeycloakRealm = kcRealm.Name
	cheCluster.Spec.Auth.KeycloakClientId = defaultClientName
	if err = serverClient.Update(ctx, cheCluster); err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not update checluster custom resource in namespace: %s", r.Config.GetNamespace()))
	}
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleAwaitingOperator(ctx context.Context) (v1alpha1.StatusPhase, error) {
	logrus.Debugf("checking installplan is created for subscription %s in namespace: %s", defaultSubscriptionName, r.Config.GetNamespace())
	ip, _, err := r.mpm.GetSubscriptionInstallPlan(defaultSubscriptionName, r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			logrus.Debugf(fmt.Sprintf("installplan resource is not found in namespace: %s", r.Config.GetNamespace()))
			return v1alpha1.PhaseAwaitingOperator, nil
		}
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not retrieve installplan in namespace: %s", r.Config.GetNamespace()))
	}

	logrus.Debugf("installplan phase is %s, waiting for %s", ip.Status.Phase, coreosv1alpha1.InstallPlanPhaseComplete)
	if ip.Status.Phase != coreosv1alpha1.InstallPlanPhaseComplete {
		return v1alpha1.PhaseAwaitingOperator, nil
	}
	return v1alpha1.PhaseCreatingComponents, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Debugf("checking that checluster custom resource is marked as available")

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not retrieve checluster for keycloak client update")
	}
	if cheCluster.Status.CheClusterRunning != "Available" {
		return v1alpha1.PhaseInProgress, nil
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKeycloakClient(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Debugf("checking keycloak client exists in keycloakrealm custom resource in namespace: %s", r.Config.GetNamespace())

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not retrieve checluster for keycloak client update")
	}
	cheURL := cheCluster.Status.CheURL
	if cheURL == "" {
		return v1alpha1.PhaseFailed, errors.New("che URL is not set")
	}

	// retrieve the sso config so we can find the keycloakrealm custom resource to update
	kcRealm := &keycloakv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.KeycloakConfig.GetRealm(),
			Namespace: r.KeycloakConfig.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: r.KeycloakConfig.GetRealm(), Namespace: r.KeycloakConfig.GetNamespace()}, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not retrieve keycloakrealm for keycloak client update")
	}

	// Create a che client that can be used in keycloak for che to login with
	kcCheClient := &keycloakv1.KeycloakClient{}
	if existingKcCheClient := findCheKeycloakClient(kcRealm.Spec.Clients); existingKcCheClient != nil {
		kcCheClient = existingKcCheClient
	}
	kcCheClient.KeycloakApiClient = &keycloakv1.KeycloakApiClient{
		ID:                        defaultClientName,
		Name:                      defaultClientName,
		Enabled:                   true,
		StandardFlowEnabled:       true,
		DirectAccessGrantsEnabled: true,
		RootURL:                   cheURL,
		RedirectUris:              []string{cheURL, fmt.Sprintf("%s/*", cheURL)},
		AdminURL:                  cheURL,
		WebOrigins:                []string{cheURL, fmt.Sprintf("%s/*", cheURL)},
		PublicClient:              true,
		ClientAuthenticatorType:   "client-secret",
	}

	// Append the che client to the list of clients in the keycloak realm and save
	kcRealm.Spec.Clients = findOrAppendCheKeycloakClient(kcRealm.Spec.Clients, kcCheClient)
	err = serverClient.Update(ctx, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not update keycloakrealm custom resource with codeready client")
	}
	return v1alpha1.PhaseCompleted, nil
}

func findOrAppendCheKeycloakClient(clients []*keycloakv1.KeycloakClient, toAppend *keycloakv1.KeycloakClient) []*keycloakv1.KeycloakClient {
	if existingKcCheClient := findCheKeycloakClient(clients); existingKcCheClient != nil {
		return clients
	}
	return append(clients, toAppend)
}

func findCheKeycloakClient(clients []*keycloakv1.KeycloakClient) *keycloakv1.KeycloakClient {
	for _, client := range clients {
		if client.ID == defaultClientName {
			return client
		}
	}
	return nil
}

func (r *Reconciler) createCheCluster(ctx context.Context, kcCfg *config.RHSSO, kr *keycloakv1.KeycloakRealm, inst *v1alpha1.Installation, serverClient pkgclient.Client) error {
	selfSignedCerts := inst.Spec.SelfSignedCerts
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf(
				"%s/%s",
				chev1.SchemeGroupVersion.Group,
				chev1.SchemeGroupVersion.Version,
			),
			Kind: "CheCluster",
		},
		Spec: chev1.CheClusterSpec{
			Server: chev1.CheClusterSpecServer{
				CheFlavor:      "codeready",
				TlsSupport:     true,
				SelfSignedCert: selfSignedCerts,
			},
			Database: chev1.CheClusterSpecDB{
				ExternalDB:            false,
				ChePostgresDb:         "",
				ChePostgresPassword:   "",
				ChePostgresPort:       "",
				ChePostgresUser:       "",
				ChePostgresDBHostname: "",
			},
			Auth: chev1.CheClusterSpecAuth{
				OpenShiftOauth:   false,
				ExternalKeycloak: true,
				KeycloakURL:      kcCfg.GetURL(),
				KeycloakRealm:    kr.Name,
				KeycloakClientId: defaultClientName,
			},
			Storage: chev1.CheClusterSpecStorage{
				PvcStrategy:       "per-workspace",
				PvcClaimSize:      "1Gi",
				PreCreateSubPaths: true,
			},
		},
	}
	cheCluster.OwnerReferences = append(cheCluster.OwnerReferences, *metav1.NewControllerRef(inst, v1alpha1.SchemaGroupVersionKind))
	return serverClient.Create(ctx, cheCluster)
}
