package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	types "github.com/integr8ly/keycloak-client/pkg/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	config2 "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	authURL = "auth/realms/master/protocol/openid-connect/token"
)

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	requester Requester
	URL       string
	token     string
}

// T is a generic type for keycloak spec resources
type T interface{}

// Generic create function for creating new Keycloak resources
func (c *Client) create(obj T, resourcePath, resourceName string) (string, error) {
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		logrus.Errorf("error %+v marshalling object", err)
		return "", nil
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		logrus.Errorf("error creating POST %s request %+v", resourceName, err)
		return "", errors.Wrapf(err, "error creating POST %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)

	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return "", errors.Wrapf(err, "error performing POST %s request", resourceName)
	}
	defer res.Body.Close()

	if res.StatusCode != 201 && res.StatusCode != 204 {
		return "", fmt.Errorf("failed to create %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	if resourceName == "client" {
		d, _ := ioutil.ReadAll(res.Body)
		fmt.Println("user response ", string(d))
	}

	location := strings.Split(res.Header.Get("Location"), "/")
	uid := location[len(location)-1]
	return uid, nil
}

func (c *Client) CreateRealm(realm *types.KeycloakRealm) (string, error) {
	return c.create(realm.Spec.Realm, "realms", "realm")
}

func (c *Client) CreateClient(client *types.KeycloakAPIClient, realmName string) (string, error) {
	return c.create(client, fmt.Sprintf("realms/%s/clients", realmName), "client")
}

func (c *Client) CreateUser(user *types.KeycloakAPIUser, realmName string) (string, error) {
	return c.create(user, fmt.Sprintf("realms/%s/users", realmName), "user")
}

func (c *Client) CreateFederatedIdentity(fid types.FederatedIdentity, userID string, realmName string) (string, error) {
	return c.create(fid, fmt.Sprintf("realms/%s/users/%s/federated-identity/%s", realmName, userID, fid.IdentityProvider), "federated-identity")
}

func (c *Client) RemoveFederatedIdentity(fid types.FederatedIdentity, userID string, realmName string) error {
	return c.delete(fmt.Sprintf("realms/%s/users/%s/federated-identity/%s", realmName, userID, fid.IdentityProvider), "federated-identity", fid)
}

func (c *Client) GetUserFederatedIdentities(userID string, realmName string) ([]types.FederatedIdentity, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users/%s/federated-identity", realmName, userID), "federated-identity", func(body []byte) (T, error) {
		var fids []types.FederatedIdentity
		err := json.Unmarshal(body, &fids)
		return fids, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]types.FederatedIdentity), err
}

func (c *Client) CreateUserClientRole(role *types.KeycloakUserRole, realmName, clientID, userID string) (string, error) {
	return c.create(
		[]*types.KeycloakUserRole{role},
		fmt.Sprintf("realms/%s/users/%s/role-mappings/clients/%s", realmName, userID, clientID),
		"user-client-role",
	)
}
func (c *Client) CreateUserRealmRole(role *types.KeycloakUserRole, realmName, userID string) (string, error) {
	return c.create(
		[]*types.KeycloakUserRole{role},
		fmt.Sprintf("realms/%s/users/%s/role-mappings/realm", realmName, userID),
		"user-realm-role",
	)
}

func (c *Client) CreateAuthenticatorConfig(authenticatorConfig *types.AuthenticatorConfig, realmName, executionID string) (string, error) {
	return c.create(authenticatorConfig, fmt.Sprintf("realms/%s/authentication/executions/%s/config", realmName, executionID), "AuthenticatorConfig")
}

func (c *Client) DeleteUserClientRole(role *types.KeycloakUserRole, realmName, clientID, userID string) error {
	err := c.delete(
		fmt.Sprintf("realms/%s/users/%s/role-mappings/clients/%s", realmName, userID, clientID),
		"user-client-role",
		[]*types.KeycloakUserRole{role},
	)
	return err
}

func (c *Client) DeleteUserRealmRole(role *types.KeycloakUserRole, realmName, userID string) error {
	err := c.delete(
		fmt.Sprintf("realms/%s/users/%s/role-mappings/realm", realmName, userID),
		"user-realm-role",
		[]*types.KeycloakUserRole{role},
	)
	return err
}

func (c *Client) UpdatePassword(user *types.KeycloakAPIUser, realmName, newPass string) error {
	passReset := &types.KeycloakAPIPasswordReset{}
	passReset.Type = "password"
	passReset.Temporary = false
	passReset.Value = newPass
	u := fmt.Sprintf("realms/%s/users/%s/reset-password", realmName, user.ID)
	if err := c.update(passReset, u, "paswordreset"); err != nil {
		return errors.Wrap(err, "error calling keycloak api ")
	}
	return nil
}

func (c *Client) FindUserByEmail(email, realm string) (*types.KeycloakAPIUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users?first=0&max=1&search=%s", realm, email), "user", func(body []byte) (T, error) {
		var users []*types.KeycloakAPIUser
		if err := json.Unmarshal(body, &users); err != nil {
			return nil, err
		}
		if len(users) == 0 {
			return nil, nil
		}
		return users[0], nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, err
	}
	return result.(*types.KeycloakAPIUser), nil
}

func (c *Client) FindUserByUsername(name, realm string) (*types.KeycloakAPIUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users?username=%s&max=-1", realm, name), "user", func(body []byte) (T, error) {
		var users []*types.KeycloakAPIUser
		if err := json.Unmarshal(body, &users); err != nil {
			return nil, err
		}

		for _, user := range users {
			if user.UserName == name {
				return user, nil
			}
		}
		return nil, errors.New("not found")
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*types.KeycloakAPIUser), nil
}

func (c *Client) CreateIdentityProvider(identityProvider *types.KeycloakIdentityProvider, realmName string) (string, error) {
	return c.create(identityProvider, fmt.Sprintf("realms/%s/identity-provider/instances", realmName), "identity provider")
}

// Generic get function for returning a Keycloak resource
func (c *Client) get(resourcePath, resourceName string, unMarshalFunc func(body []byte) (T, error)) (T, error) {
	u := fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath)
	req, err := http.NewRequest(
		"GET",
		u,
		nil,
	)
	if err != nil {
		logrus.Errorf("error creating GET %s request %+v", resourceName, err)
		return nil, errors.Wrapf(err, "error creating GET %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return nil, errors.Wrapf(err, "error performing GET %s request", resourceName)
	}

	defer res.Body.Close()
	if res.StatusCode == 404 {
		logrus.Errorf("Resource %v/%v doesn't exist", resourcePath, resourceName)
		return nil, nil
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to GET %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return nil, errors.Wrapf(err, "error reading %s GET response", resourceName)
	}

	obj, err := unMarshalFunc(body)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return obj, nil
}

func (c *Client) GetRealm(realmName string) (*types.KeycloakRealm, error) {
	result, err := c.get(fmt.Sprintf("realms/%s", realmName), "realm", func(body []byte) (T, error) {
		realm := &types.KeycloakAPIRealm{}
		err := json.Unmarshal(body, realm)
		return realm, err
	})
	if result == nil {
		return nil, nil
	}
	ret := &types.KeycloakRealm{
		Spec: types.KeycloakRealmSpec{
			Realm: result.(*types.KeycloakAPIRealm),
		},
	}
	return ret, err
}

func (c *Client) GetClient(clientID, realmName string) (*types.KeycloakAPIClient, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/clients/%s", realmName, clientID), "client", func(body []byte) (T, error) {
		client := &types.KeycloakAPIClient{}
		err := json.Unmarshal(body, client)
		return client, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	ret := result.(*types.KeycloakAPIClient)
	return ret, err
}

func (c *Client) GetClientSecret(clientID, realmName string) (string, error) {
	//"https://{{ rhsso_route }}/auth/admin/realms/{{ rhsso_realm }}/clients/{{ rhsso_client_id }}/client-secret"
	result, err := c.get(fmt.Sprintf("realms/%s/clients/%s/client-secret", realmName, clientID), "client-secret", func(body []byte) (T, error) {
		res := map[string]string{}
		if err := json.Unmarshal(body, &res); err != nil {
			return nil, err
		}
		return res["value"], nil
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get: "+fmt.Sprintf("realms/%s/clients/%s/client-secret", realmName, clientID))
	}
	if result == nil {
		return "", nil
	}
	return result.(string), nil
}

func (c *Client) GetClientInstall(clientID, realmName string) ([]byte, error) {
	var response []byte
	if _, err := c.get(fmt.Sprintf("realms/%s/clients/%s/installation/providers/keycloak-oidc-keycloak-json", realmName, clientID), "client-installation", func(body []byte) (T, error) {
		response = body
		return body, nil
	}); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) GetUser(userID, realmName string) (*types.KeycloakAPIUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users/%s", realmName, userID), "user", func(body []byte) (T, error) {
		user := &types.KeycloakAPIUser{}
		err := json.Unmarshal(body, user)
		return user, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	ret := result.(*types.KeycloakAPIUser)
	return ret, err
}

func (c *Client) GetIdentityProvider(alias string, realmName string) (*types.KeycloakIdentityProvider, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/identity-provider/instances/%s", realmName, alias), "identity provider", func(body []byte) (T, error) {
		provider := &types.KeycloakIdentityProvider{}
		err := json.Unmarshal(body, provider)
		return provider, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*types.KeycloakIdentityProvider), err
}

func (c *Client) GetAuthenticatorConfig(configID, realmName string) (*types.AuthenticatorConfig, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/authentication/config/%s", realmName, configID), "AuthenticatorConfig", func(body []byte) (T, error) {
		authenticatorConfig := &types.AuthenticatorConfig{}
		err := json.Unmarshal(body, authenticatorConfig)
		return authenticatorConfig, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*types.AuthenticatorConfig), err
}

// Generic put function for updating Keycloak resources
func (c *Client) update(obj T, resourcePath, resourceName string) error {
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		return nil
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		logrus.Errorf("error creating UPDATE %s request %+v", resourceName, err)
		return errors.Wrapf(err, "error creating UPDATE %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.token)
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrapf(err, "error performing UPDATE %s request", resourceName)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		logrus.Errorf("failed to UPDATE %s %v", resourceName, res.Status)
		return fmt.Errorf("failed to UPDATE %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	return nil
}

func (c *Client) UpdateRealm(realm *types.KeycloakRealm) error {
	return c.update(realm, fmt.Sprintf("realms/%s", realm.Spec.Realm.ID), "realm")
}

func (c *Client) UpdateClient(specClient *types.KeycloakAPIClient, realmName string) error {
	return c.update(specClient, fmt.Sprintf("realms/%s/clients/%s", realmName, specClient.ID), "client")
}

func (c *Client) UpdateUser(specUser *types.KeycloakAPIUser, realmName string) error {
	return c.update(specUser, fmt.Sprintf("realms/%s/users/%s", realmName, specUser.ID), "user")
}

func (c *Client) UpdateIdentityProvider(specIdentityProvider *types.KeycloakIdentityProvider, realmName string) error {
	return c.update(specIdentityProvider, fmt.Sprintf("realms/%s/identity-provider/instances/%s", realmName, specIdentityProvider.Alias), "identity provider")
}

func (c *Client) UpdateAuthenticatorConfig(authenticatorConfig *types.AuthenticatorConfig, realmName string) error {
	return c.update(authenticatorConfig, fmt.Sprintf("realms/%s/authentication/config/%s", realmName, authenticatorConfig.ID), "AuthenticatorConfig")
}

// Generic delete function for deleting Keycloak resources
func (c *Client) delete(resourcePath, resourceName string, obj T) error {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		nil,
	)

	if obj != nil {
		jsonValue, err := json.Marshal(obj)
		if err != nil {
			return nil
		}
		req, err = http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
			bytes.NewBuffer(jsonValue),
		)
		if err != nil {
			return nil
		}
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		logrus.Errorf("error creating DELETE %s request %+v", resourceName, err)
		return errors.Wrapf(err, "error creating DELETE %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrapf(err, "error performing DELETE %s request", resourceName)
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		logrus.Errorf("Resource %v/%v already deleted", resourcePath, resourceName)
	}
	if res.StatusCode != 204 && res.StatusCode != 404 {
		return fmt.Errorf("failed to DELETE %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	return nil
}

func (c *Client) DeleteRealm(realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s", realmName), "realm", nil)
	return err
}

func (c *Client) DeleteClient(clientID, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/clients/%s", realmName, clientID), "client", nil)
	return err
}

func (c *Client) DeleteUser(userID, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/users/%s", realmName, userID), "user", nil)
	return err
}

func (c *Client) DeleteIdentityProvider(alias string, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/identity-provider/instances/%s", realmName, alias), "identity provider", nil)
	return err
}

func (c *Client) DeleteAuthenticatorConfig(configID, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/authentication/config/%s", realmName, configID), "AuthenticatorConfig", nil)
	return err
}

// Generic list function for listing Keycloak resources
func (c *Client) list(resourcePath, resourceName string, unMarshalListFunc func(body []byte) (T, error)) (T, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		nil,
	)
	if err != nil {
		logrus.Errorf("error creating LIST %s request %+v", resourceName, err)
		return nil, errors.Wrapf(err, "error creating LIST %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return nil, errors.Wrapf(err, "error performing LIST %s request", resourceName)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("failed to LIST %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return nil, errors.Wrapf(err, "error reading %s LIST response", resourceName)
	}

	objs, err := unMarshalListFunc(body)
	if err != nil {
		logrus.Error(err)
	}

	return objs, nil
}

func (c *Client) ListRealms() ([]*types.KeycloakAPIRealm, error) {
	result, err := c.list("realms", "realm", func(body []byte) (T, error) {
		var realms []*types.KeycloakAPIRealm
		err := json.Unmarshal(body, &realms)
		return realms, err
	})
	resultAsRealm, ok := result.([]*types.KeycloakAPIRealm)
	if !ok {
		return nil, err
	}
	return resultAsRealm, err
}

func (c *Client) ListClients(realmName string) ([]*types.KeycloakAPIClient, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/clients", realmName), "clients", func(body []byte) (T, error) {
		var clients []*types.KeycloakAPIClient
		err := json.Unmarshal(body, &clients)
		return clients, err
	})

	if err != nil {
		return nil, err
	}

	res, ok := result.([]*types.KeycloakAPIClient)

	if !ok {
		return nil, errors.New("error decoding list clients response")
	}

	return res, nil
}

func (c *Client) ListUsers(realmName string) ([]*types.KeycloakAPIUser, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/users", realmName), "users", func(body []byte) (T, error) {
		var users []*types.KeycloakAPIUser
		err := json.Unmarshal(body, &users)
		return users, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*types.KeycloakAPIUser), err
}

func (c *Client) ListUsersInGroup(realmName, groupID string) ([]*types.KeycloakAPIUser, error) {
	path := fmt.Sprintf("realms/%s/groups/%s/members", realmName, groupID)
	result, err := c.list(path, "users", func(body []byte) (T, error) {
		var users []*types.KeycloakAPIUser
		err := json.Unmarshal(body, &users)
		return users, err
	})
	if err != nil {
		return nil, err
	}

	return result.([]*types.KeycloakAPIUser), nil
}

func (c *Client) AddUserToGroup(realmName, userID, groupID string) error {
	add := map[string]string{
		"userId":  userID,
		"groupId": groupID,
		"realm":   realmName,
	}
	path := fmt.Sprintf("realms/%s/users/%s/groups/%s", realmName, userID, groupID)

	return c.update(add, path, "user-group")
}

func (c *Client) DeleteUserFromGroup(realmName, userID, groupID string) error {
	path := fmt.Sprintf("realms/%s/users/%s/groups/%s", realmName, userID, groupID)

	return c.delete(path, "user-group", nil)
}

func (c *Client) ListIdentityProviders(realmName string) ([]*types.KeycloakIdentityProvider, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/identity-provider/instances", realmName), "identity providers", func(body []byte) (T, error) {
		var providers []*types.KeycloakIdentityProvider
		err := json.Unmarshal(body, &providers)
		return providers, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*types.KeycloakIdentityProvider), err
}

func (c *Client) ListUserClientRoles(realmName, clientID, userID string) ([]*types.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/clients/"+clientID, "userClientRoles", func(body []byte) (t T, e error) {
		var userClientRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &userClientRoles)
		return userClientRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) ListAvailableUserClientRoles(realmName, clientID, userID string) ([]*types.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/clients/"+clientID+"/available", "userClientRoles", func(body []byte) (t T, e error) {
		var userClientRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &userClientRoles)
		return userClientRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) ListUserRealmRoles(realmName, userID string) ([]*types.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/realm", "userRealmRoles", func(body []byte) (t T, e error) {
		var userRealmRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &userRealmRoles)
		return userRealmRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) ListAvailableUserRealmRoles(realmName, userID string) ([]*types.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/realm/available", "userClientRoles", func(body []byte) (t T, e error) {
		var userRealmRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &userRealmRoles)
		return userRealmRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) CreateAuthenticationFlow(authFlow AuthenticationFlow, realmName string) (string, error) {
	path := fmt.Sprintf("realms/%s/authentication/flows", realmName)
	return c.create(authFlow, path, "AuthenticationFlow")
}

func (c *Client) ListAuthenticationFlows(realmName string) ([]*AuthenticationFlow, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/authentication/flows", realmName), "AuthenticationFlow", func(body []byte) (T, error) {
		var authenticationFlows []*AuthenticationFlow
		err := json.Unmarshal(body, &authenticationFlows)
		return authenticationFlows, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*AuthenticationFlow), err
}

func (c *Client) FindAuthenticationFlowByAlias(flowAlias string, realmName string) (*AuthenticationFlow, error) {
	authFlows, err := c.ListAuthenticationFlows(realmName)
	if err != nil {
		return nil, err
	}
	for _, authFlow := range authFlows {
		if authFlow.Alias == flowAlias {
			return authFlow, nil
		}
	}

	return nil, nil
}

func (c *Client) AddExecutionToAuthenticatonFlow(flowAlias string, realmName string, providerID string, requirement Requirement) error {
	path := fmt.Sprintf("realms/%s/authentication/flows/%s/executions/execution", realmName, flowAlias)
	provider := map[string]string{
		"provider": providerID,
	}
	_, err := c.create(provider, path, "AuthenticationExecution")
	if err != nil {
		return errors.Wrapf(err, "error creating Authentication Execution %s", providerID)
	}

	// updates the execution requirement after its creation
	if requirement != "" {
		execution, err := c.FindAuthenticationExecutionForFlow(flowAlias, realmName, func(execution *types.AuthenticationExecutionInfo) bool {
			return execution.ProviderID == providerID
		})
		if err != nil {
			return errors.Wrapf(err, "error finding Authentication Execution %s", providerID)
		}
		execution.Requirement = string(requirement)
		err = c.UpdateAuthenticationExecutionForFlow(flowAlias, realmName, execution)
		if err != nil {
			return errors.Wrapf(err, "error updating Authentication Execution %s", providerID)
		}
	}

	return nil
}

func (c *Client) ListAuthenticationExecutionsForFlow(flowAlias, realmName string) ([]*types.AuthenticationExecutionInfo, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/authentication/flows/%s/executions", realmName, flowAlias), "AuthenticationExecution", func(body []byte) (T, error) {
		var authenticationExecutions []*types.AuthenticationExecutionInfo
		err := json.Unmarshal(body, &authenticationExecutions)
		return authenticationExecutions, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*types.AuthenticationExecutionInfo), err
}

func (c *Client) FindAuthenticationExecutionForFlow(flowAlias, realmName string, predicate func(*types.AuthenticationExecutionInfo) bool) (*types.AuthenticationExecutionInfo, error) {
	executions, err := c.ListAuthenticationExecutionsForFlow(flowAlias, realmName)

	if err != nil {
		return nil, err
	}

	for _, execution := range executions {
		if predicate(execution) {
			return execution, nil
		}
	}

	return nil, nil
}

func (c *Client) UpdateAuthenticationExecutionForFlow(flowAlias, realmName string, execution *types.AuthenticationExecutionInfo) error {
	path := fmt.Sprintf("realms/%s/authentication/flows/%s/executions", realmName, flowAlias)
	return c.update(execution, path, "AuthenticationExecution")
}

func (c *Client) FindGroupByName(groupName string, realmName string) (*Group, error) {
	// Get a list of the groups in the realm
	groups, err := c.list(fmt.Sprintf("realms/%s/groups", realmName), "Group", func(body []byte) (T, error) {
		var groups []*Group
		err := json.Unmarshal(body, &groups)
		return groups, err
	})

	if err != nil {
		return nil, err
	}

	// Function that recursively looks for the group in the hierarchy
	var findInList func([]*Group) *Group
	findInList = func(groupList []*Group) *Group {
		for _, group := range groupList {
			if group.Name == groupName {
				return group
			}

			childGroup := findInList(group.SubGroups)
			if childGroup != nil {
				return childGroup
			}
		}

		return nil
	}

	// If the loop finishes without finding the group,
	// return nil
	return findInList(groups.([]*Group)), nil
}

func (c *Client) FindGroupByPath(groupPath, realmName string) (*Group, error) {
	// Given a path "root/sub1/sub2", convert into ["root", "sub1", "sub2"]
	paths := strings.Split(groupPath, "/")

	// Find the "root" group
	rootGroup, err := c.FindGroupByName(paths[0], realmName)
	if err != nil {
		return nil, err
	}

	// If the path is "root", there's no need to keep looking, return the
	// group
	if len(paths) == 1 {
		return rootGroup, nil
	}

	// Iterate through the subpaths ["sub1", "sub2"] and look for the next
	// path in the current group level
	currentGroup := rootGroup
	for level := 1; level < len(paths); level++ {
		currentPath := paths[level]
		foundInLevel := false
		for _, subGroup := range currentGroup.SubGroups {
			if subGroup.Name != currentPath {
				continue
			}

			currentGroup = subGroup
			foundInLevel = true
		}

		// We iterated through all the subgroups and didn't find
		// the subpath, return nil as the group is not in the expected
		// hierarchy
		if !foundInLevel {
			return nil, nil
		}
	}

	return currentGroup, nil
}

func (c *Client) CreateGroup(groupName string, realmName string) (string, error) {
	group := Group{
		Name: groupName,
	}

	// Create the new group
	return c.create(group, fmt.Sprintf("realms/%s/groups", realmName), "group")
}

func (c *Client) MakeGroupDefault(groupID string, realmName string) error {
	// Get the existing default groups to check if the group is already
	// default
	defaultGroups, err := c.ListDefaultGroups(realmName)

	if err != nil {
		return err
	}

	// If the group is in the list, return
	for _, defaultGroup := range defaultGroups {
		if defaultGroup.ID == groupID {
			return nil
		}
	}

	// If not, perform the update
	return c.update(nil, fmt.Sprintf("realms/%s/default-groups/%s", realmName, groupID), "Realms")
}

func (c *Client) ListDefaultGroups(realmName string) ([]*Group, error) {
	groups, err := c.list(fmt.Sprintf("realms/%s/default-groups", realmName), "Default group", func(body []byte) (T, error) {
		var groups []*Group
		err := json.Unmarshal(body, &groups)
		return groups, err
	})

	if err != nil {
		return nil, err
	}

	return groups.([]*Group), nil
}

func (c *Client) SetGroupChild(groupID, realmName string, childGroup *Group) error {
	// Get the parent group
	parentGroup, err := c.get(
		fmt.Sprintf("realms/%s/groups/%s", realmName, groupID),
		"group",
		func(body []byte) (T, error) {
			group := &Group{}
			err := json.Unmarshal(body, group)
			return group, err
		},
	)

	if err != nil {
		return err
	}

	// If the child group to add is already in the list of children groups,
	// finish
	for _, existingChildren := range parentGroup.(*Group).SubGroups {
		if existingChildren.ID == childGroup.ID {
			return nil
		}
	}

	// Otherwise, set the child group
	_, err = c.create(
		childGroup,
		fmt.Sprintf("realms/%s/groups/%s/children", realmName, groupID),
		"group-child",
	)
	return err
}

func (c *Client) CreateGroupClientRole(role *types.KeycloakUserRole, realmName, clientID, groupID string) (string, error) {
	return c.create(
		[]*types.KeycloakUserRole{role},
		fmt.Sprintf("realms/%s/groups/%s/role-mappings/clients/%s", realmName, groupID, clientID),
		"group-client-role",
	)
}

func (c *Client) ListAvailableGroupClientRoles(realmName, clientID, groupID string) ([]*types.KeycloakUserRole, error) {
	path := fmt.Sprintf("realms/%s/groups/%s/role-mappings/clients/%s/available", realmName, groupID, clientID)
	objects, err := c.list(path, "groupRealmRoles", func(body []byte) (t T, e error) {
		var groupClientRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &groupClientRoles)
		return groupClientRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) FindAvailableGroupClientRole(realmName, clientID, groupID string, predicate func(*types.KeycloakUserRole) bool) (*types.KeycloakUserRole, error) {
	availableRoles, err := c.ListAvailableGroupClientRoles(realmName, clientID, groupID)

	if err != nil {
		return nil, err
	}

	for _, role := range availableRoles {
		if predicate(role) {
			return role, nil
		}
	}

	return nil, nil
}

func (c *Client) ListGroupClientRoles(realmName, clientID, groupID string) ([]*types.KeycloakUserRole, error) {
	path := fmt.Sprintf("realms/%s/groups/%s/role-mappings/clients/%s", realmName, groupID, clientID)
	objects, err := c.list(path, "groupClientRoles", func(body []byte) (t T, e error) {
		var groupClientRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &groupClientRoles)
		return groupClientRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) FindGroupClientRole(realmName, clientID, groupID string, predicate func(*types.KeycloakUserRole) bool) (*types.KeycloakUserRole, error) {
	groupRoles, err := c.ListGroupClientRoles(realmName, clientID, groupID)

	if err != nil {
		return nil, err
	}

	for _, role := range groupRoles {
		if predicate(role) {
			return role, nil
		}
	}

	return nil, nil
}

func (c *Client) CreateGroupRealmRole(role *types.KeycloakUserRole, realmName, groupID string) (string, error) {
	return c.create(
		[]*types.KeycloakUserRole{role},
		fmt.Sprintf("realms/%s/groups/%s/role-mappings/realm", realmName, groupID),
		"group-realm-role",
	)
}

func (c *Client) UpdateEventsConfig(realmName string, enabledEventTypes, eventsListeners []string) error {
	add := map[string]interface{}{
		"eventsEnabled":     "true",
		"eventsListeners":   eventsListeners,
		"enabledEventTypes": enabledEventTypes,
	}
	path := fmt.Sprintf("realms/%s/events/config", realmName)

	return c.update(add, path, "events-config")
}

func (c *Client) ListOfActivesUsersPerRealm(realmName, dateFrom string, max int) ([]Users, error) {
	filter := "type=LOGIN"
	if max != 0 {
		filter = fmt.Sprintf("%s&max=%d", filter, max)
	}
	if dateFrom != "" {
		filter = fmt.Sprintf("%s&dateFrom=%s", filter, dateFrom)
	}

	path := fmt.Sprintf("realms/%s/events?%s", realmName, filter)
	objects, err := c.list(path, "listActiveUsers", func(body []byte) (t T, e error) {
		var users []Users
		err := json.Unmarshal(body, &users)
		return users, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]Users), err
}

func (c *Client) ListGroupRealmRoles(realmName, groupID string) ([]*types.KeycloakUserRole, error) {
	path := fmt.Sprintf("realms/%s/groups/%s/role-mappings/realm", realmName, groupID)
	objects, err := c.list(path, "groupRealmRoles", func(body []byte) (t T, e error) {
		var groupRealmRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &groupRealmRoles)
		return groupRealmRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) ListAvailableGroupRealmRoles(realmName, groupID string) ([]*types.KeycloakUserRole, error) {
	path := fmt.Sprintf("realms/%s/groups/%s/role-mappings/realm/available", realmName, groupID)
	objects, err := c.list(path, "groupClientRoles", func(body []byte) (t T, e error) {
		var groupRealmRoles []*types.KeycloakUserRole
		err := json.Unmarshal(body, &groupRealmRoles)
		return groupRealmRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*types.KeycloakUserRole), err
}

func (c *Client) Ping() error {
	u := c.URL + "/auth/"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		logrus.Errorf("error creating ping request %+v", err)
		return errors.Wrap(err, "error creating ping request")
	}

	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrapf(err, "error performing ping request")
	}

	logrus.Debugf("response status: %v, %v", res.StatusCode, res.Status)
	if res.StatusCode != 200 {
		return fmt.Errorf("failed to ping, response status code: %v", res.StatusCode)
	}
	defer res.Body.Close()

	return nil
}

// login requests a new auth token from Keycloak
func (c *Client) login(user, pass string) error {
	form := url.Values{}
	form.Add("username", user)
	form.Add("password", pass)
	form.Add("client_id", "admin-cli")
	form.Add("grant_type", "password")

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s", c.URL, authURL),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return errors.Wrap(err, "error creating login request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrap(err, "error performing token request")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return errors.Wrap(err, "error reading token response")
	}

	tokenRes := &types.TokenResponse{}
	err = json.Unmarshal(body, tokenRes)
	if err != nil {
		return errors.Wrap(err, "error parsing token response")
	}

	if tokenRes.Error != "" {
		logrus.Errorf("error with request: " + tokenRes.ErrorDescription)
		return errors.New(tokenRes.ErrorDescription)
	}

	c.token = tokenRes.AccessToken

	return nil
}

// defaultRequester returns a default client for requesting http endpoints
func defaultRequester() Requester {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

//go:generate moq -out keycloakClient_moq.go . KeycloakInterface

type KeycloakInterface interface {
	Ping() error

	CreateRealm(realm *types.KeycloakRealm) (string, error)
	GetRealm(realmName string) (*types.KeycloakRealm, error)
	UpdateRealm(specRealm *types.KeycloakRealm) error
	DeleteRealm(realmName string) error
	ListRealms() ([]*types.KeycloakAPIRealm, error)

	CreateClient(client *types.KeycloakAPIClient, realmName string) (string, error)
	GetClient(clientID, realmName string) (*types.KeycloakAPIClient, error)
	GetClientSecret(clientID, realmName string) (string, error)
	GetClientInstall(clientID, realmName string) ([]byte, error)
	UpdateClient(specClient *types.KeycloakAPIClient, realmName string) error
	DeleteClient(clientID, realmName string) error
	ListClients(realmName string) ([]*types.KeycloakAPIClient, error)

	CreateUser(user *types.KeycloakAPIUser, realmName string) (string, error)
	CreateFederatedIdentity(fid types.FederatedIdentity, userID string, realmName string) (string, error)
	RemoveFederatedIdentity(fid types.FederatedIdentity, userID string, realmName string) error
	GetUserFederatedIdentities(userName string, realmName string) ([]types.FederatedIdentity, error)
	UpdatePassword(user *types.KeycloakAPIUser, realmName, newPass string) error
	FindUserByEmail(email, realm string) (*types.KeycloakAPIUser, error)
	FindUserByUsername(name, realm string) (*types.KeycloakAPIUser, error)
	GetUser(userID, realmName string) (*types.KeycloakAPIUser, error)
	UpdateUser(specUser *types.KeycloakAPIUser, realmName string) error
	DeleteUser(userID, realmName string) error
	ListUsers(realmName string) ([]*types.KeycloakAPIUser, error)
	ListUsersInGroup(realmName, groupID string) ([]*types.KeycloakAPIUser, error)
	AddUserToGroup(realmName, userID, groupID string) error
	DeleteUserFromGroup(realmName, userID, groupID string) error

	FindGroupByName(groupName string, realmName string) (*Group, error)
	FindGroupByPath(groupPath, realmName string) (*Group, error)
	CreateGroup(group string, realmName string) (string, error)
	MakeGroupDefault(groupID string, realmName string) error
	ListDefaultGroups(realmName string) ([]*Group, error)
	SetGroupChild(groupID, realmName string, childGroup *Group) error

	CreateGroupClientRole(role *types.KeycloakUserRole, realmName, clientID, groupID string) (string, error)
	ListGroupClientRoles(realmName, clientID, groupID string) ([]*types.KeycloakUserRole, error)
	FindGroupClientRole(realmName, clientID, groupID string, predicate func(*types.KeycloakUserRole) bool) (*types.KeycloakUserRole, error)
	ListAvailableGroupClientRoles(realmName, clientID, groupID string) ([]*types.KeycloakUserRole, error)
	FindAvailableGroupClientRole(realmName, clientID, groupID string, predicate func(*types.KeycloakUserRole) bool) (*types.KeycloakUserRole, error)

	CreateGroupRealmRole(role *types.KeycloakUserRole, realmName, groupID string) (string, error)
	ListGroupRealmRoles(realmName, groupID string) ([]*types.KeycloakUserRole, error)
	ListAvailableGroupRealmRoles(realmName, groupID string) ([]*types.KeycloakUserRole, error)

	CreateIdentityProvider(identityProvider *types.KeycloakIdentityProvider, realmName string) (string, error)
	GetIdentityProvider(alias, realmName string) (*types.KeycloakIdentityProvider, error)
	UpdateIdentityProvider(specIdentityProvider *types.KeycloakIdentityProvider, realmName string) error
	DeleteIdentityProvider(alias, realmName string) error
	ListIdentityProviders(realmName string) ([]*types.KeycloakIdentityProvider, error)

	CreateUserClientRole(role *types.KeycloakUserRole, realmName, clientID, userID string) (string, error)
	ListUserClientRoles(realmName, clientID, userID string) ([]*types.KeycloakUserRole, error)
	ListAvailableUserClientRoles(realmName, clientID, userID string) ([]*types.KeycloakUserRole, error)
	DeleteUserClientRole(role *types.KeycloakUserRole, realmName, clientID, userID string) error

	CreateUserRealmRole(role *types.KeycloakUserRole, realmName, userID string) (string, error)
	ListUserRealmRoles(realmName, userID string) ([]*types.KeycloakUserRole, error)
	ListAvailableUserRealmRoles(realmName, userID string) ([]*types.KeycloakUserRole, error)
	DeleteUserRealmRole(role *types.KeycloakUserRole, realmName, userID string) error

	CreateAuthenticationFlow(authFlow AuthenticationFlow, realmName string) (string, error)
	ListAuthenticationFlows(realmName string) ([]*AuthenticationFlow, error)
	FindAuthenticationFlowByAlias(flowAlias, realmName string) (*AuthenticationFlow, error)
	AddExecutionToAuthenticatonFlow(flowAlias, realmName string, providerID string, requirement Requirement) error

	ListAuthenticationExecutionsForFlow(flowAlias, realmName string) ([]*types.AuthenticationExecutionInfo, error)
	FindAuthenticationExecutionForFlow(flowAlias, realmName string, predicate func(*types.AuthenticationExecutionInfo) bool) (*types.AuthenticationExecutionInfo, error)
	UpdateAuthenticationExecutionForFlow(flowAlias, realmName string, execution *types.AuthenticationExecutionInfo) error

	CreateAuthenticatorConfig(authenticatorConfig *types.AuthenticatorConfig, realmName, executionID string) (string, error)
	GetAuthenticatorConfig(configID, realmName string) (*types.AuthenticatorConfig, error)
	UpdateAuthenticatorConfig(authenticatorConfig *types.AuthenticatorConfig, realmName string) error
	DeleteAuthenticatorConfig(configID, realmName string) error

	ListOfActivesUsersPerRealm(realmName, dateFrom string, max int) ([]Users, error)
	UpdateEventsConfig(realmName string, enabledEventTypes, eventsListeners []string) error
}

//go:generate moq -out keycloakClientFactory_moq.go . KeycloakClientFactory

//KeycloakClientFactory interface
type KeycloakClientFactory interface {
	AuthenticatedClient(kc types.Keycloak) (KeycloakInterface, error)
}

type LocalConfigKeycloakFactory struct {
}

// AuthenticatedClient returns an authenticated client for requesting endpoints from the Keycloak api
func (i *LocalConfigKeycloakFactory) AuthenticatedClient(kc types.Keycloak) (KeycloakInterface, error) {
	config, err := config2.GetConfig()
	if err != nil {
		return nil, err
	}

	secretClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	adminCreds, err := secretClient.CoreV1().Secrets(kc.Namespace).Get(context.TODO(), kc.Status.CredentialSecret, v12.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the admin credentials")
	}
	user := string(adminCreds.Data[types.AdminUsernameProperty])
	pass := string(adminCreds.Data[types.AdminPasswordProperty])
	url := kc.Status.ExternalURL
	client := &Client{
		URL:       url,
		requester: defaultRequester(),
	}
	if err := client.login(user, pass); err != nil {
		return nil, err
	}
	return client, nil
}
