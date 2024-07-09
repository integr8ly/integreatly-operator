package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	clientIDAdminCLI                      = "admin-cli"
	grantTypePassword                     = "password"
	grantTypeRefreshToken                 = "refresh_token"
	groupNameDedicatedAdmins              = "dedicated-admins"
	groupNameDedicatedAdminsRealmManagers = "realm-managers"
	groupNameRHMIDevelopers               = "rhmi-developers"
	realmNameMaster                       = "master"
	testUserName                          = "test-user99"
)

var (
	testUser = &userv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: testUserName,
		},
	}
	tokens          *keycloakOpenIDTokenResponse
	tokenExpiryTime time.Time
)

func TestDedicatedAdminUsersSyncedSSO(t TestingTB, tc *TestingContext) {
	ctx := context.Background()
	defer cleanUpTestDedicatedAdminUsersSyncedSSO(ctx, t, tc.Client)

	// Create Testing IDP
	if err := createTestingIDP(t, ctx, tc.Client, tc.KubeConfig, tc.SelfSignedCerts); err != nil {
		t.Fatalf("error creating testing IDP: %v", err)
	}

	// Create OpenShift user
	if err := tc.Client.Create(ctx, testUser); err != nil {
		t.Fatalf("error creating openshift user: %v", err)
	}
	t.Logf("openShift user %s created", testUser.Name)

	// Authenticate with Keycloak
	hostKeycloakUserSSO, err := getHostKeycloak(ctx, tc.Client, RHSSOUserProductNamespace)
	if err != nil {
		t.Fatalf("%v", err)
	}
	credentialsKeycloak, err := getCredentialsRHSSOUser(ctx, tc.Client)
	if err != nil {
		t.Fatalf("%v", err)
	}
	tokenOptions := keycloakTokenOptions{
		ClientID:  pointer.String(clientIDAdminCLI),
		GrantType: pointer.String(grantTypePassword),
		RealmName: realmNameMaster,
		Username:  pointer.String(credentialsKeycloak[0]),
		Password:  pointer.String(credentialsKeycloak[1]),
	}
	timeBeforeTokenReq := time.Now()
	tokens, err = getKeycloakToken(tc.HttpClient, hostKeycloakUserSSO, tokenOptions)
	if err != nil {
		t.Fatalf("%v", err)
	}
	tokenExpiryTime = timeBeforeTokenReq.Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Create a testing user in Keycloak (user-sso)
	keycloakUserToCreate := keycloakUser{
		EmailVerified: pointer.Bool(true),
		Enabled:       pointer.Bool(true),
		FirstName:     pointer.String("Test User"),
		LastName:      pointer.String("99"),
		UserName:      pointer.String(testUserName),
	}
	if err := createKeycloakUser(tc.HttpClient, hostKeycloakUserSSO, realmNameMaster, tokens.AccessToken, keycloakUserToCreate); err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("created Keycloak user with username: %s", testUserName)

	// Add the test user to the dedicated-admins group
	if err := createOrUpdateDedicatedAdminGroupCR(ctx, tc.Client, []string{testUserName}); err != nil {
		t.Fatalf("failed to add user %s to group %s: %v", testUserName, groupNameDedicatedAdmins, err)
	}
	t.Logf("added user %s to group %s", testUserName, groupNameDedicatedAdmins)

	// Wait for RHOAM to reconcile the KeycloakUser CR in user-sso namespace and verify its group memberships
	if err := pollGeneratedKeycloakUserCR(ctx, tc.Client); err != nil {
		t.Fatalf("%v", err)
	}

	tokens, err = refreshKeycloakToken(tc.HttpClient, hostKeycloakUserSSO, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("%v", err)
	}
	userOptions := keycloakUserOptions{
		RealmName: realmNameMaster,
		Username:  pointer.String(testUserName),
	}
	keycloakUsers, err := getKeycloakUsers(tc.HttpClient, hostKeycloakUserSSO, tokens.AccessToken, userOptions)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Poll for the matching Keycloak user and verify its group memberships
	if err := pollKeycloakUserGroups(tc.HttpClient, hostKeycloakUserSSO, *keycloakUsers[0].ID); err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("user %s is member of all required groups", testUserName)
}

func pollGeneratedKeycloakUserCR(ctx context.Context, c client.Client) error {
	generatedKU := &keycloak.KeycloakUser{}
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second*10, time.Minute*3, true, func(ctx2 context.Context) (done bool, err error) {
		err = c.Get(
			ctx,
			types.NamespacedName{
				Namespace: RHSSOUserProductNamespace,
				Name:      fmt.Sprintf("generated-%s-%s", testUser.Name, testUser.UID),
			},
			generatedKU,
		)
		if err != nil {
			switch err := err.(type) {
			case *errors.StatusError:
				if err.ErrStatus.Code == http.StatusNotFound {
					return false, nil
				}
				return true, err
			default:
				return true, err
			}
		}
		userGroups := generatedKU.Spec.User.Groups
		expectedGroups := []string{
			groupNameDedicatedAdmins,
			fmt.Sprintf("%s/%s", groupNameDedicatedAdmins, groupNameDedicatedAdminsRealmManagers),
		}
		if !reflect.DeepEqual(expectedGroups, userGroups) {
			return true, fmt.Errorf("expected user with ID %s to be part of groups %v, got [%s]",
				testUser.UID,
				expectedGroups,
				userGroups,
			)
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("failed to retrieve generated KeycloakUser CR: %v", err)
	}
	return nil
}

func pollKeycloakUserGroups(httpClient *http.Client, host, userID string) error {
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*15, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
		if time.Now().After(tokenExpiryTime) {
			timeBeforeTokenReq := time.Now()
			tokens, err = refreshKeycloakToken(httpClient, host, tokens.RefreshToken)
			if err != nil {
				return true, err
			}
			tokenExpiryTime = timeBeforeTokenReq.Add(time.Duration(tokens.ExpiresIn) * time.Second)
		}
		groupOptions := keycloakUserGroupOptions{
			BriefRepresentation: pointer.Bool(true),
			RealmName:           realmNameMaster,
			UserID:              userID,
		}
		userGroups, err := getKeycloakUserGroups(httpClient, host, tokens.AccessToken, groupOptions)
		if err != nil {
			return true, err
		}
		var userGroupsSlice []string
		for _, group := range userGroups {
			userGroupsSlice = append(userGroupsSlice, *group.Name)
		}
		expectedGroups := []string{groupNameDedicatedAdmins, groupNameDedicatedAdminsRealmManagers, groupNameRHMIDevelopers}
		if !reflect.DeepEqual(expectedGroups, userGroupsSlice) {
			return false, nil
		}
		return true, nil
	})
	return err
}

func getHostKeycloak(ctx context.Context, c client.Client, namespace string) (string, error) {
	// Get Keycloak OpenShift route
	route := &routev1.Route{}
	if err := c.Get(ctx, client.ObjectKey{Name: "keycloak", Namespace: namespace}, route); err != nil {
		return "", fmt.Errorf("failed to get Keycloak route for namespace %s: %v", namespace, err)
	}

	// Evaluate the Keycloak route hostname
	hostName := route.Spec.Host
	if route.Spec.TLS != nil {
		hostName = fmt.Sprintf("https://%s", hostName)
	}
	return hostName, nil
}

func getKeycloakToken(httpClient *http.Client, host string, options keycloakTokenOptions) (*keycloakOpenIDTokenResponse, error) {
	params := url.Values{}
	if options.ClientID != nil {
		params.Set("client_id", *options.ClientID)
	}
	if options.GrantType != nil {
		params.Set("grant_type", *options.GrantType)
	}
	if options.RefreshToken != nil {
		params.Set("refresh_token", *options.RefreshToken)
	}
	if options.Username != nil {
		params.Set("username", *options.Username)
	}
	if options.Password != nil {
		params.Set("password", *options.Password)
	}
	urlEncodedParams := params.Encode()
	getKeycloakOpenIDTokenPath := fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", host, options.RealmName)
	getKeycloakOpenIDTokenReq, err := http.NewRequest("POST", getKeycloakOpenIDTokenPath, strings.NewReader(urlEncodedParams))
	if err != nil {
		return nil, fmt.Errorf("failed to init get token request for Keycloak: %v", err)
	}
	getKeycloakOpenIDTokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	getKeycloakOpenIDTokenReq.Header.Set("Content-Length", strconv.FormatInt(getKeycloakOpenIDTokenReq.ContentLength, 10))
	getKeycloakOpenIDTokenRes, err := httpClient.Do(getKeycloakOpenIDTokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Keycloak tokens: %v", err)
	}
	defer getKeycloakOpenIDTokenRes.Body.Close()
	if getKeycloakOpenIDTokenRes.StatusCode != http.StatusOK {
		dumpRes, err := httputil.DumpResponse(getKeycloakOpenIDTokenRes, true)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("dump response: %q", dumpRes)
	}
	getKeycloakAccessTokenResBody, err := ioutil.ReadAll(getKeycloakOpenIDTokenRes.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for request %s: %v", getKeycloakOpenIDTokenRes.Request.URL, err)
	}
	var keycloakOpenIDTokenRes *keycloakOpenIDTokenResponse
	if err := json.Unmarshal(getKeycloakAccessTokenResBody, &keycloakOpenIDTokenRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keycloak users: %v", err)
	}
	return keycloakOpenIDTokenRes, nil
}

/* #nosec G101 -- This is a false positive */
func getCredentialsRHSSOUser(ctx context.Context, c client.Client) ([]string, error) {
	secret := &corev1.Secret{}
	secretName := "credential-rhssouser"
	if err := c.Get(ctx, types.NamespacedName{Name: secretName, Namespace: RHSSOUserProductNamespace}, secret); err != nil {
		return nil, fmt.Errorf("error getting secret %s: %v", secretName, err)
	}
	userName := string(secret.Data["ADMIN_USERNAME"])
	password := string(secret.Data["ADMIN_PASSWORD"])
	return []string{userName, password}, nil
}

func getKeycloakUsers(httpClient *http.Client, host, token string, options keycloakUserOptions) ([]*keycloakUser, error) {
	getKeycloakUsersPath := fmt.Sprintf("%s/auth/admin/realms/%s/users", host, options.RealmName)
	getKeycloakUsersReq, err := http.NewRequest("GET", getKeycloakUsersPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init get users request for Keycloak: %v", err)
	}
	query := getKeycloakUsersReq.URL.Query()
	if options.BriefRepresentation != nil {
		query.Set("briefRepresentation", *options.IDPUserID)
	}
	if options.Email != nil {
		query.Set("email", *options.Email)
	}
	if options.EmailVerified != nil {
		query.Set("emailVerified", strconv.FormatBool(*options.EmailVerified))
	}
	if options.Enabled != nil {
		query.Set("enabled", strconv.FormatBool(*options.Enabled))
	}
	if options.Exact != nil {
		query.Set("exact", strconv.FormatBool(*options.Exact))
	}
	if options.First != nil {
		query.Set("first", strconv.FormatInt(int64(*options.First), 10))
	}
	if options.FirstName != nil {
		query.Set("firstName", *options.FirstName)
	}
	if options.IDPAlias != nil {
		query.Set("idpAlias", *options.IDPAlias)
	}
	if options.IDPUserID != nil {
		query.Set("idpUserId", *options.IDPUserID)
	}
	if options.LastName != nil {
		query.Set("lastName", *options.LastName)
	}
	if options.Max != nil {
		query.Set("max", strconv.FormatInt(int64(*options.Max), 10))
	}
	if options.Search != nil {
		query.Set("search", *options.Search)
	}
	if options.Username != nil {
		query.Set("username", *options.Username)
	}
	getKeycloakUsersReq.URL.RawQuery = query.Encode()
	getKeycloakUsersReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	getKeycloakUsersRes, err := httpClient.Do(getKeycloakUsersReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve users from Keycloak: %v", err)
	}
	defer getKeycloakUsersRes.Body.Close()
	if getKeycloakUsersRes.StatusCode != http.StatusOK {
		switch getKeycloakUsersRes.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized to retrieve Keycloak users")
		default:
			dumpRes, err := httputil.DumpResponse(getKeycloakUsersRes, true)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("dump response: %q", dumpRes)
		}
	}
	getKeycloakUsersResBody, err := ioutil.ReadAll(getKeycloakUsersRes.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for request %s: %v", getKeycloakUsersRes.Request.URL, err)
	}
	var keycloakUsers []*keycloakUser
	if err := json.Unmarshal(getKeycloakUsersResBody, &keycloakUsers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keycloak users: %v", err)
	}
	if len(keycloakUsers) == 0 {
		return nil, fmt.Errorf("no Keycloak users found for query %v: %v", options, err)
	}
	return keycloakUsers, nil
}

func createKeycloakUser(httpClient *http.Client, host, realmName, token string, user keycloakUser) error {
	createKeycloakUserPath := fmt.Sprintf("%s/auth/admin/realms/%s/users", host, realmName)
	createKeycloakUserBody, err := json.Marshal(user)
	if err != nil {
		return err
	}
	createKeycloakUserReq, err := http.NewRequest("POST", createKeycloakUserPath, strings.NewReader(string(createKeycloakUserBody)))
	if err != nil {
		return fmt.Errorf("failed to init create users request for Keycloak: %v", err)
	}
	createKeycloakUserReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	createKeycloakUserReq.Header.Set("Content-Type", "application/json")
	createKeycloakUserReq.Header.Set("Content-Length", strconv.FormatInt(createKeycloakUserReq.ContentLength, 10))
	createKeycloakUserRes, err := httpClient.Do(createKeycloakUserReq)
	if err != nil {
		return fmt.Errorf("failed to create Keycloak user: %v", err)
	}
	defer createKeycloakUserRes.Body.Close()
	if createKeycloakUserRes.StatusCode != http.StatusCreated {
		switch createKeycloakUserRes.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("unauthorized to create a Keycloak user")
		default:
			dumpRes, err := httputil.DumpResponse(createKeycloakUserRes, true)
			if err != nil {
				return err
			}
			return fmt.Errorf("dump response: %q", dumpRes)
		}
	}
	return nil
}

func getKeycloakUserGroups(httpClient *http.Client, host, token string, options keycloakUserGroupOptions) ([]*keycloakUserGroup, error) {
	getKeycloakUserGroupsPath := fmt.Sprintf("%s/auth/admin/realms/%s/users/%s/groups", host, options.RealmName, options.UserID)
	getKeycloakUserGroupsReq, err := http.NewRequest("GET", getKeycloakUserGroupsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init get user groups request for Keycloak: %v", err)
	}
	query := getKeycloakUserGroupsReq.URL.Query()
	if options.BriefRepresentation != nil {
		query.Set("briefRepresentation", strconv.FormatBool(*options.BriefRepresentation))
	}
	if options.First != nil {
		query.Set("first", strconv.FormatInt(int64(*options.First), 10))
	}
	if options.Max != nil {
		query.Set("max", strconv.FormatInt(int64(*options.Max), 10))
	}
	if options.Search != nil {
		query.Set("search", *options.Search)
	}
	getKeycloakUserGroupsReq.URL.RawQuery = query.Encode()
	getKeycloakUserGroupsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	getKeycloakUserGroupsRes, err := httpClient.Do(getKeycloakUserGroupsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve groups for user with ID %s from Keycloak: %v", options.UserID, err)
	}
	defer getKeycloakUserGroupsRes.Body.Close()
	if getKeycloakUserGroupsRes.StatusCode != http.StatusOK {
		switch getKeycloakUserGroupsRes.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized to retrieve Keycloak user groups")
		default:
			dumpRes, err := httputil.DumpResponse(getKeycloakUserGroupsRes, true)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("dump response: %q", dumpRes)
		}
	}
	getKeycloakUserGroupsResBody, err := ioutil.ReadAll(getKeycloakUserGroupsRes.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for request %s: %v", getKeycloakUserGroupsRes.Request.URL, err)
	}
	var keycloakUserGroups []*keycloakUserGroup
	if err := json.Unmarshal(getKeycloakUserGroupsResBody, &keycloakUserGroups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keycloak user groups: %v", err)
	}
	return keycloakUserGroups, nil
}

func cleanUpTestDedicatedAdminUsersSyncedSSO(ctx context.Context, t TestingTB, c client.Client) {
	// Ensure OpenShift user is deleted
	err := c.Delete(ctx, testUser)
	if err != nil {
		t.Fatalf("failed to delete OpenShift user %s, err: %v", testUser.Name, err)
	}

	// Ensure KeycloakUser CR has been deleted within 2 minutes
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*15, time.Minute*2, false, func(ctx2 context.Context) (done bool, err error) {
		err = c.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", TestingIDPRealm, testUserName),
			Namespace: RHSSOProductNamespace}, &keycloak.KeycloakUser{})
		if err != nil {
			if k8serr.IsNotFound(err) {
				return true, nil
			} else {
				return false, nil
			}
		} else {
			return false, nil
		}
	})
	if err != nil {
		t.Fatalf("keycloakUser CR is not deleted as expected %s, err: %v", fmt.Sprintf("%s-%s", TestingIDPRealm, testUserName), err)
	}

	// Ensure the test user is removed from dedicated-admins group
	dedicatedAdminGroup := &userv1.Group{}
	err = c.Get(ctx, types.NamespacedName{Name: groupNameDedicatedAdmins}, dedicatedAdminGroup)
	if err != nil {
		t.Fatalf("failed to get dedicated admins group %s, err: %v", groupNameDedicatedAdmins, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, c, dedicatedAdminGroup, func() error {
		for i, user := range dedicatedAdminGroup.Users {
			if user == testUserName {
				dedicatedAdminGroup.Users = removeIndex(dedicatedAdminGroup.Users, i)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to remove user %s from dedicated admin group, err: %v", testUserName, err)
	}
}

func removeIndex(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

func refreshKeycloakToken(client *http.Client, host, refreshToken string) (*keycloakOpenIDTokenResponse, error) {
	tokenOptions := keycloakTokenOptions{
		ClientID:     pointer.String(clientIDAdminCLI),
		GrantType:    pointer.String(grantTypeRefreshToken),
		RealmName:    realmNameMaster,
		RefreshToken: pointer.String(refreshToken),
	}
	return getKeycloakToken(client, host, tokenOptions)
}
