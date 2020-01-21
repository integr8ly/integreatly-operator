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
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
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
	keycloakRealmName       = "user-sso"
	defaultSubscriptionName = "integreatly-rhsso"
	idpAlias                = "openshift-v4"
	manifestPackage         = "integreatly-rhsso"
)

const (
	userSsoLabelKey       = "sso"
	userSsoLabelValue     = "user"
	masterRealmLabelKey   = "master"
	masterRealmLabelValue = "master"
)

type Reconciler struct {
	Config        *config.RHSSOUser
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.Installation
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	ApiUrl        string
	*resources.Reconciler
	recorder              record.EventRecorder
	keycloakClientFactory keycloakCommon.KeycloakClientFactory
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, apiUrl string, keycloakClientFactory keycloakCommon.KeycloakClientFactory) (*Reconciler, error) {

	rhssoUserConfig, err := configManager.ReadRHSSOUser()
	if err != nil {
		return nil, err
	}
	if rhssoUserConfig.GetNamespace() == "" {
		rhssoUserConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultRhssoNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:                rhssoUserConfig,
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
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.Installation, product *integreatlyv1alpha1.InstallationProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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
		err = resources.RemoveOauthClient(ctx, installation, serverClient, r.oauthv1Client, r.getOAuthClientName("master"))
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		err = resources.RemoveOauthClient(ctx, installation, serverClient, r.oauthv1Client, r.getOAuthClientName("master"))
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		err = resources.RemoveOauthClient(ctx, installation, serverClient, r.oauthv1Client, r.getOAuthClientName(keycloakRealmName))
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

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.createKeycloakRoute(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, installation, serverClient)
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

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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
		kc.Labels = getInstanceLabels()
		kc.Spec.Instances = 3
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{Enabled: true}
		kc.Spec.Profile = rhsso.RHSSOProfile
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	r.logger.Infof("The operation result for keycloak %s was %s", kc.Name, or)

	host := kc.Status.InternalURL
	// The host needs to be available in order to create the openshift IDP on the realms
	if host == "" {
		r.logger.Infof("Internal URL for Keycloak not yet available")
		return integreatlyv1alpha1.PhaseAwaitingComponents, fmt.Errorf("Internal URL for Keyclak not yet available")
	}

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
			MatchLabels: getInstanceLabels(),
		}
		kcr.Labels = getInstanceLabels()
		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:              keycloakRealmName,
			Realm:           keycloakRealmName,
			Enabled:         true,
			DisplayName:     keycloakRealmName,
			EventsListeners: []string{"metrics-listener"},
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		err = r.setupOpenshiftIDP(ctx, installation, kcr, serverClient, keycloakRealmName, host)
		if err != nil {
			return errors.Wrap(err, "failed to setup Openshift IDP for user-sso")
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	// We want to update the master realm by adding an openshift-v4 idp. We can not add the idp until we know the host
	// url - kc.Status.InternalURL
	if kc.Status.InternalURL == "" {
		logrus.Warningf("Can not update keycloak master realm on user sso as internal URL is not available yet")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak master realm, internal URL not available")
	}

	// Create the master realm. The master real already exists in Keycloak but we need to get a reference to it
	// in order to create the IDP and admin users on it
	masterKcr, err := r.updateMasterRealm(ctx, serverClient, installation, kc.Status.InternalURL)
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
		or, err = r.createOrUpdateKeycloakAdmin(user, installation, ctx, serverClient)
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
func (r *Reconciler) updateMasterRealm(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.Installation, host string) (*keycloak.KeycloakRealm, error) {
	keycloakMasterRealmName := "master"

	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakMasterRealmName,
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
			MatchLabels: getInstanceLabels(),
		}

		kcr.Labels = getMasterLabels()

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          keycloakMasterRealmName,
			Realm:       keycloakMasterRealmName,
			Enabled:     true,
			DisplayName: keycloakMasterRealmName,
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		err := r.setupOpenshiftIDP(ctx, installation, kcr, serverClient, "master", host)
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

func (r *Reconciler) createOrUpdateKeycloakAdmin(user keycloak.KeycloakAPIUser, inst *integreatlyv1alpha1.Installation, ctx context.Context, serverClient k8sclient.Client) (controllerutil.OperationResult, error) {
	selector := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("generated-%v", user.UserName),
			Namespace: r.Config.GetNamespace(),
		},
	}

	return controllerutil.CreateOrUpdate(ctx, serverClient, selector, func() error {
		ownerutil.EnsureOwner(selector, inst)
		selector.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: getMasterLabels(),
		}
		selector.Labels = GetInstanceLabels()
		selector.Spec.User = user

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

func GetInstanceLabels() map[string]string {
	return map[string]string{
		userSsoLabelKey: userSsoLabelValue,
	}
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

	openshiftAdminGroup := &usersv1.Group{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: "dedicated-admins"}, openshiftAdminGroup)
	if err != nil {
		return nil, err
	}

	added, deletedIndexes, deletedUsers := getUserDiff(keycloakUsers, openshiftUsers.Items, openshiftAdminGroup.Users)

	for index := range deletedIndexes {
		keycloakUsers = remove(index, keycloakUsers)
	}

	err = deleteKeycloakUsers(deletedUsers, ns, ctx, serverClient)
	if err != nil {
		return nil, err
	}

	for _, osUser := range added {

		if !isOpenshiftAdmin(osUser, openshiftAdminGroup) {
			continue
		}

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
				},
			},
		})
	}

	return keycloakUsers, nil
}

func deleteKeycloakUsers(deletedUsers []keycloak.KeycloakAPIUser, ns string, ctx context.Context, serverClient k8sclient.Client) error {

	var allUsers keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(GetInstanceLabels()),
		k8sclient.InNamespace(ns),
	}

	err := serverClient.List(ctx, &allUsers, listOptions...)
	if err != nil {
		return err
	}

	for _, delUser := range deletedUsers {

		for _, user := range allUsers.Items {
			if user.Spec.User.UserName == delUser.UserName {
				err = serverClient.Delete(ctx, &user)
				if err != nil {
					return fmt.Errorf("failed to delete keycloak user: %w", err)
				}
				break
			}
		}
	}
	return nil
}

func getUserDiff(keycloakUsers []keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User, dedicatedAdmins []string) ([]usersv1.User, []int, []keycloak.KeycloakAPIUser) {
	var added []usersv1.User
	for _, osUser := range openshiftUsers {
		if !kcContainsOsUser(keycloakUsers, osUser) {
			added = append(added, osUser)
		}
	}

	// They are considered deleted if they are removed from OpenShift or removed from the dedicated admin group
	var deletedIndexes []int
	var deletedUsers []keycloak.KeycloakAPIUser
	for i, kcUser := range keycloakUsers {
		if !OsUserInKc(openshiftUsers, kcUser) {
			deletedIndexes = append(deletedIndexes, i)
			deletedUsers = append(deletedUsers, kcUser)
		}
		if !OsUserInDedicatedAdmins(dedicatedAdmins, kcUser) {
			deletedIndexes = append(deletedIndexes, i)
			deletedUsers = append(deletedUsers, kcUser)
		}
	}

	return added, deletedIndexes, deletedUsers
}

func OsUserInDedicatedAdmins(dedicatedAdmins []string, kcUser keycloak.KeycloakAPIUser) bool {
	for _, user := range dedicatedAdmins {
		if kcUser.UserName == user {
			return true
		}
	}
	return false
}

func kcContainsOsUser(kcUsers []keycloak.KeycloakAPIUser, osUser usersv1.User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == osUser.Name {
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

func remove(index int, kcUsers []keycloak.KeycloakAPIUser) []keycloak.KeycloakAPIUser {
	kcUsers[index] = kcUsers[len(kcUsers)-1]
	return kcUsers[:len(kcUsers)-1]
}

func isOpenshiftAdmin(user usersv1.User, adminGroup *usersv1.Group) bool {
	for _, name := range adminGroup.Users {
		if user.Name == name {
			return true
		}
	}
	return false
}

func (r *Reconciler) cleanupKeycloakResources(ctx context.Context, inst *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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

func (r *Reconciler) handleProgressPhase(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
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
	r.Config.SetRealm(keycloakRealmName)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return fmt.Errorf("could not update keycloak config for user-sso: %w", err)
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, installation *integreatlyv1alpha1.Installation, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client, realm string, host string) error {
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
			Name: r.getOAuthClientName(realm),
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			host + "/auth/realms/" + realm + "/broker/openshift-v4/endpoint",
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
				"clientId":        r.getOAuthClientName(realm),
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

	r.Config.SetHost(fmt.Sprintf("https://%v", edgeRoute.Spec.Host))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error writing to config in rhssouser reconciler")
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getOAuthClientName(realm string) string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName()) + "-" + realm
}

func containsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

func getInstanceLabels() map[string]string {
	return map[string]string{
		userSsoLabelKey: userSsoLabelValue,
	}
}
