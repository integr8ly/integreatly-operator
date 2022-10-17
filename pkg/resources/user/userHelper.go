package user

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"net/mail"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakTypes "github.com/integr8ly/keycloak-client/pkg/types"
	v1 "github.com/openshift/api/config/v1"
	usersv1 "github.com/openshift/api/user/v1"
	"github.com/pkg/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Helper for User associated functions

const (
	updateProfileAction         = "UPDATE_PROFILE"
	invalidCharacterReplacement = "-"
	GeneratedNamePrefix         = "generated-"
	defaultEmailDomain          = "@rhmi.io"
)

var (
	exclusionGroups = []string{
		"layered-cs-sre-admins",
		"osd-sre-admins",
	}
)

type MultiTenantUser struct {
	Username   string
	TenantName string
	Email      string
	UID        string
}

func GetUserEmailFromIdentity(ctx context.Context, serverClient k8sclient.Client, user usersv1.User, identitiesList usersv1.IdentityList) string {
	email := ""

	// User can have multiple identities
	for _, identityName := range user.Identities {
		for _, identity := range identitiesList.Items {
			if identityName == identity.Name {
				if identity.Extra["email"] != "" {
					return identity.Extra["email"]
				}
			}
		}
	}

	return email
}

func AppendUpdateProfileActionForUserWithoutEmail(keycloakUser *keycloakTypes.KeycloakAPIUser) {
	if keycloakUser.Email == "" {
		keycloakUser.RequiredActions = []string{updateProfileAction}
	}
}

func GetValidGeneratedUserName(keycloakUser keycloakTypes.KeycloakAPIUser) string {
	// Regex for only alphanumeric values
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")

	// Replace all non-alphanumeric values with the replacement character
	processedString := reg.ReplaceAllString(strings.ToLower(keycloakUser.UserName), invalidCharacterReplacement)

	// Remove occurrence of replacement character at end of string
	processedString = strings.TrimSuffix(processedString, invalidCharacterReplacement)

	for _, federatedIdentity := range keycloakUser.FederatedIdentities {
		userId := federatedIdentity.UserID
		// Append user id to name to ensure uniqueness
		if userId != "" {
			processedString = fmt.Sprintf("%s-%s", processedString, federatedIdentity.UserID)
			break
		}
	}

	return fmt.Sprintf("%v%v", GeneratedNamePrefix, processedString)
}

func UserInExclusionGroup(user usersv1.User, groups *usersv1.GroupList) bool {

	// Below is a slightly complex way to determine if the user exists in an exlcusion group
	// Ideally we would use the user.Groups field but this does not seem to get populated.
	for _, group := range groups.Items {
		for _, xGroup := range exclusionGroups {
			if group.Name == xGroup {
				for _, groupUser := range group.Users {
					if groupUser == user.Name {
						return true
					}
				}
			}
		}
	}
	return false
}

// User has no Identity ID on user CR => not an active user.
// User has identity ID on user CR, Identity CR exist and are part of an active IDP => Is an active user.
// User has identity ID on user CR, Identity CR do not exist => Not an active user. Assume the identity CR is associated with a non active IDP. User can log in to rectify
// User has identity ID on user CR, Identity CR exist but is NOT part of an active IDP => not an active user.
func GetUsersInActiveIDPs(ctx context.Context, serverClient k8sclient.Client, logger l.Logger) (*usersv1.UserList, error) {
	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, openshiftUsers)
	if err != nil {
		return nil, errors.Wrap(err, "could not list users")
	}
	idpNames := map[string]bool{}

	oAuths := &v1.OAuthList{}
	err = serverClient.List(ctx, oAuths)
	if err != nil {
		return nil, errors.Wrap(err, "could not list oAuths")
	}

	// get active idp names
	for _, oauth := range oAuths.Items {
		for _, idp := range oauth.Spec.IdentityProviders {
			idpNames[idp.Name] = true
		}
	}

	clusterIdentities := &usersv1.IdentityList{}
	err = serverClient.List(ctx, clusterIdentities)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list cluster identities")
	}

	activeUsers := &usersv1.UserList{}

	for _, user := range openshiftUsers.Items {
		// if user CR lists no identities, move to the next user
		if len(user.Identities) == 0 {
			logger.Info(fmt.Sprintf("user %v has no identities list", user.Name))
			continue
		}
		// get  their identities - can be multiple?
		identities := GetIdentities(user, clusterIdentities)

		// If the identity id on the user does not exist as an identity cr then we have to assume the identity
		// is associated with an invalid IDP. The user can rectify this by logging in to OpenShift. The identity
		// will be recreated.

		for _, identity := range identities.Items {
			//if any identity is provided by an active idp
			if _, ok := idpNames[identity.ProviderName]; ok {
				//add user to return set
				activeUsers.Items = append(activeUsers.Items, user)
				//move to next user - so we don't add a user twice
				break
			}
		}
	}
	return activeUsers, nil
}

func GetIdentities(user usersv1.User, clusterIdentities *usersv1.IdentityList) *usersv1.IdentityList {
	identities := &usersv1.IdentityList{}

	for _, identityName := range user.Identities {
		// find current user identity in list of cluster identities
		identity := getIdentity(identityName, clusterIdentities)
		if identity != nil {
			identities.Items = append(identities.Items, *identity)
		}
	}

	return identities
}

func getIdentity(name string, identities *usersv1.IdentityList) *usersv1.Identity {
	for _, identity := range identities.Items {
		if identity.Name == name {
			return &identity
		}
	}
	return nil
}

func getUsersFromAdminGroups(ctx context.Context, serverClient k8sclient.Client, excludeGroups []string) (*usersv1.UserList, error) {
	adminGroups := &usersv1.GroupList{}
	err := serverClient.List(ctx, adminGroups)
	if err != nil {
		return nil, errors.Wrap(err, "could not list users")
	}

	adminUsers := &usersv1.UserList{}
	for _, adminGroup := range adminGroups.Items {
		if excludeGroup(excludeGroups, adminGroup.Name) {
			for _, user := range adminGroup.Users {
				adminUsers.Items = append(adminUsers.Items, usersv1.User{
					ObjectMeta: metav1.ObjectMeta{Name: user}},
				)
			}
		}
	}

	return adminUsers, nil
}

func excludeGroup(groups []string, group string) bool {
	for _, gr := range groups {
		if group == gr {
			return true
		}
	}
	return false
}

func GetIdentitiesByProviderName(ctx context.Context, serverClient k8sclient.Client, providerName string) (*usersv1.IdentityList, error) {
	identities := &usersv1.IdentityList{}
	err := serverClient.List(ctx, identities)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get identities by provider %s", providerName)
	}

	identitiesByProvider := &usersv1.IdentityList{}
	for _, identity := range identities.Items {
		if identity.ProviderName == providerName {
			identitiesByProvider.Items = append(identitiesByProvider.Items, identity)
		}
	}

	return identitiesByProvider, nil
}

func GetUsersByProviderName(ctx context.Context, serverClient k8sclient.Client, providerName string) (*usersv1.UserList, error) {
	users := &usersv1.UserList{}
	err := serverClient.List(ctx, users)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get users by provider %s", providerName)
	}

	usersByProvider := &usersv1.UserList{}
	for _, user := range users.Items {
		for _, identity := range user.Identities {
			identityName := strings.Split(identity, ":")
			if identityName[0] == providerName {
				usersByProvider.Items = append(usersByProvider.Items, user)
			}
		}
	}

	return usersByProvider, nil
}

func GetMultiTenantUsers(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (users []MultiTenantUser, err error) {
	identities := &usersv1.IdentityList{}
	err = serverClient.List(ctx, identities)
	if err != nil {
		return nil, fmt.Errorf("Error getting identity list")
	}

	usersList := &usersv1.UserList{}
	err = serverClient.List(ctx, usersList)
	if err != nil {
		return nil, fmt.Errorf("Error getting users list")
	}

	for i := range usersList.Items {
		user := usersList.Items[i]
		if isUserHasTenantAnnotation(&user, installation) {
			users = append(users, MultiTenantUser{
				Username:   user.Name,
				TenantName: SanitiseTenantUserName(user.Name),
				Email:      getUserEmail(&user, identities),
				UID:        string(user.UID),
			})
		}
	}

	return users, nil
}

func isUserHasTenantAnnotation(user *usersv1.User, installation *integreatlyv1alpha1.RHMI) bool {
	if user.Annotations == nil {
		return false
	}
	if _, ok := user.Annotations["tenant"]; ok {
		return true
	}
	return false
}

func getUserEmail(user *usersv1.User, identities *usersv1.IdentityList) string {
	var email = ""
	identityForUserFound := false

	for _, identity := range identities.Items {
		if identity.User.Name == user.Name {
			identityForUserFound = true
			if identity.Extra["email"] != "" {
				email = identity.Extra["email"]
			} else {
				email = SetUserNameAsEmail(identity.User.Name)
			}
			break
		}
	}

	if !identityForUserFound {
		email = SetUserNameAsEmail(user.Name)
	}

	return email
}

func SanitiseTenantUserName(username string) string {
	// Regex for only alphanumeric values
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")

	// Replace all non-alphanumeric values with the replacement character
	processedString := reg.ReplaceAllString(strings.ToLower(username), invalidCharacterReplacement)

	// Remove occurrence of replacement character at end of string
	return strings.TrimSuffix(processedString, invalidCharacterReplacement)
}

func SetUserNameAsEmail(userName string) string {
	// If username is a valid email address
	_, err := mail.ParseAddress(userName)
	if err == nil {
		return userName
	}

	// Otherwise sanitise and append default domain
	return fmt.Sprintf("%s%s", SanitiseTenantUserName(userName), defaultEmailDomain)
}
