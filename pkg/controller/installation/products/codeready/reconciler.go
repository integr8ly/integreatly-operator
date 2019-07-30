package codeready

import (
	"context"
	"fmt"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	keycloakv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
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

type Reconciler struct {
	Config         *config.CodeReady
	KeycloakConfig *config.RHSSO
	ConfigManager  config.ConfigReadWriter
	mpm            marketplace.MarketplaceInterface
	logger         *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadCodeReady()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve che config")
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	kcConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve keycloak config")
	}
	if err = kcConfig.Validate(); err != nil {
		return nil, errors.Wrap(err, "keycloak config is not valid")
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager:  configManager,
		Config:         config,
		KeycloakConfig: kcConfig,
		mpm:            mpm,
		logger:         logger,
		Reconciler:     resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, inst, defaultSubscriptionName, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileCheCluster(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	phase, err = r.reconcileKeycloakClient(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBackups(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBackups(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace:     r.Config.GetNamespace(),
		Name:          "codeready",
		BackendSecret: resources.BackupSecretLocation{Name: r.Config.GetBackendSecretName(), Namespace: r.ConfigManager.GetOperatorNamespace()},
		Components: []resources.BackupComponent{
			{
				Name:     "codeready-postgres-backup",
				Type:     "postgres",
				Secret:   resources.BackupSecretLocation{Name: r.Config.GetPostgresBackupSecretName(), Namespace: r.Config.GetNamespace()},
				Schedule: r.Config.GetBackupSchedule(),
			},
			{
				Name:     "codeready-pv-backup",
				Type:     "codeready_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}
	err := r.reconcilePostgresSecret(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to reconcile postgres component backup secret")
	}
	err = resources.ReconcileBackup(ctx, serverClient, inst, backupConfig)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create backups for codeready")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilePostgresSecret(ctx context.Context, serverClient pkgclient.Client) error {
	//get values from deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "postgres",
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Namespace: deployment.Namespace, Name: deployment.Name}, deployment)
	if err != nil {
		return errors.Wrapf(err, "could not get postgres deployment to reconcile component backup secret")
	}

	postgresqlSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.GetPostgresBackupSecretName(),
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{
			"POSTGRES_HOST": []byte("postgres." + r.Config.GetNamespace() + ".svc"),
		},
	}

	for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
		switch env.Name {
		case "POSTGRESQL_USER":
			postgresqlSecret.Data["POSTGRES_USERNAME"] = []byte(env.Value)
		case "POSTGRESQL_PASSWORD":
			postgresqlSecret.Data["POSTGRES_PASSWORD"] = []byte(env.Value)
		case "POSTGRESQL_DATABASE":
			postgresqlSecret.Data["POSTGRES_DATABASE"] = []byte(env.Value)
		}
	}

	err = resources.CreateOrUpdate(ctx, serverClient, postgresqlSecret)
	if err != nil {
		return errors.Wrapf(err, "error reconciling postgres component backup secret")
	}
	logrus.Infof("codeready postgres component backup secret successfully reconciled")

	return nil
}

func (r *Reconciler) reconcileCheCluster(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("creating required custom resources in namespace: %s", r.Config.GetNamespace())

	kcRealm := &keycloakv1.KeycloakRealm{}
	key := pkgclient.ObjectKey{Name: r.KeycloakConfig.GetRealm(), Namespace: r.KeycloakConfig.GetNamespace()}
	err := serverClient.Get(ctx, key, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve: %+v", key))
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
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve checluster custom resource in namespace: %s", r.Config.GetNamespace()))
		}
		if err = r.createCheCluster(ctx, r.KeycloakConfig, kcRealm, inst, serverClient); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create checluster custom resource in namespace: %s", r.Config.GetNamespace()))
		}
		return v1alpha1.PhaseInProgress, err
	}

	// check cr values
	if cheCluster.Spec.Auth.ExternalKeycloak &&
		!cheCluster.Spec.Auth.OpenShiftOauth &&
		cheCluster.Spec.Auth.KeycloakURL == r.KeycloakConfig.GetURL() &&
		cheCluster.Spec.Auth.KeycloakRealm == r.KeycloakConfig.GetRealm() &&
		cheCluster.Spec.Auth.KeycloakClientId == defaultClientName {
		logrus.Debug("skipping checluster custom resource update as all values are correct")
		return v1alpha1.PhaseCompleted, nil
	}

	// update cr values
	cheCluster.Spec.Auth.ExternalKeycloak = true
	cheCluster.Spec.Auth.OpenShiftOauth = false
	cheCluster.Spec.Auth.KeycloakURL = r.KeycloakConfig.GetURL()
	cheCluster.Spec.Auth.KeycloakRealm = kcRealm.Name
	cheCluster.Spec.Auth.KeycloakClientId = defaultClientName
	if err = serverClient.Update(ctx, cheCluster); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not update checluster custom resource in namespace: %s", r.Config.GetNamespace()))
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("checking that checluster custom resource is marked as available")

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve checluster for keycloak client update")
	}
	if cheCluster.Status.CheClusterRunning != "Available" {
		return v1alpha1.PhaseInProgress, nil
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKeycloakClient(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("checking keycloak client exists in keycloakrealm custom resource in namespace: %s", r.Config.GetNamespace())

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve checluster for keycloak client update")
	}

	cheURL := cheCluster.Status.CheURL
	if cheURL == "" {
		//still waiting for the Che URL, so exit codeready reconciling now and try again
		return v1alpha1.PhaseInProgress, nil
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
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve keycloakrealm for keycloak client update")
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
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not update keycloakrealm custom resource with codeready client")
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
	ownerutil.EnsureOwner(cheCluster, inst)
	return serverClient.Create(ctx, cheCluster)
}
