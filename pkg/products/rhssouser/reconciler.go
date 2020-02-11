package rhssouser

import (
	"context"
	"fmt"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	usersv1 "github.com/openshift/api/user/v1"
	"github.com/pkg/errors"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultRhssoNamespace   = "user-sso"
	keycloakName            = "rhssouser"
	defaultSubscriptionName = "integreatly-rhsso"
	idpAlias                = "openshift-v4"
	manifestPackage         = "integreatly-rhsso"
	masterRealmName         = "master"
)

const (
	masterRealmLabelKey   = "sso"
	masterRealmLabelValue = "master"
)

type Reconciler struct {
	Config        *config.RHSSOUser
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	ApiUrl        string
	*resources.Reconciler
	recorder              record.EventRecorder
	keycloakClientFactory keycloakCommon.KeycloakClientFactory
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, apiUrl string, keycloakClientFactory keycloakCommon.KeycloakClientFactory) (*Reconciler, error) {
	config, err := configManager.ReadRHSSOUser()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultRhssoNamespace)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:                config,
		ConfigManager:         configManager,
		mpm:                   mpm,
		installation:          installation,
		logger:                logger,
		oauthv1Client:         oauthv1Client,
		Reconciler:            resources.NewReconciler(mpm),
		recorder:              recorder,
		ApiUrl:                apiUrl,
		keycloakClientFactory: keycloakClientFactory,
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
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := r.cleanupKeycloakResources(ctx, installation, serverClient)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = r.isKeycloakResourcesDeleted(ctx, serverClient)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
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
		err = resources.RemoveOauthClient(r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
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

	phase, err = r.createKeycloakRoute(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
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

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling Keycloak components")
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func() error {
		owner.AddIntegreatlyOwnerAnnotations(kc, installation)
		kc.Spec.Extensions = []string{
			"https://github.com/aerogear/keycloak-metrics-spi/releases/download/1.0.4/keycloak-metrics-spi-1.0.4.jar",
		}
		kc.Labels = getMasterLabels()
		kc.Spec.Instances = 3
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{Enabled: true}
		kc.Spec.Profile = rhsso.RHSSOProfile
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	r.logger.Infof("The operation result for keycloak %s was %s", kc.Name, or)

	// We want to update the master realm by adding an openshift-v4 idp. We can not add the idp until we know the host
	if r.Config.GetHost() == "" {
		logrus.Warningf("Can not update keycloak master realm on user sso as host is not available yet")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak master realm, host not available")
	}

	// Create the master realm. The master real already exists in Keycloak but we need to get a reference to it
	// in order to create the IDP and admin users on it
	masterKcr, err := r.updateMasterRealm(ctx, serverClient, installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	kcClient, err := r.keycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Ensure the IDP exists before trying to create via rhsso client.
	// We have to create via rhsso client as keycloak will not accept changes to the master realm, via cr changes,
	// after its initial creation
	if masterKcr.Spec.Realm.IdentityProviders == nil && masterKcr.Spec.Realm.IdentityProviders[0] == nil {
		logrus.Warningf("Identity Provider not present on Realm - user sso")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update keycloak master realm with required IDP: %w", err)
	}

	exists, err := identityProviderExists(kcClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error attempting to get existing idp on user sso, master realm")
	} else if !exists {
		err = kcClient.CreateIdentityProvider(masterKcr.Spec.Realm.IdentityProviders[0], masterKcr.Spec.Realm.Realm)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error creating idp on master realm, user sso")
		}
	}

	phase, err := r.reconcileBrowserAuthFlow(ctx, kc, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile browser authentication flow", err)
		return phase, err
	}

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to list the keycloak users")
	}

	// Sync keycloak with openshift users
	users, err := syncAdminUsersInMasterRealm(keycloakUsers, ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to synchronize the users")
	}

	// Create / update the synchronized users
	for _, user := range users {
		or, err = r.createOrUpdateKeycloakAdmin(user, ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update the customer admin user")
		} else {
			r.logger.Infof("The operation result for keycloakuser %s was %s", user.UserName, or)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func identityProviderExists(kcClient keycloakCommon.KeycloakInterface) (bool, error) {
	provider, err := kcClient.GetIdentityProvider(idpAlias, "master")
	if err != nil {
		return false, err
	}
	if provider != nil {
		return true, nil
	}
	return false, nil
}

// The master realm will be created as part of the Keycloak install. Here we update it to add the openshift idp
func (r *Reconciler) updateMasterRealm(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (*keycloak.KeycloakRealm, error) {

	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      masterRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func() error {
		kcr.Spec.RealmOverrides = []*keycloak.RedirectorIdentityProviderOverride{
			{
				IdentityProvider: idpAlias,
				ForFlow:          "browser",
			},
		}

		kcr.Spec.InstanceSelector = &metav1.LabelSelector{
			MatchLabels: getMasterLabels(),
		}

		kcr.Labels = getMasterLabels()

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          masterRealmName,
			Realm:       masterRealmName,
			Enabled:     true,
			DisplayName: masterRealmName,
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		err := r.setupOpenshiftIDP(ctx, installation, kcr, serverClient)
		if err != nil {
			return errors.Wrap(err, "failed to setup Openshift IDP for user-sso")
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	return kcr, nil
}

func (r *Reconciler) createOrUpdateKeycloakAdmin(user keycloak.KeycloakAPIUser, ctx context.Context, serverClient k8sclient.Client) (controllerutil.OperationResult, error) {
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("generated-%v", user.UserName),
			Namespace: r.Config.GetNamespace(),
		},
	}

	return controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: getMasterLabels(),
		}
		kcUser.Labels = getMasterLabels()
		kcUser.Spec.User = user

		return nil
	})
}

func GetKeycloakUsers(ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(getMasterLabels()),
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

func getMasterLabels() map[string]string {
	return map[string]string{
		masterRealmLabelKey: masterRealmLabelValue,
	}
}

func syncAdminUsersInMasterRealm(keycloakUsers []keycloak.KeycloakAPIUser, ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {

	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, openshiftUsers)
	if err != nil {
		return nil, err
	}
	openshiftGroups := &usersv1.GroupList{}
	err = serverClient.List(ctx, openshiftGroups)
	if err != nil {
		return nil, err
	}

	dedicatedAdminUsers := getDedicatedAdmins(*openshiftUsers, *openshiftGroups)

	// added => Newly added to dedicated-admins group and OS
	// deleted => No longer exists in OS, remove from SSO
	// promoted => existing KC user, added to dedicated-admins group, promote KC privileges
	// demoted => existing KC user, removed from dedicated-admins group, demote KC privileges
	added, deleted, promoted, demoted := getUserDiff(keycloakUsers, openshiftUsers.Items, dedicatedAdminUsers)

	keycloakUsers, err = deleteKeycloakUsers(keycloakUsers, deleted, ns, ctx, serverClient)
	if err != nil {
		return nil, err
	}

	keycloakUsers = addKeycloakUsers(keycloakUsers, added)
	keycloakUsers = promoteKeycloakUsers(keycloakUsers, promoted)
	keycloakUsers = demoteKeycloakUsers(keycloakUsers, demoted)

	return keycloakUsers, nil
}

func addKeycloakUsers(keycloakUsers []keycloak.KeycloakAPIUser, added []usersv1.User) []keycloak.KeycloakAPIUser {

	for _, osUser := range added {

		email := osUser.Name
		if !strings.Contains(email, "@") {
			email = email + "@example.com"
		}
		keycloakUsers = append(keycloakUsers, keycloak.KeycloakAPIUser{
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
			RealmRoles: []string{"offline_access", "uma_authorization", "create-realm"},
			ClientRoles: map[string][]string{
				"account": {
					"manage-account",
					"view-profile",
				},
				"master-realm": {
					"view-clients",
					"view-realm",
					"manage-users",
				},
			},
		})
	}
	return keycloakUsers
}

func promoteKeycloakUsers(allUsers []keycloak.KeycloakAPIUser, promoted []keycloak.KeycloakAPIUser) []keycloak.KeycloakAPIUser {

	for _, promotedUser := range promoted {
		for i, user := range allUsers {
			// ID is not populated, have to use UserName. Should be unique on master Realm
			if promotedUser.UserName == user.UserName {
				allUsers[i].ClientRoles = map[string][]string{
					"account": {
						"manage-account",
						"view-profile",
					},
					"master-realm": {
						"view-clients",
						"view-realm",
						"manage-users",
					}}
				allUsers[i].RealmRoles = []string{"offline_access", "uma_authorization", "create-realm"}
				break
			}
		}
	}

	return allUsers
}

func demoteKeycloakUsers(allUsers []keycloak.KeycloakAPIUser, demoted []keycloak.KeycloakAPIUser) []keycloak.KeycloakAPIUser {

	for _, demotedUser := range demoted {
		for i, user := range allUsers {
			// ID is not populated, have to use UserName. Should be unique on master Realm
			if demotedUser.UserName == user.UserName { // ID is not set but UserName is
				allUsers[i].ClientRoles = map[string][]string{
					"account": {
						"manage-account",
						"manage-account-links",
						"view-profile",
					}}
				allUsers[i].RealmRoles = []string{"offline_access", "uma_authorization"}
				break
			}
		}
	}

	return allUsers
}

// NOTE: The users type has a Groups field on it but it does not seem to get populated
// hence the need to check by name which is not ideal. However, this is the only field
// available on the Group type
func getDedicatedAdmins(osUsers usersv1.UserList, groups usersv1.GroupList) (dedicatedAdmins []usersv1.User) {

	var osUsersInGroup = getOsUsersInAdminsGroup(groups)

	for _, osUser := range osUsers.Items {
		if contains(osUsersInGroup, osUser.Name) {
			dedicatedAdmins = append(dedicatedAdmins, osUser)
		}
	}
	return dedicatedAdmins
}

func getOsUsersInAdminsGroup(groups usersv1.GroupList) (users []string) {

	for _, group := range groups.Items {
		if group.Name == "dedicated-admins" {
			if group.Users != nil {
				users = group.Users
			}
			break
		}
	}

	return users
}

func deleteKeycloakUsers(allKcUsers []keycloak.KeycloakAPIUser, deletedUsers []keycloak.KeycloakAPIUser, ns string, ctx context.Context, serverClient k8sclient.Client) ([]keycloak.KeycloakAPIUser, error) {

	for _, delUser := range deletedUsers {
		// Remove from all users list
		for i, user := range allKcUsers {
			// ID is not populated, have to use UserName. Should be unique on master Realm
			if delUser.UserName == user.UserName {
				allKcUsers[i] = allKcUsers[len(allKcUsers)-1]
				allKcUsers = allKcUsers[:len(allKcUsers)-1]
				break
			}
		}

		// Delete the CR
		kcUser := &keycloak.KeycloakUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("generated-%v", delUser.UserName),
				Namespace: ns,
			},
		}
		err := serverClient.Delete(ctx, kcUser)
		if err != nil {
			return nil, fmt.Errorf("failed to delete keycloak user: %w", err)
		}
	}

	return allKcUsers, nil
}

// There are 3 conceptual user types
// 1. OpenShift User. 2. Keycloak User created by CR 3. Keycloak User created by customer
// The distinction is important as we want to try avoid managing users created by the customer apart from certain
// scenarios such as removing a user if they do not exist in OpenShift. This needs further consideration

// This function should return
// 1. Users in dedicated-admins group but not keycloak master realm => Added
// return osUser list
// 2. Users in dedicated-admins group, in keycloak master realm, but not have privledges => Promoted
// return keylcoak user list
// 3. Users in OS, Users in keycloak master realm, represented by a Keycloak CR, but not dedicated-admins group => Demote
// return keylcoak user list
// 4. Users not in OpenShift but in Keycloak Master Realm, represented by a Keycloak CR 			=> Delete
// return keylcoak user list
func getUserDiff(keycloakUsers []keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User, dedicatedAdmins []usersv1.User) ([]usersv1.User, []keycloak.KeycloakAPIUser, []keycloak.KeycloakAPIUser, []keycloak.KeycloakAPIUser) {
	var added []usersv1.User
	var deleted []keycloak.KeycloakAPIUser
	var promoted []keycloak.KeycloakAPIUser
	var demoted []keycloak.KeycloakAPIUser

	for _, admin := range dedicatedAdmins {
		keycloakUser := getKeyCloakUser(admin, keycloakUsers)
		if keycloakUser == nil {
			// User in dedicated-admins group but not keycloak master realm
			added = append(added, admin)
		} else {
			if !hasAdminPrivileges(keycloakUser) {
				// User in dedicated-admins group, in keycloak master realm, but does not have privledges
				promoted = append(promoted, *keycloakUser)
			}
		}
	}

	for _, kcUser := range keycloakUsers {
		osUser := getOpenShiftUser(kcUser, openshiftUsers)
		if osUser != nil && !kcUserInDedicatedAdmins(kcUser, dedicatedAdmins) && hasAdminPrivileges(&kcUser) {
			// User in OS and keycloak master realm, represented by a Keycloak CR, but not dedicated-admins group
			demoted = append(demoted, kcUser)
		} else if osUser == nil {
			// User not in OpenShift but is in Keycloak Master Realm, represented by a Keycloak CR
			deleted = append(deleted, kcUser)
		}
	}

	return added, deleted, promoted, demoted
}

func kcUserInDedicatedAdmins(kcUser keycloak.KeycloakAPIUser, admins []usersv1.User) bool {
	for _, admin := range admins {
		if kcUser.FederatedIdentities[0].UserID == string(admin.UID) {
			return true
		}
	}
	return false
}

func getOpenShiftUser(kcUser keycloak.KeycloakAPIUser, osUsers []usersv1.User) *usersv1.User {
	for _, osUser := range osUsers {
		if len(kcUser.FederatedIdentities) >= 1 && kcUser.FederatedIdentities[0].UserID == string(osUser.UID) {
			return &osUser
		}
	}
	return nil
}

// Look for 2 key privileges to determine if user has admin rights
func hasAdminPrivileges(kcUser *keycloak.KeycloakAPIUser) bool {
	if len(kcUser.ClientRoles["master-realm"]) >= 1 && contains(kcUser.ClientRoles["master-realm"], "manage-users") && contains(kcUser.RealmRoles, "create-realm") {
		return true
	}
	return false
}

func contains(items []string, find string) bool {
	for _, item := range items {
		if item == find {
			return true
		}
	}
	return false
}

func getKeyCloakUser(admin usersv1.User, kcUsers []keycloak.KeycloakAPIUser) *keycloak.KeycloakAPIUser {
	for _, kcUser := range kcUsers {
		if kcUser.FederatedIdentities[0].UserID == string(admin.UID) {
			return &kcUser
		}
	}
	return nil
}

func OsUserInDedicatedAdmins(dedicatedAdmins []string, kcUser keycloak.KeycloakAPIUser) bool {
	for _, user := range dedicatedAdmins {
		if kcUser.UserName == user {
			return true
		}
	}
	return false
}

func OsUserInKc(osUsers []usersv1.User, kcUser keycloak.KeycloakAPIUser) bool {
	for _, osu := range osUsers {
		if osu.Name == kcUser.UserName {
			return true
		}
	}

	return false
}

func (r *Reconciler) cleanupKeycloakResources(ctx context.Context, inst *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	opts := &k8sclient.ListOptions{
		Namespace: r.Config.GetNamespace(),
	}

	// Delete all users
	users := &keycloak.KeycloakUserList{}
	err := serverClient.List(ctx, users, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	for _, user := range users.Items {
		err = serverClient.Delete(ctx, &user)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Delete all clients
	clients := &keycloak.KeycloakClientList{}
	err = serverClient.List(ctx, clients, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	for _, client := range clients.Items {
		err = serverClient.Delete(ctx, &client)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Delete all realms
	realms := &keycloak.KeycloakRealmList{}
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
	}
	for _, realm := range realms.Items {
		err = serverClient.Delete(ctx, &realm)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		realm.SetFinalizers([]string{})
		err := serverClient.Update(ctx, &realm)
		if !k8serr.IsNotFound(err) && err != nil {
			logrus.Info("Error removing finalizer from Realm", err)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) isKeycloakResourcesDeleted(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	opts := &k8sclient.ListOptions{
		Namespace: r.Config.GetNamespace(),
	}

	// Check if users are all gone
	users := &keycloak.KeycloakUserList{}
	err := serverClient.List(ctx, users, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(users.Items) > 0 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Check if clients are all gone
	clients := &keycloak.KeycloakClientList{}
	err = serverClient.List(ctx, clients, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(clients.Items) > 0 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Check if realms are all gone
	realms := &keycloak.KeycloakRealmList{}
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
	}
	if len(realms.Items) > 0 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	kc := &keycloak.Keycloak{}
	// if this errors, it can be ignored
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err == nil && string(r.Config.GetProductVersion()) != kc.Status.Version {
		r.Config.SetProductVersion(kc.Status.Version)
		err = r.ConfigManager.WriteConfig(r.Config)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to write keycloak config: %w", err)
		}
	}

	r.logger.Info("checking ready status for user-sso")
	kcr := &keycloak.KeycloakRealm{}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: masterRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get keycloak realm custom resource: %w", err)
	}

	if kcr.Status.Phase == keycloak.PhaseReconciling {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to write user-sso config: %w", err)
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm for user-sso")
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	r.logger.Infof("user-sso KeycloakRealm status phase is: %s", kcr.Status.Phase)
	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) exportConfig(ctx context.Context, serverClient k8sclient.Client) error {
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return fmt.Errorf("could not retrieve keycloak custom resource for keycloak config for user-sso: %w", err)
	}
	r.Config.SetRealm(masterRealmName)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return fmt.Errorf("could not update keycloak config for user-sso: %w", err)
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, installation *integreatlyv1alpha1.RHMI, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client) error {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return fmt.Errorf("Could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return fmt.Errorf("Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	clientSecret := string(clientSecretBytes)

	oauthc := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			r.Config.GetHost() + "/auth/realms/" + masterRealmName + "/broker/openshift-v4/endpoint",
		},
		GrantMethod: oauthv1.GrantHandlerAuto,
	}
	_, err = r.ReconcileOauthClient(ctx, installation, oauthc, serverClient)
	if err != nil {
		return fmt.Errorf("Could not create OauthClient object for OpenShift IDP: %w", err)
	}

	if !containsIdentityProvider(kcr.Spec.Realm.IdentityProviders, idpAlias) {
		logrus.Infof("Adding keycloak realm client")

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
				"baseUrl":         "https://" + strings.Replace(r.installation.Spec.RoutingSubdomain, "apps", "api", 1) + ":6443",
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

// workaround: the keycloak operator creates a route with TLS passthrough config
// this should use the same valid certs as the cluster itself but for some reason the
// signing operator gives out self signed certs
// to circumvent this we create another keycloak route with edge termination
func (r *Reconciler) createKeycloakRoute(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// We need a route with edge termination to serve the correct cluster certificate
	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, edgeRoute, func() error {
		host := edgeRoute.Spec.Host
		edgeRoute.Spec = routev1.RouteSpec{
			Host: host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "keycloak",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("keycloak"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationReencrypt,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "error creating keycloak edge route")
	}
	r.logger.Info(fmt.Sprintf("operation result of creating %v service was %v", edgeRoute.Name, or))

	if edgeRoute.Spec.Host == "" {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}
	r.logger.Infof("Created Edge route host %s", edgeRoute.Spec.Host)

	// TODO: once the keycloak operator generates a route with a valid certificate, that
	// should be reverted back to using the InternalURL
	r.Config.SetHost(fmt.Sprintf("https://%v", edgeRoute.Spec.Host))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error writing to config in rhssouser reconciler")
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}

func containsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

// Add authenticator config to the master realm. Because it is the master realm we need to make direct calls
// with the Keycloak client. This config allows for the automatic redirect to openshift-v4 as the IDP for Keycloak,
// as apposed to presenting the user with multiple login options.
func (r *Reconciler) reconcileBrowserAuthFlow(ctx context.Context, kc *keycloak.Keycloak, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	kcClient, err := r.keycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	executions, err := kcClient.ListAuthenticationExecutionsForFlow("browser", masterRealmName)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Failed to retrieve execution flows on master realm ")
	}

	executionID := ""
	for _, execution := range executions {
		if execution.ProviderID == "identity-provider-redirector" {
			if execution.AuthenticationConfig != "" {
				r.logger.Infof("Authenticator Config exists on master realm, rhsso-user")
				return integreatlyv1alpha1.PhaseCompleted, nil
			}
			executionID = execution.ID
			break
		}
	}
	if executionID == "" {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Failed to find relevant ProviderID in Authentication Executions")
	}

	config := keycloak.AuthenticatorConfig{Config: map[string]string{"defaultProvider": "openshift-v4"}, Alias: "openshift-v4"}
	err = kcClient.CreateAuthenticatorConfig(&config, masterRealmName, executionID)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Failed ot ")
	}

	r.logger.Infof("Successfully created Authenticator Config")

	return integreatlyv1alpha1.PhaseCompleted, nil
}
