package user

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"net/mail"
	"os"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
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

func GetUserEmailFromIdentity(ctx context.Context, serverClient k8sclient.Client, user usersv1.User) (string, error) {
	email := ""

	// User can have multiple identities
	for _, identityName := range user.Identities {
		identity := &usersv1.Identity{}
		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: identityName}, identity)

		if err != nil {
			return "", fmt.Errorf("failed to get identity provider: %w", err)
		}

		// Get first identity with email and break loop
		if identity.Extra["email"] != "" {
			email = identity.Extra["email"]
			break
		}
	}

	return email, nil
}

func AppendUpdateProfileActionForUserWithoutEmail(keycloakUser *keycloak.KeycloakAPIUser) {
	if keycloakUser.Email == "" {
		keycloakUser.RequiredActions = []string{updateProfileAction}
	}
}

func GetValidGeneratedUserName(keycloakUser keycloak.KeycloakAPIUser) string {
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

func GetUsersInActiveIDPs(ctx context.Context, serverClient k8sclient.Client) (*usersv1.UserList, error) {
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

	activeUsers := &usersv1.UserList{}

	//go over each user
	for _, user := range openshiftUsers.Items {
		// get  their identities - can be multiple?
		identities, err := GetIdentities(ctx, serverClient, user)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get identities for user %v", user.Name)
		}

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

func GetIdentities(ctx context.Context, serverClient k8sclient.Client, user usersv1.User) (*usersv1.IdentityList, error) {
	identities := &usersv1.IdentityList{}

	for _, identityName := range user.Identities {
		identity := &usersv1.Identity{}
		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: identityName}, identity)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get identity %v for user %v", identityName, user.Name)
		}
		identities.Items = append(identities.Items, *identity)
	}
	return identities, nil
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

func GetMultiTenantUsers(ctx context.Context, serverClient k8sclient.Client) (users []MultiTenantUser, err error) {
	requiredIdp, err := GetIdpName()
	if err != nil {
		return nil, err
	}

	identities, err := GetIdentitiesByProviderName(ctx, serverClient, requiredIdp)
	if err != nil {
		return nil, fmt.Errorf("Error getting identity list for multi tenants")
	}
	for _, identity := range identities.Items {

		var email = ""
		if identity.Extra["email"] != "" {
			email = identity.Extra["email"]
		} else {
			email = SetUserNameAsEmail(identity.User.Name)
		}

		users = append(users, MultiTenantUser{
			Username:   identity.User.Name,
			TenantName: SanitiseTenantUserName(identity.User.Name),
			Email:      email,
			UID:        string(identity.User.UID),
		})
	}
	return users, nil
}

func GetMultiTenantUsersCount(ctx context.Context, serverClient k8sclient.Client, log l.Logger) (int, error) {
	requiredIdp, err := GetIdpName()
	if err != nil {
		return 0, fmt.Errorf("error when pulling IDP name from the envvar")
	}

	log.Infof("Looking for identities from", l.Fields{"idp": requiredIdp})

	identities, err := GetIdentitiesByProviderName(ctx, serverClient, requiredIdp)
	if err != nil {
		return 0, fmt.Errorf("Error getting identity list for multi tenants")
	}

	return len(identities.Items), nil
}

func SanitiseTenantUserName(username string) string {
	// Regex for only alphanumeric values
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")

	// Replace all non-alphanumeric values with the replacement character
	processedString := reg.ReplaceAllString(strings.ToLower(username), invalidCharacterReplacement)

	// Remove occurrence of replacement character at end of string
	return strings.TrimSuffix(processedString, invalidCharacterReplacement)
}

func GetIdpName() (string, error) {
	idpName, ok := os.LookupEnv("IDENTITY_PROVIDER_NAME")
	if ok != true {
		return "devsandbox", fmt.Errorf("error retrieving IDENTITY_PROVIDER_NAME from envar")
	}

	return idpName, nil
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
