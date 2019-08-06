package rhsso

import (
	"context"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	oauthv1 "github.com/openshift/api/oauth/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultRhssoNamespace   = "rhsso"
	customerAdminPassword   = "Password1"
	keycloakName            = "rhsso"
	keycloakRealmName       = "openshift"
	oauthId                 = "rhsso"
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
		ClientRoles: map[string][]string{
			"account": {
				"manage-account",
				"view-profile",
			},
			"realm-management": {
				"manage-users",
				"manage-identity-providers",
				"view-realm",
			},
		},
	},
	Password:     &customerAdminPassword,
	OutputSecret: "customer-admin-user-credentials",
}

type Reconciler struct {
	Config        *config.RHSSO
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	installation  *v1alpha1.Installation
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	rhssoConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, err
	}
	if rhssoConfig.GetNamespace() == "" {
		rhssoConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultRhssoNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:        rhssoConfig,
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  instance,
		logger:        logger,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

// Reconcile reads that state of the cluster for rhsso and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
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

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	kc := &aerogearv1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: aerogearv1.KeycloakSpec{
			AdminCredentials: "",
			Plugins: []string{
				"keycloak-metrics-spi",
				"openshift4-idp",
			},
			Backups:   []aerogearv1.KeycloakBackup{},
			Provision: true,
		},
	}
	ownerutil.EnsureOwner(kc, inst)
	err := serverClient.Create(ctx, kc)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create keycloak custom resource")
	}

	r.logger.Info("creating Keycloakrealm")
	kcr := &aerogearv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: aerogearv1.KeycloakRealmSpec{
			CreateOnly:                        true,
			BrowserRedirectorIdentityProvider: idpAlias,
			KeycloakApiRealm: &aerogearv1.KeycloakApiRealm{
				ID:          keycloakRealmName,
				Realm:       keycloakRealmName,
				DisplayName: keycloakRealmName,
				Enabled:     true,
				EventsListeners: []string{
					"metrics-listener",
				},
				Users: []*aerogearv1.KeycloakUser{
					CustomerAdminUser,
				},
			},
		},
	}
	ownerutil.EnsureOwner(kcr, inst)
	err = serverClient.Create(ctx, kcr)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create keycloak realm")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("checking ready status for rhsso")
	kcr := &aerogearv1.KeycloakRealm{}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get keycloak realm custom resource")
	}

	if kcr.Status.Phase == aerogearv1.PhaseReconcile {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to write rhsso config")
		}

		err = r.setupOpenshiftIDP(ctx, kcr, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to setup Openshift IDP")
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm")
		return v1alpha1.PhaseCompleted, nil
	}

	r.logger.Infof("KeycloakRealm status phase is: %s", kcr.Status.Phase)
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
		return pkgerr.Wrap(err, "could not retrieve keycloak custom resource for keycloak config")
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
		return pkgerr.Wrap(err, "could not retrieve keycloak admin credential secret for keycloak config")
	}
	kcURLBytes := kcAdminCredSecret.Data["SSO_ADMIN_URL"]
	r.Config.SetRealm(keycloakRealmName)
	r.Config.SetHost(string(kcURLBytes))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return pkgerr.Wrap(err, "could not update keycloak config")
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, kcr *aerogearv1.KeycloakRealm, serverClient pkgclient.Client) error {
	_, err := r.oauthv1Client.OAuthClients().Get(oauthId, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		oauthClient := &oauthv1.OAuthClient{
			ObjectMeta: metav1.ObjectMeta{
				Name: oauthId,
			},
			Secret: clientSecret,
			RedirectURIs: []string{
				r.Config.GetHost() + "/auth/realms/openshift/broker/openshift-v4/endpoint",
			},
			GrantMethod: oauthv1.GrantHandlerPrompt,
		}
		_, err = r.oauthv1Client.OAuthClients().Create(oauthClient)
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
			TrustEmail:                false,
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
