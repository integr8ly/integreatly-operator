package user

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"net/mail"
	"regexp"
	"strings"

	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
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

func GetEmailFromIdentity(user usersv1.User, identitiesList usersv1.IdentityList) string {

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

	return ""
}

func AppendUpdateProfileActionForUserWithoutEmail(keycloakUser *keycloak.KeycloakAPIUser) {
	if keycloakUser.Email == "" {
		keycloakUser.RequiredActions = []string{updateProfileAction}
	}
}

func GetValidGeneratedUserName(keycloakUser keycloak.KeycloakAPIUser) string {
	// Regex for only alphanumeric values
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		fmt.Printf("failed to compile regex for alphanumeric values with error %v", err)
		return ""
	}

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

func IsInExclusionGroup(user usersv1.User, groups *usersv1.GroupList) bool {

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

func GetMultiTenantUsers(ctx context.Context, serverClient k8sclient.Client) (users []MultiTenantUser, err error) {
	identities := &usersv1.IdentityList{}
	err = serverClient.List(ctx, identities)
	if err != nil {
		return nil, fmt.Errorf("error getting identity list")
	}

	usersList := &usersv1.UserList{}
	err = serverClient.List(ctx, usersList)
	if err != nil {
		return nil, fmt.Errorf("error getting users list")
	}

	for i := range usersList.Items {
		user := usersList.Items[i]
		if hasTenantAnnotation(&user) {
			tenantName, err := SanitiseTenantUserName(user.Name)
			if err != nil {
				return nil, err
			}
			email := getUserEmail(&user, identities)
			if email == "" {
				return nil, err
			}
			users = append(users, MultiTenantUser{
				Username:   user.Name,
				TenantName: tenantName,
				Email:      email,
				UID:        string(user.UID),
			})
		}
	}

	return users, nil
}

func hasTenantAnnotation(user *usersv1.User) bool {
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
	var err error
	identityForUserFound := false

	for _, identity := range identities.Items {
		if identity.User.Name == user.Name {
			identityForUserFound = true
			if identity.Extra["email"] != "" {
				email = identity.Extra["email"]
			} else {
				email, err = SetUserNameAsEmail(identity.User.Name)
				if err != nil {
					return ""
				}
			}
			break
		}
	}

	if !identityForUserFound {
		email, err = SetUserNameAsEmail(user.Name)
		if err != nil {
			return ""
		}
	}

	return email
}

func SanitiseTenantUserName(username string) (string, error) {
	// Regex for only alphanumeric values
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", fmt.Errorf("failed to compile regex for alphanumeric values with error %v", err)
	}

	// Replace all non-alphanumeric values with the replacement character
	processedString := reg.ReplaceAllString(strings.ToLower(username), invalidCharacterReplacement)

	// Remove occurrence of replacement character at end of string
	return strings.TrimSuffix(processedString, invalidCharacterReplacement), nil
}

func SetUserNameAsEmail(userName string) (string, error) {
	// If username is a valid email address
	_, err := mail.ParseAddress(userName)
	if err == nil {
		return userName, nil
	}
	sanitisedUserName, err := SanitiseTenantUserName(userName)
	if err != nil {
		return "", err
	}

	// Otherwise sanitise and append default domain
	return fmt.Sprintf("%s%s", sanitisedUserName, defaultEmailDomain), nil
}
