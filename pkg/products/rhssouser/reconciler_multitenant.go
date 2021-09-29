package rhssouser

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	stagingRealmName         = "staging"
	)

func (r *Reconciler) ReconcileMultiTenantUserSSO(ctx context.Context, serverClient k8sclient.Client, kc *keycloak.Keycloak) (integreatlyv1alpha1.StatusPhase, error) {


	// Get all users
	// Get all realms
	// Get all staging users
	// Create a staging Realm if it doesn't exist
	// Reconcile Config of Events on the Staging Realm inc. Expiration
	// ReconcileStagingUsers: newUsers: users not in staging list
	// Create console link with "Provision SSO Account"

	// GetRecentLogins() Events and return a list of users recently logged in
	// Get added: Logged in users that don't have a realm:
	// Delete added from staging realm
	// ReconcileTenantRealms

	// Delete from staging
	// Delete user from staging
	// Delete realm from staging

	mtUsers, err := userHelper.GetMultiTenantUsers(ctx, serverClient)
	if (err != nil) {
		r.Log.Error("Error getting mt users", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting mt users")
	}
	r.Log.Infof("Found multi tenant users", l.Fields{"num": len(mtUsers)})

	allRealms, err := r.getAllRealms(ctx, serverClient)
	if (err != nil) {
		r.Log.Error("Error getting all Realms", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting all Realms")
	}
	r.Log.Infof("Found realms in user sso", l.Fields{"num": len(allRealms.Items)})

	err = r.reconcileStagingRealm(ctx, serverClient, *allRealms) // including adding events stuff
	if (err != nil) {
		r.Log.Error("Error reconciling staging realm", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error reconciling staging realms")
	}

	existingStagingUsers, err := r.getAllStagingUsers(ctx, serverClient)
	if (err != nil) {
		r.Log.Error("Error getting all staging users", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting all staging users")
	}

	newStagingUsers, err := r.reconcileStagingUsers(ctx, serverClient, mtUsers, existingStagingUsers, allRealms)
	if (err != nil) {
		r.Log.Error("Error getting all staging users", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting all staging users")
	}

	err = r.reconcileStagingConsoleLinks(ctx, serverClient, existingStagingUsers, newStagingUsers)
	if (err != nil) {
		r.Log.Error("Error reconciling Staging Console Links", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error reconciling Staging Console Links")
	}

	loginEvents, err := getLoginEvents(ctx, serverClient)
	if (err != nil) {
		r.Log.Error("Error getting login events", err)
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "Error getting login events")
	}

	reconcileTenantRealms(ctx, serverClient, loginEvents, realms, users)



	//users := []userHelper.MultiTenantUser{}
	//users, err := userHelper.GetMultiTenantUsers(ctx, serverClient)
	//if err != nil {
	//	r.Log.Error("Error getting multi tenant users", err)
	//	return integreatlyv1alpha1.PhaseFailed, nil
	//}
	//
	//r.reconcileTenants(ctx, serverClient, r.Config.GetNamespace(), users, kc)
	//if err != nil {
	//	return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling multi tenant users: %w", err)
	//}


}

func (r *Reconciler) reconcileStagingRealm(ctx context.Context, serverClient k8sclient.Client, realms keycloak.KeycloakRealmList) error {

	if (stagingRealmExists(realms)) {
		return nil
	}

	r.Log.Info("Creating staging realm")
	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stagingRealmName,
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
			"sso": stagingRealmName,
		}

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          stagingRealmName,
			Realm:       stagingRealmName,
			Enabled:     true,
			DisplayName: "*************** SSO Realm Provisioning ******************",
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		redirectURIs := []string{r.Config.GetHost() + "/auth/realms/" + stagingRealmName + "/broker/openshift-v4/endpoint"}
		err := r.SetupOpenshiftIDP(ctx, serverClient, r.Installation, r.Config, kcr, redirectURIs, stagingRealmName)
		if err != nil {
			r.Log.Error("Failed to setup Openshift IDP for user-sso staging realm", err)
			return fmt.Errorf("failed to setup Openshift IDP for user-sso staging realm: %w", err)
		}

		return nil
	})

	if err != nil {
		r.Log.Errorf("Failed create/update", l.Fields{"realm": stagingRealmName}, err)
		return fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloakrealm": kcr.Name, "result": or})

	return nil
}

func stagingRealmExists(realms keycloak.KeycloakRealmList) bool {
	for _, realm := range realms.Items {
		if realm.Name == stagingRealmName {
			return true
		}
	}
	return false
}

func  (r *Reconciler) getAllRealms(ctx context.Context, serverClient k8sclient.Client) (*keycloak.KeycloakRealmList, error) {
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
			"sso": stagingRealmName,
		}),
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		return nil, err
	}
	return &users, nil
}

// Create a user in staging if there is no realm, no existing user but a userCR
func (r *Reconciler) reconcileStagingUsers(ctx context.Context, serverClient k8sclient.Client, mtUsers []userHelper.MultiTenantUser, stgUsers *keycloak.KeycloakUserList, realms *keycloak.KeycloakRealmList) ([]userHelper.MultiTenantUser, error) {
	newUsers := []userHelper.MultiTenantUser{}
	for _, user := range mtUsers {
		if (!realmExistsForMtUser(user, realms.Items) && !stagingUserExists(user, stgUsers)) {
			err := r.createStagingUser(ctx, serverClient, user)
			if (err != nil) {

			}
			newUsers = append(newUsers, user)
		}
	}
	return newUsers, nil
}

func (r *Reconciler) createStagingUser(ctx context.Context, serverClient k8sclient.Client, user userHelper.MultiTenantUser) error {
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.TenantName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"sso" : stagingRealmName,
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
					UserName:         user.TenantName,
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

func stagingUserExists(user userHelper.MultiTenantUser, stgUsers *keycloak.KeycloakUserList) bool {
	for _, stgUser := range stgUsers.Items {
		if stgUser.Name == user.TenantName {
			return true
		}
	}
	return false
}

func (r *Reconciler) reconcileStagingConsoleLinks(ctx context.Context, serverClient k8sclient.Client, existingUsers *keycloak.KeycloakUserList, newUsers []userHelper.MultiTenantUser) error {

	stagingLink := r.Config.GetHost() + "/auth/admin/staging/console/"

	for _, user := range existingUsers.Items {
		tenantNs := user.Name + "-dev"
		if err := r.reconcileDashboardLink(ctx, serverClient, user.Name, stagingLink, tenantNs); err != nil {
			return err
		}
	}

	for _, user := range newUsers {
		tenantNs := user.TenantName + "-dev"
		if err := r.reconcileDashboardLink(ctx, serverClient, user.TenantName, stagingLink, tenantNs); err != nil {
			return err
		}
	}
	return nil
}
