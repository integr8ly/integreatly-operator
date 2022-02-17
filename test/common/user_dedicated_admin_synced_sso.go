package common

import (
	"context"
	"encoding/json"
	"fmt"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	userv1 "github.com/openshift/api/user/v1"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"strings"
	"time"
)

const (
	testUserName                          = "test-user99"
	groupNameRHMIDevelopers               = "rhmi-developers"
	groupNameDedicatedAdmins              = "dedicated-admins"
	groupNameDedicatedAdminsRealmManagers = "realm-managers"
	clientIDAdminCLI                      = "admin-cli"
	grantTypePassword                     = "password"
)

var testUser = &userv1.User{
	ObjectMeta: metav1.ObjectMeta{
		Name: testUserName,
	},
}

func TestDedicatedAdminUsersSyncedSSO(t TestingTB, ctx *TestingContext) {
	goCtx := context.TODO()
	defer cleanUpTestDedicatedAdminUsersSyncedSSO(goCtx, ctx.Client)

	// Create Testing IDP
	if err := createTestingIDP(t, goCtx, ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error creating testing IDP: %v", err)
	}

	// Get RHMI CR details for test
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Create OpenShift user
	if err := ctx.Client.Create(goCtx, testUser); err != nil {
		t.Fatalf("error creating openshift user: %v", err)
	}
	t.Logf("openShift user %s created", testUser.Name)

	// List KeycloakUser CRs and validate there is no CR with such username
	validateUserNotListedInKeyCloakCR(t, ctx, goCtx, testUser.Name)

	// Create KeycloakUser CR in rhsso namespace
	var testUsersToCreate = []TestUser{
		{
			UserName:  testUserName,
			FirstName: "Test",
			LastName:  "User99",
		},
	}
	if err := createOrUpdateKeycloakUserCR(goCtx, ctx.Client, testUsersToCreate, rhmi.Name); err != nil {
		t.Fatalf("failed to create KeycloakUser CR in namespace %s: %v", RHSSOProductNamespace, err)
	}
	t.Logf("created KeycloakUser CR %s in namespace %s ", testUserName, RHSSOProductNamespace)

	// Add user to the dedicated-admins group
	if err := createOrUpdateDedicatedAdminGroupCR(goCtx, ctx.Client, []string{testUserName}); err != nil {
		t.Fatalf("failed to add user %s to group %s: %v", testUserName, groupNameDedicatedAdmins, err)
	}
	t.Logf("added user %s to group %s", testUserName, groupNameDedicatedAdmins)

	// Wait for RHOAM to reconcile the KeycloakUser CR in user-sso namespace and verify their group membership
	if err := pollGeneratedKeycloakUserCR(goCtx, ctx.Client); err != nil {
		t.Fatalf("%v", err)
	}

	// Get the matching test KeyCloak user and verify their group membership
	hostKeycloakUserSSO, err := getHostKeycloak(ctx.Client, RHSSOUserProductNamespace)
	tokens, err := getKeycloakTokens(ctx.Client, ctx.HttpClient, hostKeycloakUserSSO)
	if err != nil {
		t.Fatalf("%v", err)
	}
	keycloakUser, err := getKeycloakUserByIDPUserID(ctx.HttpClient, hostKeycloakUserSSO, tokens.AccessToken, testUser.UID)
	if err != nil {
		t.Fatalf("%v", err)
	}
	userGroups, err := getKeycloakUserGroups(ctx.HttpClient, hostKeycloakUserSSO, tokens.AccessToken, keycloakUser.ID)
	if err != nil {
		t.Fatalf("%v", err)
	}
	var userGroupsSlice []string
	for _, group := range userGroups {
		userGroupsSlice = append(userGroupsSlice, group.Name)
	}
	expectedGroups := []string{groupNameDedicatedAdmins, groupNameDedicatedAdminsRealmManagers, groupNameRHMIDevelopers}
	if !reflect.DeepEqual(expectedGroups, userGroupsSlice) {
		t.Fatalf("Expected user groups %s, got %s", expectedGroups, userGroupsSlice)
	}

	t.Log("all checks for test B09 completed successfully")
}

func pollGeneratedKeycloakUserCR(ctx context.Context, client client.Client) error {
	generatedKU := &keycloak.KeycloakUser{}
	if err := wait.Poll(time.Second*10, time.Minute*3, func() (done bool, err error) {
		err = client.Get(
			ctx,
			types.NamespacedName{
				Namespace: RHSSOUserProductNamespace,
				Name:      fmt.Sprintf("generated-%s-%s", testUser.Name, testUser.UID),
			},
			generatedKU,
		)
		if err != nil {
			switch err.(type) {
			case *errors.StatusError:
				statusErr := err.(*errors.StatusError)
				if statusErr.ErrStatus.Code == http.StatusNotFound {
					return false, nil
				}
				return true, statusErr
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

func getHostKeycloak(c client.Client, namespace string) (string, error) {
	Context := context.TODO()
	// Get Keycloak OpenShift route
	route := &routev1.Route{}
	if err := c.Get(Context, client.ObjectKey{Name: "keycloak", Namespace: namespace}, route); err != nil {
		return "", fmt.Errorf("failed to get Keycloak route for namespace %s: %v", namespace, err)
	}

	// Evaluate the Keycloak route hostname
	hostName := route.Spec.Host
	if route.Spec.TLS != nil {
		hostName = fmt.Sprintf("https://%s", hostName)
	}
	return hostName, nil
}

func getKeycloakTokens(client client.Client, httpClient *http.Client, host string) (*keycloakOpenIDTokenResponse, error) {
	secret, err := getCredentialsRHSSOUser(client)
	if err != nil {
		return nil, err
	}
	formParams := url.Values{}
	formParams.Set("client_id", clientIDAdminCLI)
	formParams.Set("username", secret[0])
	formParams.Set("password", secret[1])
	formParams.Set("grant_type", grantTypePassword)
	urlEncodedParams := formParams.Encode()
	getKeycloakOpenIDTokenPath := fmt.Sprintf("%s/auth/realms/master/protocol/openid-connect/token", host)
	getKeycloakOpenIDTokenReq, err := http.NewRequest("POST", getKeycloakOpenIDTokenPath, strings.NewReader(urlEncodedParams))
	if err != nil {
		return nil, fmt.Errorf("failed to create get token request for Keycloak user-sso: %v", err)
	}
	getKeycloakOpenIDTokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	getKeycloakOpenIDTokenReq.Header.Add("Content-Length", strconv.Itoa(len(urlEncodedParams)))
	getKeycloakOpenIDTokenRes, err := httpClient.Do(getKeycloakOpenIDTokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Keycloak tokens: %v", err)
	}
	defer getKeycloakOpenIDTokenRes.Body.Close()
	if getKeycloakOpenIDTokenRes.StatusCode != http.StatusOK {
		dumpRes, _ := httputil.DumpResponse(getKeycloakOpenIDTokenRes, true)
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

func getCredentialsRHSSOUser(client client.Client) ([]string, error) {
	secret := &corev1.Secret{}
	secretName := "credential-rhssouser"
	if err := client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: RHSSOUserProductNamespace}, secret); err != nil {
		return nil, fmt.Errorf("error getting secret %s: %v", secretName, err)
	}
	userName := string(secret.Data["ADMIN_USERNAME"])
	password := string(secret.Data["ADMIN_PASSWORD"])
	return []string{userName, password}, nil
}

func getKeycloakUserByIDPUserID(httpClient *http.Client, host, token string, idpUserID types.UID) (*keycloakUser, error) {
	getKeycloakUsersPath := fmt.Sprintf("%s/auth/admin/realms/master/users/?idpUserId=%s", host, idpUserID)
	getKeycloakUsersReq, err := http.NewRequest("GET", getKeycloakUsersPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get users request for Keycloak user-sso: %v", err)
	}
	getKeycloakUsersReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	getKeycloakUsersRes, err := httpClient.Do(getKeycloakUsersReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve users from Keycloak user-sso: %v", err)
	}
	defer getKeycloakUsersRes.Body.Close()
	if getKeycloakUsersRes.StatusCode != http.StatusOK {
		dumpRes, _ := httputil.DumpResponse(getKeycloakUsersRes, true)
		return nil, fmt.Errorf("dump response: %q", dumpRes)
	}
	getKeycloakUsersResBody, err := ioutil.ReadAll(getKeycloakUsersRes.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for request %s: %v", getKeycloakUsersRes.Request.URL, err)
	}
	var keycloakUsers []*keycloakUser
	if err := json.Unmarshal(getKeycloakUsersResBody, &keycloakUsers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keycloak users: %v", err)
	}
	return keycloakUsers[0], nil
}

func getKeycloakUserGroups(httpClient *http.Client, host, token, keycloakUserID string) ([]*keycloakUserGroup, error) {
	getKeycloakUserGroupsPath := fmt.Sprintf("%s/auth/admin/realms/master/users/%s/groups", host, keycloakUserID)
	getKeycloakUserGroupsReq, err := http.NewRequest("GET", getKeycloakUserGroupsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get user groups request for Keycloak user-sso: %v", err)
	}
	getKeycloakUserGroupsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	getKeycloakUserGroupsRes, err := httpClient.Do(getKeycloakUserGroupsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve groups for user with ID %s from Keycloak user-sso: %v", keycloakUserID, err)
	}
	defer getKeycloakUserGroupsRes.Body.Close()
	if getKeycloakUserGroupsRes.StatusCode != http.StatusOK {
		dumpRes, _ := httputil.DumpResponse(getKeycloakUserGroupsRes, true)
		return nil, fmt.Errorf("dump response: %q", dumpRes)
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

func cleanUpTestDedicatedAdminUsersSyncedSSO(ctx context.Context, client client.Client) {
	// Ensure OpenShift user is deleted
	client.Delete(ctx, testUser)

	// Ensure KeycloakUser CR is deleted
	client.Delete(ctx, &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", TestingIDPRealm, testUserName),
			Namespace: RHSSOProductNamespace,
		},
	})

	// Ensure the test user is removed from dedicated-admins group
	dedicatedAdminGroup := &userv1.Group{}
	client.Get(ctx, types.NamespacedName{Name: groupNameDedicatedAdmins}, dedicatedAdminGroup)
	controllerutil.CreateOrUpdate(ctx, client, dedicatedAdminGroup, func() error {
		for i, user := range dedicatedAdminGroup.Users {
			if user == testUserName {
				dedicatedAdminGroup.Users = removeIndex(dedicatedAdminGroup.Users, i)
			}
		}
		return nil
	})
}

func removeIndex(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
