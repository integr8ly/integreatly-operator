package rhsso

import (
	"context"
	"fmt"
	"github.com/pkg/errors"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhssocommon"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"

	corev1 "k8s.io/api/core/v1"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/keycloak/keycloak-operator/pkg/common"

	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultOperandNamespace   = "rhsso"
	keycloakName              = "rhsso"
	keycloakRealmName         = "openshift"
	idpAlias                  = "openshift-v4"
	githubIdpAlias            = "github"
	authFlowAlias             = "authdelay"
	adminCredentialSecretName = "credential-" + keycloakName
	ssoType                   = "rhsso"
	postgresResourceName      = "rhsso-postgres-rhmi"
	routeName                 = "keycloak-edge"
)

const (
	SSOLabelKey   = "sso"
	SSOLabelValue = "integreatly"
	RHSSOProfile  = "RHSSO"
)

type Reconciler struct {
	Config *config.RHSSO
	Log    l.Logger
	*rhssocommon.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, APIURL string, keycloakClientFactory keycloakCommon.KeycloakClientFactory, logger l.Logger) (*Reconciler, error) {
	config, err := configManager.ReadRHSSO()

	if err != nil {
		return nil, err
	}

	rhssocommon.SetNameSpaces(installation, config.RHSSOCommon, defaultOperandNamespace)

	return &Reconciler{
		Config:     config,
		Log:        logger,
		Reconciler: rhssocommon.NewReconciler(configManager, mpm, installation, logger, oauthv1Client, recorder, APIURL, keycloakClientFactory),
	}, nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.AuthenticationStage].Products[integreatlyv1alpha1.ProductRHSSO],
		string(integreatlyv1alpha1.VersionRHSSO),
		string(integreatlyv1alpha1.OperatorVersionRHSSO),
	)
}

// Reconcile reads that state of the cluster for rhsso and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, productNamespace, serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := r.CleanupKeycloakResources(ctx, installation, serverClient, productNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = resources.RemoveNamespace(ctx, installation, serverClient, productNamespace, r.Log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		_, err = resources.GetNS(ctx, operatorNamespace, serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.Log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		err = resources.RemoveOauthClient(r.Oauthv1Client, r.GetOAuthClientName(r.Config), r.Log)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		//if both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, operatorNamespace, serverClient)
		_, nsErr := resources.GetNS(ctx, productNamespace, serverClient)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(nsErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	}, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}
	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileSecretToProductNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, productNamespace, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile admin credentials secret", err)
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace, postgresResourceName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.RHSSOSubscriptionName), err)
		return phase, err
	}

	phase, err = r.CreateKeycloakRoute(ctx, serverClient, r.Config, r.Config.RHSSOCommon, routeName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.ReconcileCloudResources(constants.RHSSOPostgresPrefix, defaultOperandNamespace, ssoType, r.Config.RHSSOCommon, ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile cloud resources", err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.ReconcileStatefulSet(ctx, serverClient, r.Config.RHSSOCommon)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconsile RHSSO pod priority", err)
		return phase, err
	}

	phase, err = r.HandleProgressPhase(ctx, serverClient, keycloakName, keycloakRealmName, r.Config, r.Config.RHSSOCommon, string(integreatlyv1alpha1.VersionRHSSO), string(integreatlyv1alpha1.OperatorVersionRHSSO))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error writing to config in rhsso cluster reconciler: %w", err)
	}

	phase, err = r.ReconcileBlackboxTargets(ctx, installation, serverClient, "integreatly-rhsso", r.Config.GetHost(), "rhsso-ui")
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.RemovePodMonitors(ctx, serverClient, r.Config)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to remove pod monitor", err)
		return phase, err
	}

	phase, err = resources.ReconcileSecretToRHMIOperatorNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile admin credential secret to RHMI operator namespace", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler(r.Log, r.Installation.Spec.Type).ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.Recorder, installation, integreatlyv1alpha1.AuthenticationStage, r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.Log.Info("Reconciling Keycloak components")
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func() error {
		kc.Spec.Extensions = []string{
			"https://github.com/aerogear/keycloak-metrics-spi/releases/download/2.0.1/keycloak-metrics-spi-2.0.1.jar",
			"https://github.com/integr8ly/authentication-delay-plugin/releases/download/1.0.2/authdelay.jar",
		}
		kc.Labels = GetInstanceLabels()
		kc.Spec.ExternalDatabase = keycloak.KeycloakExternalDatabase{Enabled: true}
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{
			Enabled: true,
		}
		kc.Spec.Profile = RHSSOProfile
		kc.Spec.PodDisruptionBudget = keycloak.PodDisruptionBudgetConfig{Enabled: true}
		kc.Spec.KeycloakDeploymentSpec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("650m"), corev1.ResourceMemory: k8sresource.MustParse("2G")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("650m"), corev1.ResourceMemory: k8sresource.MustParse("2G")},
		}
		//OSD has more resources than PROW, so adding an exception
		numberOfReplicas := r.Config.GetReplicasConfig(r.Installation)

		if kc.Spec.Instances < numberOfReplicas {
			kc.Spec.Instances = numberOfReplicas
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	host := r.Config.GetHost()
	if host == "" {
		r.Log.Info("URL for Keycloak not yet available")
		return integreatlyv1alpha1.PhaseAwaitingComponents, fmt.Errorf("Host for Keycloak not yet available")
	}

	r.Log.Infof("Operation result", l.Fields{"keycloak": kc.Name, "result": or})
	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func() error {
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
		redirectUris := []string{r.Config.GetHost() + "/auth/realms/openshift/broker/openshift-v4/endpoint"}
		err = r.SetupOpenshiftIDP(ctx, serverClient, installation, r.Config, kcr, redirectUris)
		if err != nil {
			return fmt.Errorf("failed to setup Openshift IDP: %w", err)
		}

		err = r.setupGithubIDP(ctx, kc, kcr, serverClient, installation)
		if err != nil {
			return fmt.Errorf("failed to setup Github IDP: %w", err)
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloakrealm": kcr.Name, "result": or})

	// create keycloak authentication delay flow and adds to openshift idp
	authenticated, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to authenticate client in keycloak api %w", err)
	}

	err = createAuthDelayAuthenticationFlow(authenticated)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create and add keycloak authentication flow: %w", err)
	}
	r.Log.Infof("Authentication flow added to IDP", l.Fields{"idpAlias": idpAlias})

	err = r.SyncOpenshiftIDPClientSecret(ctx, serverClient, authenticated, r.Config, keycloakRealmName)
	if err != nil {
		r.Log.Error("Failed to sync openshift idp client secret", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to sync openshift idp client secret: %w", err)
	}

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list the keycloak users: %w", err)
	}

	// Sync keycloak with openshift users
	users, err := syncronizeWithOpenshiftUsers(ctx, keycloakUsers, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to synchronize the users: %w", err)
	}

	// Create / update the synchronized users
	for _, user := range users {
		or, err = r.createOrUpdateKeycloakUser(ctx, user, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update the customer admin user: %w", err)
		}
		r.Log.Infof("Operation result", l.Fields{"keycloakuser": user.UserName, "result": or})
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) setupGithubIDP(ctx context.Context, kc *keycloak.Keycloak, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) error {
	githubCreds := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: r.ConfigManager.GetGHOauthClientsSecretName(), Namespace: r.ConfigManager.GetOperatorNamespace()}, githubCreds)
	if err != nil {
		r.Log.Errorf("Unable to find Github oauth credentials secret", l.Fields{"ns": r.ConfigManager.GetOperatorNamespace()}, err)
		return err
	}

	if !rhssocommon.ContainsIdentityProvider(kcr.Spec.Realm.IdentityProviders, githubIdpAlias) {
		r.Log.Info("Adding github identity provider to the keycloak realm")
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

	githubClientID := string(githubCreds.Data["clientId"])
	githubClientSecret := string(githubCreds.Data["secret"])

	// check if GH credentials have been set up
	githubMockCred := "dummy"
	if githubClientID == githubMockCred || githubClientSecret == githubMockCred {
		return nil
	}

	r.Log.Info("Syncing github identity provider to the keycloak realm")

	// Get an authenticated keycloak api client for the instance
	keycloakFactory := common.LocalConfigKeycloakFactory{}
	authenticated, err := keycloakFactory.AuthenticatedClient(*kc)
	if err != nil {
		return fmt.Errorf("Unable to authenticate to the Keycloak API: %s", err)
	}

	identityProvider, err := authenticated.GetIdentityProvider(githubIdpAlias, kcr.Spec.Realm.DisplayName)
	if err != nil {
		return fmt.Errorf("Unable to get Identity Provider from Keycloak API: %s", err)
	}

	identityProvider.Config["clientId"] = githubClientID
	identityProvider.Config["clientSecret"] = githubClientSecret
	err = authenticated.UpdateIdentityProvider(identityProvider, kcr.Spec.Realm.DisplayName)
	if err != nil {
		return fmt.Errorf("Unable to update Identity Provider to Keycloak API: %s", err)
	}

	installation.Status.GitHubOAuthEnabled = true

	return nil
}

func getUserDiff(keycloakUsers []keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User, groups *usersv1.GroupList) (added []usersv1.User, deleted []keycloak.KeycloakAPIUser) {
	for _, osUser := range openshiftUsers {
		if !kcContainsOsUser(keycloakUsers, osUser) && !userHelper.UserInExclusionGroup(osUser, groups) {
			added = append(added, osUser)
		}
	}

	for _, kcUser := range keycloakUsers {
		if !rhssocommon.OsUserInKc(openshiftUsers, kcUser) {
			deleted = append(deleted, kcUser)
		}
	}

	return added, deleted
}

func syncronizeWithOpenshiftUsers(ctx context.Context, keycloakUsers []keycloak.KeycloakAPIUser, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	openshiftUsers, err := userHelper.GetUsersInActiveIDPs(ctx, serverClient)
	if err != nil {
		return nil, errors.Wrap(err, "could not get users in active IDPs")
	}

	groups := &usersv1.GroupList{}
	err = serverClient.List(ctx, groups)
	if err != nil {
		return nil, err
	}

	added, deletedUsers := getUserDiff(keycloakUsers, openshiftUsers.Items, groups)

	keycloakUsers, err = rhssocommon.DeleteKeycloakUsers(keycloakUsers, deletedUsers, ns, ctx, serverClient)
	if err != nil {
		return nil, err
	}

	for _, osUser := range added {
		email, err := userHelper.GetUserEmailFromIdentity(ctx, serverClient, osUser)

		if err != nil {
			return nil, err
		}

		if email == "" {
			email = osUser.Name + "@rhmi.io"
		}

		newKeycloakUser := keycloak.KeycloakAPIUser{
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
		}
		userHelper.AppendUpdateProfileActionForUserWithoutEmail(&newKeycloakUser)

		keycloakUsers = append(keycloakUsers, newKeycloakUser)
	}

	if err != nil && !k8serr.IsNotFound(err) {
		return nil, err
	}
	for index := range keycloakUsers {
		keycloakUsers[index].ClientRoles = getKeycloakRoles()
	}

	return keycloakUsers, nil
}

func kcContainsOsUser(kcUsers []keycloak.KeycloakAPIUser, osUser usersv1.User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == osUser.Name {
			return true
		}
	}

	return false
}

func (r *Reconciler) createOrUpdateKeycloakUser(ctx context.Context, user keycloak.KeycloakAPIUser, serverClient k8sclient.Client) (controllerutil.OperationResult, error) {
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userHelper.GetValidGeneratedUserName(user),
			Namespace: r.Config.GetNamespace(),
		},
	}

	return controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: GetInstanceLabels(),
		}
		kcUser.Labels = GetInstanceLabels()
		kcUser.Spec.User = user
		return nil
	})
}

func GetKeycloakUsers(ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(GetInstanceLabels()),
		k8sclient.InNamespace(ns),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		return nil, err
	}

	mappedUsers := make([]keycloak.KeycloakAPIUser, len(users.Items))
	for i, user := range users.Items {
		mappedUsers[i] = user.Spec.User
	}

	return mappedUsers, nil
}

// creates keycloak authentication flow to delay login until user is reconciled in 3scale or other products
func createAuthDelayAuthenticationFlow(authenticated keycloakCommon.KeycloakInterface) error {

	authFlow, err := authenticated.FindAuthenticationFlowByAlias(authFlowAlias, keycloakRealmName)
	if err != nil {
		return fmt.Errorf("failed to find authentication flow by alias via keycloak api %w", err)
	}
	if authFlow == nil {
		authFlow := keycloakCommon.AuthenticationFlow{
			Alias:      authFlowAlias,
			ProviderID: "basic-flow", // providerId is "client-flow" for client and "basic-flow" for generic in Top Level Flow Type
			TopLevel:   true,
			BuiltIn:    false,
		}
		_, err := authenticated.CreateAuthenticationFlow(authFlow, keycloakRealmName)
		if err != nil {
			return fmt.Errorf("failed to create authentication flow via keycloak api %w", err)
		}
	}

	executionProviderID := "delay-authentication"
	authExecution, err := authenticated.FindAuthenticationExecutionForFlow(authFlowAlias, keycloakRealmName, func(execution *keycloak.AuthenticationExecutionInfo) bool {
		return execution.ProviderID == executionProviderID
	})
	if err != nil {
		return fmt.Errorf("failed to find authentication execution flow via keycloak api %w", err)
	}
	if authExecution == nil {
		err = authenticated.AddExecutionToAuthenticatonFlow(authFlowAlias, keycloakRealmName, executionProviderID, keycloakCommon.Required)
		if err != nil {
			return fmt.Errorf("failed to add execution to authentication flow via keycloak api %w", err)
		}
	}

	idp, err := authenticated.GetIdentityProvider(idpAlias, keycloakRealmName)
	if err != nil {
		return fmt.Errorf("failed to get identity provider via keycloak api %w", err)
	}
	if idp.FirstBrokerLoginFlowAlias != authFlowAlias {
		idp.FirstBrokerLoginFlowAlias = authFlowAlias
		err = authenticated.UpdateIdentityProvider(idp, keycloakRealmName)
		if err != nil {
			return fmt.Errorf("failed to update identity provider via keycloak api %w", err)
		}
	}

	return nil
}

func getKeycloakRoles() map[string][]string {
	roles := map[string][]string{
		"account": {
			"manage-account",
			"view-profile",
		},
		"broker": {
			"read-token",
		},
	}
	return roles
}

func GetInstanceLabels() map[string]string {
	return map[string]string{
		SSOLabelKey: SSOLabelValue,
	}
}
