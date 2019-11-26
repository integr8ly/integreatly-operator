package rhsso

import (
	"context"
	"fmt"
	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"k8s.io/apimachinery/pkg/labels"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	keycloak "github.com/integr8ly/integreatly-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/pkg/errors"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultRhssoNamespace               = "rhsso"
	customerAdminPassword               = "Password1"
	keycloakName                        = "rhsso"
	keycloakRealmName                   = "openshift"
	defaultSubscriptionName             = "keycloak-rhsso"
	idpAlias                            = "openshift-v4"
	githubIdpAlias                      = "github"
	githubOauthAppCredentialsSecretName = "github-oauth-secret"
)

const (
	SSOLabelKey   = "sso"
	SSOLabelValue = "integreatly"
	RHSSOProfile  = "RHSSO"
)

var CustomerAdminUser = keycloak.KeycloakAPIUser{
	ID:            "",
	UserName:      "customer-admin",
	EmailVerified: true,
	Enabled:       true,
	ClientRoles:   getKeycloakRoles(true),
	Email:         "customer-admin@example.com",
	Credentials: []keycloak.KeycloakCredential{
		{
			Type:      "password",
			Value:     customerAdminPassword,
			Temporary: false,
		},
	},
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

	phase, err := r.ReconcileFinalizer(ctx, serverClient, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, inst, serverClient, r.Config.GetNamespace())
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}

		err = resources.RemoveOauthClient(ctx, inst, serverClient, r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
		return v1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, ns, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	version, err := resources.NewVersion(v1alpha1.OperatorVersionRHSSO)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for rhsso")
	}
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, r.Config.GetNamespace(), serverClient, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling Keycloak components")
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func(existing runtime.Object) error {
		kc := existing.(*keycloak.Keycloak)
		kc.Spec.Extensions = []string{
			"keycloak-metrics-spi",
		}
		kc.Labels = GetInstanceLabels()
		kc.Spec.Instances = 1
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{
			Enabled: true,
		}
		kc.Spec.Profile = RHSSOProfile
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update keycloak custom resource")
	}
	r.logger.Infof("The operation result for keycloak %s was %s", kc.Name, or)

	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func(existing runtime.Object) error {
		kcr := existing.(*keycloak.KeycloakRealm)
		kcr.Spec.RealmOverrides = []*keycloak.RedirectorIdentityProviderOverride{
			{
				IdentityProvider: idpAlias,
				ForFlow:          "browser",
			},
		}

		kcr.Spec.InstanceSelector = &metav1.LabelSelector{
			MatchLabels: GetInstanceLabels(),
		}

		// The labels are needed so that created users can identify their realm
		// with a selector
		kcr.Labels = GetInstanceLabels()

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          keycloakRealmName,
			Realm:       keycloakRealmName,
			Enabled:     true,
			DisplayName: keycloakRealmName,
			EventsListeners: []string{
				"metrics-listener",
			},
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		err = r.setupOpenshiftIDP(ctx, inst, kcr, serverClient)
		if err != nil {
			return errors.Wrap(err, "failed to setup Openshift IDP")
		}

		err = r.setupGithubIDP(ctx, kcr, serverClient)
		if err != nil {
			return errors.Wrap(err, "failed to setup Github IDP")
		}
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update keycloak realm")
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	// Create the customer admin
	or, err = r.createOrUpdateKeycloakUser(CustomerAdminUser, inst, ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update the customer admin user")
	} else {
		r.logger.Infof("The operation result for keycloakuser %s was %s", CustomerAdminUser.UserName, or)
	}

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to list the keycloak users")
	}

	// Sync keycloak with openshift users
	users, err := syncronizeWithOpenshiftUsers(keycloakUsers, ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to synchronize the users")
	}

	// Create / update the synchronized users
	for _, user := range users {
		or, err = r.createOrUpdateKeycloakUser(*user, inst, ctx, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update the customer admin user")
		} else {
			r.logger.Infof("The operation result for keycloakuser %s was %s", user.UserName, or)
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	kc := &keycloak.Keycloak{}
	// if this errors, it can be ignored
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err == nil && string(r.Config.GetProductVersion()) != kc.Status.Version {
		r.Config.SetProductVersion(kc.Status.Version)
		r.ConfigManager.WriteConfig(r.Config)
	}

	// The Keycloak Operator doesn't currently set the operator version
	r.Config.SetOperatorVersion(string(v1alpha1.OperatorVersionRHSSO))
	r.ConfigManager.WriteConfig(r.Config)

	r.logger.Info("checking ready status for rhsso")
	kcr := &keycloak.KeycloakRealm{}

	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get keycloak realm custom resource")
	}

	if kcr.Status.Phase == keycloak.PhaseReconciling {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to write rhsso config")
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm")
		return v1alpha1.PhaseCompleted, nil
	}

	r.logger.Infof("KeycloakRealm status phase is: %s", kcr.Status.Phase)
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) exportConfig(ctx context.Context, serverClient pkgclient.Client) error {
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return pkgerr.Wrap(err, "could not retrieve keycloak custom resource for keycloak config")
	}

	r.Config.SetRealm(keycloakRealmName)
	r.Config.SetHost(kc.Status.InternalURL)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return pkgerr.Wrap(err, "could not update keycloak config")
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, inst *v1alpha1.Installation, kcr *keycloak.KeycloakRealm, serverClient pkgclient.Client) error {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return pkgerr.Wrapf(err, "Could not find %s Secret", oauthClientSecrets.Name)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return pkgerr.Wrapf(err, "Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	clientSecret := string(clientSecretBytes)

	oauthc := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			r.Config.GetHost() + "/auth/realms/openshift/broker/openshift-v4/endpoint",
		},
		GrantMethod: oauthv1.GrantHandlerPrompt,
	}

	_, err = r.ReconcileOauthClient(ctx, inst, oauthc, serverClient)
	if err != nil {
		return err
	}

	if !containsIdentityProvider(kcr.Spec.Realm.IdentityProviders, idpAlias) {
		logrus.Infof("Adding keycloak realm client")
		if kcr.Spec.Realm.IdentityProviders == nil {
			kcr.Spec.Realm.IdentityProviders = []*keycloak.KeycloakIdentityProvider{}
		}
		kcr.Spec.Realm.IdentityProviders = append(kcr.Spec.Realm.IdentityProviders, &keycloak.KeycloakIdentityProvider{
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
				"clientId":        r.getOAuthClientName(),
				"disableUserInfo": "",
				"clientSecret":    clientSecret,
				"defaultScope":    "user:full",
				"useJwksUrl":      "true",
			},
		})
	}
	return nil
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error reading monitoring config")
	}

	err = monitoring.CreateBlackboxTarget("integreatly-rhsso", v1alpha12.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "rhsso-ui",
	}, ctx, cfg, inst, client)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating rhsso blackbox target")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) setupGithubIDP(ctx context.Context, kcr *keycloak.KeycloakRealm, serverClient pkgclient.Client) error {
	githubCreds := &v1.Secret{}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: githubOauthAppCredentialsSecretName, Namespace: r.ConfigManager.GetOperatorNamespace()}, githubCreds)
	if err != nil {
		logrus.Errorf("Unable to find Github oauth credentials secret in namespace %s", r.ConfigManager.GetOperatorNamespace())
		return err
	}

	if !containsIdentityProvider(kcr.Spec.Realm.IdentityProviders, githubIdpAlias) {
		logrus.Infof("Adding github identity provider to the keycloak realm")
		if kcr.Spec.Realm.IdentityProviders == nil {
			kcr.Spec.Realm.IdentityProviders = []*keycloak.KeycloakIdentityProvider{}
		}
		kcr.Spec.Realm.IdentityProviders = append(kcr.Spec.Realm.IdentityProviders, &keycloak.KeycloakIdentityProvider{
			Alias:                     githubIdpAlias,
			ProviderID:                githubIdpAlias,
			Enabled:                   true,
			TrustEmail:                false,
			StoreToken:                true,
			AddReadTokenRoleOnCreate:  true,
			FirstBrokerLoginFlowAlias: "first broker login",
			LinkOnly:                  true,
			Config: map[string]string{
				"hideOnLoginPage": "true",
				"clientId":        fmt.Sprintf("%s", githubCreds.Data["clientId"]),
				"disableUserInfo": "",
				"clientSecret":    fmt.Sprintf("%s", githubCreds.Data["secret"]),
				"defaultScope":    "repo,user,write:public_key,admin:repo_hook,read:org,public_repo,user:email",
				"useJwksUrl":      "true",
			},
		})
	}
	// We need to revisit how the github idp gets created/updated
	// Client ID and secret can get outdated we need to ensure they are synced with the value secret in the github-oauth-secret
	return nil
}

func containsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

func getUserDiff(keycloakUsers []*keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User) ([]usersv1.User, []int) {
	var added []usersv1.User
	for _, osUser := range openshiftUsers {
		if !kcContainsOsUser(keycloakUsers, osUser) {
			added = append(added, osUser)
		}
	}

	var deleted []int
	for i, kcUser := range keycloakUsers {
		if kcUser.UserName != CustomerAdminUser.UserName && !OsUserInKc(openshiftUsers, kcUser) {
			deleted = append(deleted, i)
		}
	}

	return added, deleted
}

func syncronizeWithOpenshiftUsers(keycloakUsers []*keycloak.KeycloakAPIUser, ctx context.Context, serverClient pkgclient.Client) ([]*keycloak.KeycloakAPIUser, error) {
	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, &pkgclient.ListOptions{}, openshiftUsers)
	if err != nil {
		return nil, err
	}

	added, deletedIndexes := getUserDiff(keycloakUsers, openshiftUsers.Items)
	for _, index := range deletedIndexes {
		keycloakUsers = remove(index, keycloakUsers)
	}

	for _, osUser := range added {
		email := osUser.Name
		if !strings.Contains(email, "@") {
			email = email + "@example.com"
		}
		keycloakUsers = append(keycloakUsers, &keycloak.KeycloakAPIUser{
			Enabled:       true,
			UserName:      osUser.Name,
			EmailVerified: true,
			Email:         email,
			FederatedIdentities: []keycloak.FederatedIdentity{
				{
					IdentityProvider: idpAlias,
					UserID:           string(osUser.UID),
					UserName:         osUser.Name,
				},
			},
		})
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

func remove(index int, kcUsers []*keycloak.KeycloakAPIUser) []*keycloak.KeycloakAPIUser {
	kcUsers[index] = kcUsers[len(kcUsers)-1]
	return kcUsers[:len(kcUsers)-1]
}

func kcContainsOsUser(kcUsers []*keycloak.KeycloakAPIUser, osUser usersv1.User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == osUser.Name {
			return true
		}
	}

	return false
}

func OsUserInKc(osUsers []usersv1.User, kcUser *keycloak.KeycloakAPIUser) bool {
	for _, osu := range osUsers {
		if osu.Name == kcUser.UserName {
			return true
		}
	}

	return false
}

func isOpenshiftAdmin(kcUser *keycloak.KeycloakAPIUser, adminGroup *usersv1.Group) bool {
	for _, name := range adminGroup.Users {
		if kcUser.UserName == name {
			return true
		}
	}

	return false
}

func (r *Reconciler) createOrUpdateKeycloakUser(user keycloak.KeycloakAPIUser, inst *v1alpha1.Installation, ctx context.Context, serverClient pkgclient.Client) (controllerutil.OperationResult, error) {
	selector := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("generated-%v", user.UserName),
			Namespace: r.Config.GetNamespace(),
		},
	}

	ownerutil.EnsureOwner(selector, inst)
	return controllerutil.CreateOrUpdate(ctx, serverClient, selector, func(existing runtime.Object) error {
		keycloakUser := existing.(*keycloak.KeycloakUser)
		keycloakUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: GetInstanceLabels(),
		}
		keycloakUser.Spec.User = user
		return nil
	})
}

func GetKeycloakUsers(ctx context.Context, serverClient pkgclient.Client, ns string) ([]*keycloak.KeycloakAPIUser, error) {
	var users keycloak.KeycloakUserList
	var mappedUsers []*keycloak.KeycloakAPIUser

	labelSelector, err := labels.Parse(fmt.Sprintf("%v=%v", SSOLabelKey, SSOLabelValue))
	if err != nil {
		return nil, err
	}

	listOptions := pkgclient.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     ns,
	}
	err = serverClient.List(ctx, &listOptions, &users)
	if err != nil {
		return nil, err
	}

	for _, user := range users.Items {
		mappedUsers = append(mappedUsers, &user.Spec.User)
	}

	return mappedUsers, nil
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

func GetInstanceLabels() map[string]string {
	return map[string]string{
		SSOLabelKey: SSOLabelValue,
	}
}
