package common

import (
	"fmt"
	"testing"
	"time"

	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"

	"github.com/integr8ly/integreatly-operator/test/resources"
	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	v12 "github.com/openshift/api/authorization/v1"
	configv1 "github.com/openshift/api/config/v1"
	v1 "github.com/openshift/api/route/v1"
	userv1 "github.com/openshift/api/user/v1"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	TestingIDPRealm                = "testing-idp"
	defaultDedicatedAdminName      = "customer-admin"
	defaultNumberOfTestUsers       = 2
	defaultNumberOfDedicatedAdmins = 2
	defaultSecret                  = "rhmiForeva"
	DefaultTestUserName            = "test-user"
	DefaultPassword                = "Password1"
)

var (
	keycloakClientName           = fmt.Sprintf("%s-client", TestingIDPRealm)
	keycloakClientNamespace      = RHSSOProductNamespace
	clusterOauthClientSecretName = fmt.Sprintf("idp-%s", TestingIDPRealm)
	idpCAName                    = fmt.Sprintf("idp-ca-%s", TestingIDPRealm)
)

type TestUser struct {
	UserName  string
	FirstName string
	LastName  string
}

// creates testing idp
func createTestingIDP(t *testing.T, ctx context.Context, client dynclient.Client, kubeConfig *rest.Config, hasSelfSignedCerts bool) error {

	// checks if the IDP is created already
	if hasIDPCreated(ctx, client, t) {
		t.Log("not creating IDP, it's already created")
		return nil
	}

	rhmiCR, err := GetRHMI(client, true)
	if err != nil {
		return fmt.Errorf("error occurred while getting rhmi cr: %w", err)
	}
	masterURL := rhmiCR.Spec.MasterURL

	// create dedicated admins group is it doesnt exist
	if !hasDedicatedAdminGroup(ctx, client) {
		if err := setupDedicatedAdminGroup(ctx, client); err != nil {
			return fmt.Errorf("error occurred while creating dedicated admin group: %w", err)
		}
	}

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := client.Get(ctx, types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		return fmt.Errorf("error occurred while getting Openshift Oauth Route: %w ", err)
	}
	// create keycloak client secret
	if err := createClientSecret(ctx, client, []byte(defaultSecret)); err != nil {
		return fmt.Errorf("error occurred while setting up testing idp client secret: %w", err)
	}

	// create keycloak realm
	if err := createKeycloakRealm(ctx, client, rhmiCR.Name); err != nil {
		return fmt.Errorf("error occurred while setting up keycloak realm: %w", err)
	}

	// Delete current client to ensure new created client secret is correct
	if err := deleteKeycloakClient(ctx, client); err != nil {
		return err
	}

	// create keycloak client
	if err := createKeycloakClient(ctx, client, oauthRoute.Spec.Host, rhmiCR.Name); err != nil {
		return fmt.Errorf("error occurred while setting up keycloak client: %w", err)
	}

	if err := ensureKeycloakClientIsReady(ctx, client); err != nil {
		return fmt.Errorf("error occurred while waiting on keycloak client: %w", err)
	}

	// create keycloak rhmi developer users
	if err := createKeycloakUsers(ctx, client, rhmiCR.Name); err != nil {
		return fmt.Errorf("error occurred while setting up keycloak users: %w", err)
	}

	if hasSelfSignedCerts {
		if err := setupIDPConfig(ctx, client); err != nil {
			return fmt.Errorf("error occurred while updating openshift config: %w", err)
		}
	}

	// create idp in cluster oauth
	if err := addIDPToOauth(ctx, client, hasSelfSignedCerts); err != nil {
		return fmt.Errorf("error occurred while adding testing idp to cluster config: %w", err)
	}

	// add users to dedicated admin users
	if err := addDedicatedAdminUsers(ctx, client, defaultNumberOfDedicatedAdmins); err != nil {
		return fmt.Errorf("error occurred while adding users to dedicated admin group: %w", err)
	}

	// ensure oauth has redeployed
	if err := waitForOauthDeployment(ctx, client); err != nil {
		return fmt.Errorf("error occurred while waiting for oauth deployment: %w", err)
	}
	// ensure the IDP is available in OpenShift
	err = wait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
		// Use a temporary HTTP client to avoid polluting testing client
		tempHTTPClient, err := NewTestingHTTPClient(kubeConfig)
		if err != nil {
			return false, fmt.Errorf("failed to create temporary client for idp setup: %w", err)
		}

		dedicatedAdminUsername := fmt.Sprintf("%s-%d", defaultDedicatedAdminName, defaultNumberOfDedicatedAdmins-1)
		authErr := resources.DoAuthOpenshiftUser(fmt.Sprintf("https://%s/auth/login", masterURL), dedicatedAdminUsername, DefaultPassword, tempHTTPClient, TestingIDPRealm, t)
		if authErr != nil {
			t.Logf("Error while checking IDP is setup, retrying: %+v", authErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check for openshift idp: %w", err)
	}
	return nil
}

func waitForOauthDeployment(ctx context.Context, client dynclient.Client) error {
	err := wait.PollImmediate(time.Second*5, time.Minute*1, func() (done bool, err error) {
		oauthDeployment := &appsv1.Deployment{}
		if err := client.Get(ctx, types.NamespacedName{Name: "oauth-openshift", Namespace: "openshift-authentication"}, oauthDeployment); err != nil {
			return true, fmt.Errorf("error occurred while getting dedicated admin group")
		}

		if oauthDeployment.Status.AvailableReplicas > 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error occurred while polling oauth deployment: %w", err)
	}
	return nil
}

// add users to dedicated admin users
func addDedicatedAdminUsers(ctx context.Context, client dynclient.Client, numberOfAdmins int) error {

	// populate admin users
	var adminUsers []string
	postfix := 0
	for postfix < numberOfAdmins {
		user := fmt.Sprintf("%s-%d", defaultDedicatedAdminName, postfix)
		adminUsers = append(adminUsers, user)
		postfix++
	}

	err := createOrUpdateDedicatedAdminGroupCR(ctx, client, adminUsers)
	if err != nil {
		return fmt.Errorf("error occurred while creating or updating dedicated admin group: %w", err)
	}

	return nil
}

func createOrUpdateDedicatedAdminGroupCR(ctx context.Context, client dynclient.Client, adminUsers []string) error {
	dedicatedAdminGroup := &userv1.Group{}
	if err := client.Get(ctx, types.NamespacedName{Name: "dedicated-admins"}, dedicatedAdminGroup); err != nil {
		return fmt.Errorf("error occurred while getting dedicated admin group")
	}

	// add admin users to group
	if _, err := controllerutil.CreateOrUpdate(ctx, client, dedicatedAdminGroup, func() error {
		for _, user := range adminUsers {
			if !contains(dedicatedAdminGroup.Users, user) {
				dedicatedAdminGroup.Users = append(dedicatedAdminGroup.Users, user)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// create idp config with self signed cert
func setupIDPConfig(ctx context.Context, client dynclient.Client) error {
	routerSecret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: "router-ca", Namespace: "openshift-ingress-operator"}, routerSecret); err != nil {
		return fmt.Errorf("error occurred while getting router ca: %w", err)
	}
	tlsCrt := routerSecret.Data["tls.crt"]

	idpConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      idpCAName,
			Namespace: "openshift-config",
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, client, idpConfigMap, func() error {
		if idpConfigMap.Data == nil {
			idpConfigMap.Data = map[string]string{}
		}
		idpConfigMap.Data["ca.crt"] = string(tlsCrt)
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating openshift cluster oauth: %w", err)
	}
	return nil
}

// add idp to cluster oauth
func addIDPToOauth(ctx context.Context, client dynclient.Client, hasSelfSignedCerts bool) error {
	// setup identity provider ca name
	identityProviderCA := ""
	if hasSelfSignedCerts {
		identityProviderCA = idpCAName
	}

	clusterOauth := &configv1.OAuth{}
	if err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, clusterOauth); err != nil {
		return fmt.Errorf("error occurred while getting cluster oauth: %w", err)
	}

	keycloakRoute := &v1.Route{}
	if err := client.Get(ctx, types.NamespacedName{Name: "keycloak-edge", Namespace: RHSSOProductNamespace}, keycloakRoute); err != nil {
		return fmt.Errorf("error occurred while getting Keycloak Edge Route: %w ", err)
	}
	identityProviderIssuer := fmt.Sprintf("https://%s/auth/realms/%s", keycloakRoute.Spec.Host, TestingIDPRealm)

	// check if identity providers is nil or contains testing IDP
	identityProviders := clusterOauth.Spec.IdentityProviders
	idpAlreadySetUpByScript := false
	idpIndex := 0
	if identityProviders != nil {
		for index, providers := range identityProviders {
			if providers.Name == TestingIDPRealm {
				idpAlreadySetUpByScript = true
				idpIndex = index
				break
			}
		}
	}

	// create identity provider
	testingIdentityProvider := &configv1.IdentityProvider{
		Name:          TestingIDPRealm,
		MappingMethod: "claim",
		IdentityProviderConfig: configv1.IdentityProviderConfig{
			Type: configv1.IdentityProviderTypeOpenID,
			OpenID: &configv1.OpenIDIdentityProvider{
				ClientID: "openshift",
				ClientSecret: configv1.SecretNameReference{
					Name: clusterOauthClientSecretName,
				},
				CA: configv1.ConfigMapNameReference{
					Name: identityProviderCA,
				},
				Issuer: identityProviderIssuer,
				Claims: configv1.OpenIDClaims{
					Email: []string{
						"email",
					},
					Name: []string{
						"name",
					},
					PreferredUsername: []string{
						"preferred_username",
					},
				},
			},
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, client, clusterOauth, func() error {
		if clusterOauth.Spec.IdentityProviders == nil {
			clusterOauth.Spec.IdentityProviders = []configv1.IdentityProvider{}
		}
		// If idp is already set up - ensure using the correct client secret generated from test
		if idpAlreadySetUpByScript {
			clusterOauth.Spec.IdentityProviders[idpIndex].OpenID.ClientSecret.Name = clusterOauthClientSecretName
		} else {
			clusterOauth.Spec.IdentityProviders = append(clusterOauth.Spec.IdentityProviders, *testingIdentityProvider)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating openshift cluster oauth: %w", err)
	}
	return nil
}

// creates secret to be used by keycloak client
func createClientSecret(ctx context.Context, client dynclient.Client, clientSecret []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterOauthClientSecretName,
			Namespace: "openshift-config",
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, client, secret, func() error {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data["clientSecret"] = clientSecret
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating openshift client secret: %w", err)
	}
	return nil
}

// create rhmi developer keycloak users
func createKeycloakUsers(ctx context.Context, client dynclient.Client, installationName string) error {
	// populate users to be created
	var testUsers []TestUser
	postfix := 0
	// build rhmi developer users
	for postfix < defaultNumberOfTestUsers {
		user := TestUser{
			UserName:  fmt.Sprintf("%s-%d", DefaultTestUserName, postfix),
			FirstName: "Test",
			LastName:  fmt.Sprintf("User %d", postfix),
		}
		testUsers = append(testUsers, user)
		postfix++
	}
	postfix = 0
	// build dedicated admin users
	for postfix < defaultNumberOfDedicatedAdmins {
		user := TestUser{
			UserName:  fmt.Sprintf("%s-%d", defaultDedicatedAdminName, postfix),
			FirstName: defaultDedicatedAdminName,
			LastName:  fmt.Sprintf("User %d", postfix),
		}
		testUsers = append(testUsers, user)
		postfix++
	}

	err := createOrUpdateKeycloakUserCR(ctx, client, testUsers, installationName)
	if err != nil {
		return fmt.Errorf("error occurred while creating or updating keycloak user: %w", err)
	}

	return nil
}

func createOrUpdateKeycloakUserCR(ctx context.Context, client dynclient.Client, testUsers []TestUser, installationName string) error {
	// create rhmi developer users from test users
	for _, user := range testUsers {
		keycloakUser := &v1alpha1.KeycloakUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", TestingIDPRealm, user.UserName),
				Namespace: RHSSOProductNamespace,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, client, keycloakUser, func() error {
			keycloakUser.Annotations = map[string]string{
				"integreatly-namespace": RHMIOperatorNamespace,
				"integreatly-name":      installationName,
			}
			keycloakUser.Spec = v1alpha1.KeycloakUserSpec{
				RealmSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"sso": TestingIDPRealm,
					},
				},
				User: v1alpha1.KeycloakAPIUser{
					ID:        keycloakUser.Spec.User.ID,
					FirstName: user.FirstName,
					LastName:  user.LastName,
					UserName:  user.UserName,
					Email:     fmt.Sprintf("%s@example.com", user.UserName),
					ClientRoles: map[string][]string{
						"account": {
							"manage-account",
							"view-profile",
						},
						"broker": {
							"read-token",
						},
					},
					EmailVerified: true,
					Enabled:       true,
					Credentials: []v1alpha1.KeycloakCredential{
						{
							Type:  "password",
							Value: DefaultPassword,
						},
					},
				},
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// polls the keycloak client until it is ready
func ensureKeycloakClientIsReady(ctx context.Context, client dynclient.Client) error {
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		keycloakClient := &v1alpha1.KeycloakClient{}

		if err := client.Get(ctx, types.NamespacedName{Name: keycloakClientName, Namespace: keycloakClientNamespace}, keycloakClient); err != nil {
			return false, fmt.Errorf("error occurred while getting keycloak client")
		}
		if keycloakClient.Status.Ready {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error occurred while polling keycloak client: %w", err)
	}
	return nil
}

// creates keycloak client
func createKeycloakClient(ctx context.Context, client dynclient.Client, oauthURL string, installationName string) error {
	keycloakClient := &v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakClientName,
			Namespace: keycloakClientNamespace,
			Annotations: map[string]string{
				"integreatly-namespace": RHMIOperatorNamespace,
				"integreatly-name":      installationName,
			},
		},
		Spec: v1alpha1.KeycloakClientSpec{
			RealmSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"sso": TestingIDPRealm,
				},
			},
			Client: &v1alpha1.KeycloakAPIClient{
				// ID:                      "openshift",
				ClientID:                "openshift",
				Enabled:                 true,
				ClientAuthenticatorType: "client-secret",
				Secret:                  defaultSecret,
				RootURL:                 fmt.Sprintf("https://%s", oauthURL),
				RedirectUris: []string{
					fmt.Sprintf("https://%s/oauth2callback/%s", oauthURL, TestingIDPRealm),
				},
				WebOrigins: []string{
					fmt.Sprintf("https://%s", oauthURL),
					fmt.Sprintf("https://%s/*", oauthURL),
				},
				StandardFlowEnabled:       true,
				DirectAccessGrantsEnabled: true,
				FullScopeAllowed:          true,
				ProtocolMappers: []v1alpha1.KeycloakProtocolMapper{
					{
						Config: map[string]string{
							"access.token.claim":   "true",
							"claim.name":           "given_name",
							"id.token.claim":       "true",
							"jsonType.label":       "String",
							"user.attribute":       "firstName",
							"userinfo.token.claim": "true",
						},
						ConsentRequired: true,
						ConsentText:     "${givenName}",
						Name:            "given name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
					},
					{
						Config: map[string]string{
							"access.token.claim":   "true",
							"id.token.claim":       "true",
							"userinfo.token.claim": "true",
						},
						ConsentRequired: true,
						ConsentText:     "${fullName}",
						Name:            "full name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-full-name-mapper",
					},
					{
						Config: map[string]string{
							"access.token.claim":   "true",
							"claim.name":           "family_name",
							"id.token.claim":       "true",
							"jsonType.label":       "String",
							"user.attribute":       "lastName",
							"userinfo.token.claim": "true",
						},
						ConsentRequired: true,
						ConsentText:     "${familyName}",
						Name:            "family name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
					},
					{
						Config: map[string]string{
							"attribute.name":       "Role",
							"attribute.nameformat": "Basic",
							"single":               "false",
						},
						ConsentText:    "${familyName}",
						Name:           "role list",
						Protocol:       "saml",
						ProtocolMapper: "saml-role-list-mapper",
					},
					{
						Config: map[string]string{
							"access.token.claim":   "true",
							"claim.name":           "email",
							"id.token.claim":       "true",
							"jsonType.label":       "String",
							"user.attribute":       "email",
							"userinfo.token.claim": "true",
						},
						ConsentRequired: true,
						ConsentText:     "${email}",
						Name:            "email",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
					},
					{
						Config: map[string]string{
							"access.token.claim":   "true",
							"claim.name":           "preferred_username",
							"id.token.claim":       "true",
							"jsonType.label":       "String",
							"user.attribute":       "username",
							"userinfo.token.claim": "true",
						},
						ConsentText:    "n.a.",
						Name:           "username",
						Protocol:       "openid-connect",
						ProtocolMapper: "oidc-usermodel-property-mapper",
					},
				},
				Access: map[string]bool{
					"configure": true,
					"manage":    true,
					"view":      true,
				},
			},
		},
	}

	if err := client.Create(ctx, keycloakClient); err != nil {
		return fmt.Errorf("error occurred while creating keycloak client: %w", err)
	}
	return nil
}

func deleteKeycloakClient(ctx context.Context, client dynclient.Client) error {
	err := wait.PollImmediate(time.Second*2, time.Minute*2, func() (done bool, err error) {
		if err := client.Delete(ctx, &v1alpha1.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      keycloakClientName,
				Namespace: keycloakClientNamespace,
			},
		}); err != nil {
			if !k8errors.IsNotFound(err) {
				return false, nil
			}

			if k8errors.IsNotFound(err) {
				return true, nil
			}
		}

		return false, nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete testing idp keycloak client: %s", err)
	}

	return nil
}

// create keycloak realm
func createKeycloakRealm(ctx context.Context, client dynclient.Client, installationName string) error {
	keycloakRealm := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestingIDPRealm,
			Namespace: RHSSOProductNamespace,
			Labels: map[string]string{
				"sso": TestingIDPRealm,
			},
		},
	}

	keycloakRealmSpec := v1alpha1.KeycloakRealmSpec{
		InstanceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"sso": "integreatly",
			},
		},
		Realm: &v1alpha1.KeycloakAPIRealm{
			ID:          TestingIDPRealm,
			Realm:       TestingIDPRealm,
			Enabled:     true,
			DisplayName: "Testing IDP",
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, client, keycloakRealm, func() error {
		keycloakRealm.Annotations = map[string]string{
			"integreatly-namespace": RHMIOperatorNamespace,
			"integreatly-name":      installationName,
		}
		keycloakRealm.Spec = keycloakRealmSpec
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating keycloak realm: %w", err)
	}
	return nil
}

// creates dedicated admin group
func setupDedicatedAdminGroup(ctx context.Context, client dynclient.Client) error {
	dedicatedAdminGroup := &userv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-admins",
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, client, dedicatedAdminGroup, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating dedicated admins group : %w", err)
	}

	// create cluster role
	if _, err := controllerutil.CreateOrUpdate(ctx, client, dedicatedAdminClusterRole(), func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating dedicated admin cluster role: %w", err)
	}

	// create cluster role binding
	if _, err := controllerutil.CreateOrUpdate(ctx, client, dedicatedAdminClusterRoleBindingCluster(), func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating dedicated admin cluster role binding: %w", err)
	}
	return nil
}

func hasIDPCreated(ctx context.Context, client dynclient.Client, t *testing.T) bool {
	clusterOauth := &configv1.OAuth{}
	if err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, clusterOauth); err != nil {
		t.Logf("error occurred while getting cluster oauth: %w", err)
	}

	idpExists := false
	identityProviders := clusterOauth.Spec.IdentityProviders
	if identityProviders != nil {
		for _, providers := range identityProviders {
			if providers.Name == TestingIDPRealm {
				idpExists = true
			}
		}
	}

	return idpExists
}

// checks to see if a dedicated admin group exists
func hasDedicatedAdminGroup(ctx context.Context, client dynclient.Client) bool {
	dedicatedAdminGroup := &userv1.Group{}
	if err := client.Get(ctx, types.NamespacedName{Name: "dedicated-admins"}, dedicatedAdminGroup); err != nil {
		return false
	}
	return true
}

func dedicatedAdminClusterRoleBindingCluster() *v12.ClusterRoleBinding {
	return &v12.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-dedicated-cluster",
			Namespace: "dedicated-admin",
		},
		Subjects: []corev1.ObjectReference{
			{
				Kind:       "Group",
				APIVersion: "rbac.authorization.k8s.io/v1",
				Name:       "dedicated-admins",
			},
		},
		RoleRef: corev1.ObjectReference{
			Kind:       "ClusterRole",
			Name:       "dedicated-admin-cluster",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
	}
}

func dedicatedAdminClusterRole() *v12.ClusterRole {
	return &v12.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-admin-cluster",
		},
		Rules: []v12.PolicyRule{
			{
				Verbs: []string{
					"create",
				},
				APIGroups: []string{
					"",
					"project.openshift.io",
				},
				Resources: []string{
					"projectrequests",
				},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{
					"project.openshift.io",
				},
				Resources: []string{
					"project",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"update",
					"patch",
				},
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"namespaces",
				},
			},
		},
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
