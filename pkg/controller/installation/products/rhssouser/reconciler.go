package rhssouser

import (
	"context"

	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultRhssoNamespace   = "user-sso"
	customerAdminPassword   = "Password1"
	keycloakName            = "rhssouser"
	keycloakRealmName       = "user-sso"
	oauthId                 = "rhssouser"
	clientSecret            = "placeholder" // this should be replaced in INTLY-2784
	defaultSubscriptionName = "integreatly-rhsso"
	idpAlias                = "openshift-v4"
)

var CustomerAdminUser = &aerogearv1.KeycloakUser{
	KeycloakApiUser: &aerogearv1.KeycloakApiUser{
		Enabled:       true,
		Attributes:    aerogearv1.KeycloakAttributes{},
		UserName:      "customer-admin",
		EmailVerified: true,
		Email:         "customer-admin@example.com",
		ClientRoles:   getKeycloakRoles(true),
	},
	Password:     &customerAdminPassword,
	OutputSecret: "customer-admin-user-credentials",
}

type Reconciler struct {
	Config        *config.RHSSOUser
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	installation  *v1alpha1.Installation
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	rhssoUserConfig, err := configManager.ReadRHSSOUser()
	if err != nil {
		return nil, err
	}
	if rhssoUserConfig.GetNamespace() == "" {
		rhssoUserConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultRhssoNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:        rhssoUserConfig,
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  instance,
		logger:        logger,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sso",
			Namespace: ns,
		},
	}
}

// Reconcile reads that state of the cluster for rhsso and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()

	phase, err := r.ReconcileNamespace(ctx, ns, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling Keycloak components")
	kc := &aerogearv1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	ownerutil.EnsureOwner(kc, inst)
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func(existing runtime.Object) error {
		kc := existing.(*aerogearv1.Keycloak)
		kc.Spec.Plugins = []string{
			"keycloak-metrics-spi",
			"openshift4-idp",
		}
		kc.Spec.Provision = true
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update keycloak custom resource")
	}
	r.logger.Infof("The operation result for keycloak %s was %s", kc.Name, or)

	kcr := &aerogearv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	ownerutil.EnsureOwner(kcr, inst)
	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func(existing runtime.Object) error {
		kcr := existing.(*aerogearv1.KeycloakRealm)
		kcr.Spec.CreateOnly = false
		kcr.Spec.BrowserRedirectorIdentityProvider = idpAlias

		if kcr.Spec.KeycloakApiRealm == nil {
			kcr.Spec.KeycloakApiRealm = &aerogearv1.KeycloakApiRealm{}
		}
		kcr.Spec.ID = keycloakRealmName
		kcr.Spec.Realm = keycloakRealmName
		kcr.Spec.DisplayName = keycloakRealmName
		kcr.Spec.Enabled = true
		kcr.Spec.EventsListeners = []string{
			"metrics-listener",
		}

		if kcr.Spec.Users == nil {
			kcr.Spec.Users = []*aerogearv1.KeycloakUser{CustomerAdminUser}
		}
		users, err := syncronizeWithOpenshiftUsers(kcr.Spec.Users, ctx, serverClient)
		if err != nil {
			return err
		}
		kcr.Spec.Users = users

		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update keycloak realm")
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	kc := &aerogearv1.Keycloak{}
	// if this errors, it can be ignored
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err == nil && string(r.Config.GetProductVersion()) != kc.Status.Version {
		r.Config.SetProductVersion(kc.Status.Version)
		r.ConfigManager.WriteConfig(r.Config)
	}

	r.logger.Info("checking ready status for user-sso")
	kcr := &aerogearv1.KeycloakRealm{}

	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get keycloak realm custom resource")
	}

	if kcr.Status.Phase == aerogearv1.PhaseReconcile {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to write user-sso config")
		}

		err = r.setupOpenshiftIDP(ctx, kcr, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to setup Openshift IDP for user-sso")
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm for user-sso")
		return v1alpha1.PhaseCompleted, nil
	}

	r.logger.Infof("user-sso KeycloakRealm status phase is: %s", kcr.Status.Phase)
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) exportConfig(ctx context.Context, serverClient pkgclient.Client) error {
	kc := &aerogearv1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return pkgerr.Wrap(err, "could not retrieve keycloak custom resource for keycloak config for user-sso")
	}
	kcAdminCredSecretName := kc.Spec.AdminCredentials

	kcAdminCredSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kcAdminCredSecretName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: kcAdminCredSecretName, Namespace: r.Config.GetNamespace()}, kcAdminCredSecret)
	if err != nil {
		return pkgerr.Wrap(err, "could not retrieve keycloak admin credential secret for keycloak config for user-sso")
	}
	kcURLBytes := kcAdminCredSecret.Data["SSO_ADMIN_URL"]
	r.Config.SetRealm(keycloakRealmName)
	r.Config.SetHost(string(kcURLBytes))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return pkgerr.Wrap(err, "could not update keycloak config for user-sso")
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, kcr *aerogearv1.KeycloakRealm, serverClient pkgclient.Client) error {
	_, err := r.oauthv1Client.OAuthClients().Get(oauthId, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		oauthc := &oauthv1.OAuthClient{
			ObjectMeta: metav1.ObjectMeta{
				Name: oauthId,
			},
			Secret: clientSecret,
			RedirectURIs: []string{
				r.Config.GetHost() + "/auth/realms/openshift/broker/openshift-v4/endpoint",
			},
			GrantMethod: oauthv1.GrantHandlerPrompt,
		}
		_, err = r.oauthv1Client.OAuthClients().Create(oauthc)
		if err != nil {
			return pkgerr.Wrap(err, "Could not create OauthClient object for OpenShift IDP")
		}
	}

	if !containsIdentityProvider(kcr.Spec.IdentityProviders, idpAlias) {
		logrus.Infof("Adding keycloak realm client")

		kcr.Spec.IdentityProviders = append(kcr.Spec.IdentityProviders, &aerogearv1.KeycloakIdentityProvider{
			Alias:                     idpAlias,
			ProviderID:                "openshift-v4",
			Enabled:                   true,
			TrustEmail:                true,
			StoreToken:                true,
			AddReadTokenRoleOnCreate:  true,
			FirstBrokerLoginFlowAlias: "first broker login",
			Config: map[string]string{
				"hideOnLoginPage": "",
				"baseUrl":         "https://openshift.default.svc.cluster.local",
				"clientId":        oauthId,
				"disableUserInfo": "",
				"clientSecret":    clientSecret,
				"defaultScope":    "user:full",
				"useJwksUrl":      "true",
			},
		})

		return serverClient.Update(ctx, kcr)
	}
	return nil
}

func containsIdentityProvider(providers []*aerogearv1.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

func syncronizeWithOpenshiftUsers(keycloakUsers []*aerogearv1.KeycloakUser, ctx context.Context, serverClient pkgclient.Client) ([]*aerogearv1.KeycloakUser, error) {
	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, &pkgclient.ListOptions{}, openshiftUsers)
	if err != nil {
		return nil, err
	}

	openshiftAdminGroup := &usersv1.Group{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: "dedicated-admins"}, openshiftAdminGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return nil, err
	}
	for _, kcUser := range keycloakUsers {
		if kcUser.UserName == CustomerAdminUser.UserName {
			continue
		}

		kcUser.ClientRoles = getKeycloakRoles(isOpenshiftAdmin(kcUser, openshiftAdminGroup))
	}

	return keycloakUsers, nil
}

func isOpenshiftAdmin(kcUser *aerogearv1.KeycloakUser, adminGroup *usersv1.Group) bool {
	for _, name := range adminGroup.Users {
		if kcUser.UserName == name {
			return true
		}
	}

	return false
}

func getKeycloakRoles(isAdmin bool) map[string][]string {
	roles := map[string][]string{
		"account": {
			"manage-account",
			"view-profile",
		},
		"broker": {
			"read-token",
		},
	}
	if isAdmin {
		roles["realm-management"] = []string{
			"manage-users",
			"manage-identity-providers",
			"view-realm",
		}
	}

	return roles
}
