package rhssouser

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	kccommon "github.com/integr8ly/keycloak-client/pkg/common"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const StagingRealmName = "staging"

func (r *Reconciler) ReconcileMultiTenantUserSSO(ctx context.Context, serverClient k8sclient.Client, kc *keycloak.Keycloak) (integreatlyv1alpha1.StatusPhase, error) {

	mtUsers, err := userHelper.GetMultiTenantUsers(ctx, serverClient)
	if err != nil {
		r.Log.Error("Error getting mt users", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting mt users")
	}
	r.Log.Infof("Found multi tenant users", l.Fields{"num": len(mtUsers)})

	allRealms, err := r.getAllRealms(ctx, serverClient)
	if err != nil {
		r.Log.Error("Error getting all Realms", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting all Realms")
	}
	r.Log.Infof("Found realms in user sso", l.Fields{"num": len(allRealms.Items)})

	err = r.reconcileStagingRealm(ctx, serverClient, *allRealms)
	if err != nil {
		r.Log.Error("Error reconciling staging realm", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error reconciling staging realms")
	}

	existingStagingUsers, err := r.getAllStagingUsers(ctx, serverClient)
	if err != nil {
		r.Log.Error("Error getting all staging users", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting all staging users")
	}

	newStagingUsers, err := r.reconcileStagingUsers(ctx, serverClient, mtUsers, existingStagingUsers, allRealms)
	if err != nil {
		r.Log.Error("Error getting all staging users", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting all staging users")
	}

	err = r.reconcileStagingUsersConsoleLink(ctx, serverClient, existingStagingUsers, newStagingUsers)
	if err != nil {
		r.Log.Error("Error reconciling Staging Console Links", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error reconciling Staging Console Links")
	}

	loginEventUserIds, err := r.getUserIdLoginEvents(ctx, serverClient, kc)
	if err != nil {
		r.Log.Error("Error getting login events", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting login events")
	}

	err = r.reconcileTenantRealms(ctx, serverClient, allRealms, mtUsers, loginEventUserIds)
	if err != nil {
		r.Log.Error("Error reconcileTenantRealms", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error reconcileTenantRealms")
	}

	err = r.reconcileTenantConsoleLinks(ctx, serverClient, mtUsers)
	if err != nil {
		r.Log.Error("Error reconciling Tenant Console Links", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error reconciling Staging Console Links")
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTenantConsoleLinks(ctx context.Context, serverClient k8sclient.Client, users []userHelper.MultiTenantUser) error {
	realms, err := r.getAllRealms(ctx, serverClient)
	if err != nil {
		r.Log.Error("Error getting all realms", err)
		return err
	}

	for _, realm := range realms.Items {
		if realm.Name == "master" || realm.Name == StagingRealmName {
			continue
		}
		tenantLink := r.Config.GetHost() + "/auth/admin/" + realm.Name + "/console/"
		if err := r.reconcileDashboardLink(ctx, serverClient, realm.Name, tenantLink, ""); err != nil {
			return err
		}
	}

	return nil
}

// If a user logged in to the staging realm but does not have a tenant realm, create one.
func (r *Reconciler) reconcileTenantRealms(ctx context.Context, serverClient k8sclient.Client, realms *keycloak.KeycloakRealmList, users []userHelper.MultiTenantUser, loginEventUserIds []kccommon.Users) error {

	added, deleted, err := r.getTenantDiff(ctx, serverClient, users, realms.Items, loginEventUserIds)
	if err != nil {
		return err
	}

	for _, user := range added {
		_, err := r.createTenantRealm(ctx, serverClient, user.TenantName)
		if err != nil {
			return err
		}
		_, err = r.createTenantKCUser(ctx, serverClient, user)
		if err != nil {
			return err
		}
		if err = r.removeUserFromStagingRealm(ctx, serverClient, user); err != nil {
			return err
		}
	}

	// delete all tenant details
	for _, realm := range deleted {
		_, err = r.deleteTenant(ctx, serverClient, realm)
		if err != nil {
			r.Log.Errorf("Failed to delete tenant on realm ", l.Fields{"realm": realm.Name}, err)
			return fmt.Errorf("failed to delete tenant on realm: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) removeUserFromStagingRealm(ctx context.Context, serverClient k8sclient.Client, user userHelper.MultiTenantUser) error {
	kcUser, err := r.getKeycloakUserCRByUID(ctx, serverClient, user.UID)
	if err != nil {
		r.Log.Errorf("Error finding staging user", l.Fields{"uid": user.UID}, err)
		return errors.Wrap(err, "Error finding staging user")
	}
	if kcUser == nil {
		r.Log.Warningf("User not found in staging realm to delete ", l.Fields{"uid": user.UID})
		return nil
	}
	err = serverClient.Delete(ctx, kcUser)
	if err != nil {
		r.Log.Errorf("Failed to delete kcUser ", l.Fields{"kcUser": kcUser.Name}, err)
		return fmt.Errorf("failed to delete tenant kcUser: %w", err)
	}
	return nil
}

func (r *Reconciler) getTenantDiff(ctx context.Context, serverClient k8sclient.Client, mtUsers []userHelper.MultiTenantUser, realms []keycloak.KeycloakRealm, loginEventUserIds []kccommon.Users) (added []userHelper.MultiTenantUser, deleted []keycloak.KeycloakRealm, err error) {

	for _, loginEventUserId := range loginEventUserIds {
		mtUser, err := r.findUser(ctx, serverClient, loginEventUserId, mtUsers)
		if err != nil {
			r.Log.Error("Error finding user", err)
			return nil, nil, errors.Wrap(err, "Error finding user")
		}
		if mtUser == nil {
			r.Log.Warningf("User not found ", l.Fields{"loginEventUserId": loginEventUserId})
			continue
		}
		if !realmExistsForTenant(realms, *mtUser) && !userInList(added, *mtUser) {
			added = append(added, *mtUser)
		}
	}

	// Find realms with no corresponding user. Delete it.
	for _, realm := range realms {
		if realm.Name == "master" || realm.Name == StagingRealmName {
			continue
		}
		if !userExistsForRealm(mtUsers, realm) {
			deleted = append(deleted, realm)
		}
	}

	return added, deleted, nil
}

func userInList(added []userHelper.MultiTenantUser, user userHelper.MultiTenantUser) bool {
	for _, usr := range added {
		if usr.UID == user.UID {
			return true
		}
	}
	return false
}

// The loginEventUserId is the keycloak user id.
// We can use this to find the KCUser CR and on that there is a UID
// representing the user id on the user CR
func (r *Reconciler) findUser(ctx context.Context, serverClient k8sclient.Client, loginEventUserId kccommon.Users, users []userHelper.MultiTenantUser) (*userHelper.MultiTenantUser, error) {
	kucr, err := r.getKeycloakUserCR(ctx, serverClient, loginEventUserId)
	if err != nil {
		r.Log.Error("Error getting keylcoak user cr", err)
		return nil, err
	}
	if kucr == nil {
		return nil, nil
	}
	uid := kucr.Spec.User.FederatedIdentities[0].UserID
	for _, user := range users {
		if (user.UID) == uid {
			return &user, nil
		}
	}
	return nil, nil
}

func (r *Reconciler) getKeycloakUserCR(ctx context.Context, serverClient k8sclient.Client, User kccommon.Users) (*keycloak.KeycloakUser, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"sso": StagingRealmName,
		}),
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		r.Log.Error("Error getting keylcoak user list", err)
		return nil, err
	}

	for _, user := range users.Items {
		if user.Spec.User.ID == User.UserID {
			return &user, nil
		}
	}

	r.Log.Warningf("Keycloak user not found", l.Fields{"UserId": User.UserID})

	return nil, nil
}

func (r *Reconciler) getKeycloakUserCRByUID(ctx context.Context, serverClient k8sclient.Client, UserId string) (*keycloak.KeycloakUser, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"sso": StagingRealmName,
		}),
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		r.Log.Error("Error getting keylcoak user list", err)
		return nil, err
	}

	for _, user := range users.Items {
		if user.Spec.User.FederatedIdentities[0].UserID == UserId {
			return &user, nil
		}
	}

	r.Log.Warningf("Keycloak user not found", l.Fields{"UserId": UserId})

	return nil, nil
}

func (r *Reconciler) getUserIdLoginEvents(ctx context.Context, serverClient k8sclient.Client, kc *keycloak.Keycloak) ([]kccommon.Users, error) {

	kcClient, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return nil, err
	}
	return kcClient.ListOfActivesUsersPerRealm(StagingRealmName, "", 1000)
}

func (r *Reconciler) reconcileStagingRealm(ctx context.Context, serverClient k8sclient.Client, realms keycloak.KeycloakRealmList) error {

	r.Log.Info("Reconciling staging realm")
	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StagingRealmName,
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

		kcr.Labels = map[string]string{
			"sso": StagingRealmName,
		}

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:            StagingRealmName,
			Realm:         StagingRealmName,
			Enabled:       true,
			DisplayName:   "*************** SSO Realm Provisioning ******************",
			EventsEnabled: boolPtr(true),
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		redirectURIs := []string{r.Config.GetHost() + "/auth/realms/" + StagingRealmName + "/broker/openshift-v4/endpoint"}
		err := r.SetupOpenshiftIDP(ctx, serverClient, r.Installation, r.Config, kcr, redirectURIs, StagingRealmName)
		if err != nil {
			r.Log.Error("Failed to setup Openshift IDP for user-sso staging realm", err)
			return fmt.Errorf("failed to setup Openshift IDP for user-sso staging realm: %w", err)
		}

		return nil
	})

	if err != nil {
		r.Log.Errorf("Failed create/update", l.Fields{"realm": StagingRealmName}, err)
		return fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloakrealm": kcr.Name, "result": or})

	return nil
}

func (r *Reconciler) getAllRealms(ctx context.Context, serverClient k8sclient.Client) (*keycloak.KeycloakRealmList, error) {
	realms := &keycloak.KeycloakRealmList{}

	listOptions := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := serverClient.List(ctx, realms, listOptions...)
	return realms, err
}

func (r *Reconciler) getAllStagingUsers(ctx context.Context, serverClient k8sclient.Client) (*keycloak.KeycloakUserList, error) {

	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"sso": StagingRealmName,
		}),
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		return nil, err
	}
	return &users, nil
}

func stagingUserExists(user userHelper.MultiTenantUser, stgUsers *keycloak.KeycloakUserList) bool {
	for _, stgUser := range stgUsers.Items {
		if stgUser.Spec.User.FederatedIdentities[0].UserID == user.UID {
			return true
		}
	}
	return false
}

// Create a user in staging if there is no realm, no existing user but a userCR
func (r *Reconciler) reconcileStagingUsers(ctx context.Context, serverClient k8sclient.Client, mtUsers []userHelper.MultiTenantUser, stgUsers *keycloak.KeycloakUserList, realms *keycloak.KeycloakRealmList) ([]userHelper.MultiTenantUser, error) {
	newUsers := []userHelper.MultiTenantUser{}
	for _, user := range mtUsers {
		if !realmExistsForMtUser(user, realms.Items) && !stagingUserExists(user, stgUsers) {
			err := r.createStagingUser(ctx, serverClient, user)
			if err != nil {
				r.Log.Error("Error creating staging user", err)
				return newUsers, errors.Wrap(err, "Error creating staging user")
			}
			newUsers = append(newUsers, user)
		}
	}

	// If a staging user exists but a user CR does not exist, then delete the staging user
	for _, stgUser := range stgUsers.Items {
		uid := stgUser.Spec.User.FederatedIdentities[0].UserID
		if uid == "" {
			r.Log.Warningf("user id on staging keycloak user not found", l.Fields{"stgUser": stgUser.Name})
		} else if !userExsitsForStagingUser(uid, mtUsers) {
			err := serverClient.Delete(ctx, &stgUser)
			if err != nil {
				r.Log.Errorf("Failed to delete staging user ", l.Fields{"stgUser": stgUser.Name}, err)
				return newUsers, fmt.Errorf("failed to delete keycloak user: %w", err)
			}
		}
	}

	return newUsers, nil
}

func userExsitsForStagingUser(uid string, mtUsers []userHelper.MultiTenantUser) bool {
	for _, mtUser := range mtUsers {
		if mtUser.UID == uid {
			return true
		}
	}
	return false
}

func (r *Reconciler) createStagingUser(ctx context.Context, serverClient k8sclient.Client, user userHelper.MultiTenantUser) error {
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.TenantName + "-stg",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"sso": StagingRealmName,
			},
		}
		kcUser.Labels = map[string]string{
			"sso": StagingRealmName,
		}
		kcUser.Spec.User = keycloak.KeycloakAPIUser{
			Enabled:       true,
			UserName:      user.TenantName,
			EmailVerified: true,
			Email:         user.Email,
			FederatedIdentities: []keycloak.FederatedIdentity{
				{
					IdentityProvider: idpAlias,
					UserID:           string(user.UID),
					UserName:         user.Username,
				},
			},
			ClientRoles: getStagingClientRoles("realm-management"),
			RealmRoles:  []string{"offline_access", "uma_authorization"},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update keycloak tenant user: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloak user": kcUser.Name, "res": or})

	return nil
}

func getStagingClientRoles(realm string) map[string][]string {
	return map[string][]string{
		realm: {
			"view-realm",
		},
	}
}

func realmExistsForMtUser(user userHelper.MultiTenantUser, realms []keycloak.KeycloakRealm) bool {
	for _, realm := range realms {
		if realm.Name == user.TenantName {
			return true
		}
	}
	return false
}

func (r *Reconciler) reconcileStagingUsersConsoleLink(ctx context.Context, serverClient k8sclient.Client, existingUsers *keycloak.KeycloakUserList, newUsers []userHelper.MultiTenantUser) error {

	stagingLink := r.Config.GetHost() + "/auth/admin/staging/console/"

	for _, user := range existingUsers.Items {
		if err := r.reconcileDashboardLink(ctx, serverClient, user.Spec.User.UserName, stagingLink, "Provision SSO Realm"); err != nil {
			return err
		}
	}

	for _, user := range newUsers {
		if err := r.reconcileDashboardLink(ctx, serverClient, user.TenantName, stagingLink, "Provision SSO Realm"); err != nil {
			return err
		}
	}
	return nil
}

func boolPtr(value bool) *bool {
	return &value
}

func realmExistsForTenant(realms []keycloak.KeycloakRealm, user userHelper.MultiTenantUser) bool {
	for _, realm := range realms {
		if realm.Name == user.TenantName {
			return true
		}
	}
	return false
}

func userExistsForRealm(users []userHelper.MultiTenantUser, realm keycloak.KeycloakRealm) bool {
	for _, user := range users {
		if realm.Name == user.TenantName {
			return true
		}
	}
	return false
}

func (r *Reconciler) createTenantKCUser(ctx context.Context, serverClient k8sclient.Client, user userHelper.MultiTenantUser) (integreatlyv1alpha1.StatusPhase, error) {

	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.TenantName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				masterRealmLabelKey: user.TenantName,
			},
		}
		kcUser.Labels = map[string]string{
			"sso": user.TenantName,
		}
		kcUser.Spec.User = keycloak.KeycloakAPIUser{
			Enabled:       true,
			UserName:      user.TenantName,
			EmailVerified: true,
			Email:         user.Email,
			FederatedIdentities: []keycloak.FederatedIdentity{
				{
					IdentityProvider: idpAlias,
					UserID:           string(user.UID),
					UserName:         user.Username,
				},
			},
			ClientRoles: getTenantClientRoles("realm-management"),
			RealmRoles:  []string{"offline_access", "uma_authorization"},
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak tenant user: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloak user": kcUser.Name, "res": or})

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getTenantClientRoles(realm string) map[string][]string {
	return map[string][]string{
		realm: {
			"create-client",
			"manage-authorization",
			"manage-clients",
			"manage-events",
			"manage-identity-providers",
			"manage-realm",
			"manage-users",
			"query-clients",
			"query-groups",
			"query-realms",
			"query-users",
			"view-authorization",
			"view-clients",
			"view-events",
			"view-identity-providers",
			"view-realm",
			"view-users",
		},
	}
}

func (r *Reconciler) createTenantRealm(ctx context.Context, serverClient k8sclient.Client, username string) (integreatlyv1alpha1.StatusPhase, error) {
	r.Log.Infof("Creating tenant realm", l.Fields{"tenant": username})
	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      username,
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

		kcr.Labels = map[string]string{
			"sso": username,
		}

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          username,
			Realm:       username,
			Enabled:     true,
			DisplayName: username,
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		redirectURIs := []string{r.Config.GetHost() + "/auth/realms/" + username + "/broker/openshift-v4/endpoint"}
		err := r.SetupOpenshiftIDP(ctx, serverClient, r.Installation, r.Config, kcr, redirectURIs, username)
		if err != nil {
			return fmt.Errorf("failed to setup Openshift IDP for user-sso: %w", err)
		}

		return nil
	})

	if err != nil {
		r.Log.Errorf("Failed create/update", l.Fields{"realm": username}, err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloakrealm": kcr.Name, "result": or})

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTenantDashboardLinks(ctx context.Context, serverClient k8sclient.Client) error {

	users := []userHelper.MultiTenantUser{}
	users, err := userHelper.GetMultiTenantUsers(ctx, serverClient)
	if err != nil {
		r.Log.Error("Error getting multi tenant users", err)
		return err
	}

	for _, user := range users {
		tenantLink := r.Config.GetHost() + "/auth/admin/" + user.TenantName + "/console/"
		if err := r.reconcileDashboardLink(ctx, serverClient, user.TenantName, tenantLink, ""); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileDashboardLink(ctx context.Context, serverClient k8sclient.Client, username string, tenantLink string, text string) error {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: username + "-usersso",
		},
	}

	if text == "" {
		text = "API Management SSO"
	}

	tenantNamespaces := []string{fmt.Sprintf("%s-stage", username), fmt.Sprintf("%s-dev", username)}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cl, func() error {
		cl.Spec = consolev1.ConsoleLinkSpec{
			Location: consolev1.NamespaceDashboard,
			Link: consolev1.Link{
				Href: tenantLink,
				Text: text,
			},
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: tenantNamespaces,
			},
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error reconciling console link: %v", err)
	}

	return nil
}

func (r *Reconciler) deleteTenant(ctx context.Context, serverClient k8sclient.Client, realm keycloak.KeycloakRealm) (integreatlyv1alpha1.StatusPhase, error) {

	// Delete the Keycloak User
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      realm.Name,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Delete(ctx, kcUser)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete keycloak user: %w", err)
	}

	err = serverClient.Delete(ctx, &realm)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := resources.RemoveOauthClient(r.Oauthv1Client, realm.Name, r.Log); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := r.deleteConsoleLink(ctx, serverClient, realm.Name); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
