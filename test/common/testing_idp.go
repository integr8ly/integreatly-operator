package common

import (
	goctx "context"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	v1 "github.com/openshift/api/route/v1"
	userv1 "github.com/openshift/api/user/v1"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const (
	testingIDPRealm                = "testing-idp"
	defaultTestUserName            = "test-user"
	defaultDedicatedAdminName      = "customer-admin"
	defaultNumberOfTestUsers       = 2
	defaultNumberOfDedicatedAdmins = 2
	defaultSecret                  = "rhmiForeva"
	DefaultPassword                = "Password1"
)

type TestUser struct {
	UserName  string
	FirstName string
	LastName  string
}

// creates testing idp
func createTestingIDP(ctx *TestingContext, httpClient *http.Client) error {
	rhmiCR, err := getRHMI(ctx)
	if err != nil {
		return fmt.Errorf("error occurred while getting rhmi cr: %w", err)
	}

	hasSelfSignedCerts := false
	masterURL := rhmiCR.Spec.MasterURL
	_, err = httpClient.Get(fmt.Sprintf("https://%s", masterURL))
	if err != nil {
		if _, ok := errors.Unwrap(err).(x509.UnknownAuthorityError); !ok {
			return fmt.Errorf("error while performing self-signed certs test request: %w", err)
		}
		hasSelfSignedCerts = true
	}

	// create dedicated admins group is it doesnt exist
	if !hasDedicatedAdminGroup(ctx) {
		if err := setupDedicatedAdminGroup(ctx); err != nil {
			return fmt.Errorf("error occurred while creating dedicated admin group: %w", err)
		}
	}

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		return fmt.Errorf("error occurred while getting Openshift Oauth Route: %w ", err)
	}

	// create keycloak client secret
	if err := createClientSecret(ctx, []byte(defaultSecret)); err != nil {
		return fmt.Errorf("error occurred while setting up testing idp client secret: %w", err)
	}

	// create keycloak realm
	if err := createKeycloakRealm(ctx); err != nil {
		return fmt.Errorf("error occurred while setting up keycloak realm: %w", err)
	}

	// create keycloak client
	keycloakClientName := fmt.Sprintf("%s-client", testingIDPRealm)
	keycloakClientNamespace := fmt.Sprintf("%srhsso", NamespacePrefix)
	if err := createKeycloakClient(ctx, oauthRoute.Spec.Host, keycloakClientName, keycloakClientNamespace); err != nil {
		return fmt.Errorf("error occurred while setting up keycloak client: %w", err)
	}

	// create keycloak rhmi developer users
	if err := createKeycloakUsers(ctx, keycloakClientName, keycloakClientNamespace); err != nil {
		return fmt.Errorf("error occurred while setting up keycloak users: %w", err)
	}

	if hasSelfSignedCerts {
		if err := setupIDPConfig(ctx); err != nil {
			return fmt.Errorf("error occurred while updating openshift config: %w", err)
		}
	}

	// create idp in cluster oauth
	if err := addIDPToOauth(ctx, hasSelfSignedCerts); err != nil {
		return fmt.Errorf("error occurred while adding testing idp to cluster config: %w", err)
	}

	// add users to dedicated admin users
	if err := addDedicatedAdminUsers(ctx, defaultNumberOfDedicatedAdmins); err != nil {
		return fmt.Errorf("error occurred while adding users to dedicated admin group: %w", err)
	}

	// ensure oauth has redeployed
	if err := waitForOauthDeployment(ctx); err != nil {
		return fmt.Errorf("error occurred while waiting for oauth deployment: %w", err)
	}

	// ensure the IDP is available in OpenShift
	err = wait.PollImmediate(time.Second*10, time.Minute*3, func() (done bool, err error) {
		return resources.OpenshiftIDPCheck(fmt.Sprintf("https://%s/auth/login", masterURL), httpClient)
	})
	if err != nil {
		return fmt.Errorf("failed to check for openshift idp: %w", err)
	}

	return nil
}

func waitForOauthDeployment(ctx *TestingContext) error {
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		oauthDeployment := &appsv1.Deployment{}
		if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "oauth-openshift", Namespace: "openshift-authentication"}, oauthDeployment); err != nil {
			return true, fmt.Errorf("error occurred while getting dedicated admin group")
		}
		if oauthDeployment.Status.UnavailableReplicas == 0 {
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
func addDedicatedAdminUsers(ctx *TestingContext, numberOfAdmins int) error {
	dedicatedAdminGroup := &userv1.Group{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "dedicated-admins"}, dedicatedAdminGroup); err != nil {
		return fmt.Errorf("error occurred while getting dedicated admin group")
	}

	// populate admin users
	var adminUsers []string
	postfix := 0
	for postfix < numberOfAdmins {
		user := fmt.Sprintf("%s-%d", defaultDedicatedAdminName, postfix)
		adminUsers = append(adminUsers, user)
		postfix++
	}

	// add admin users to group
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, dedicatedAdminGroup, func() error {
		for _, user := range adminUsers {
			if !contains(dedicatedAdminGroup.Users, user) {
				dedicatedAdminGroup.Users = append(dedicatedAdminGroup.Users, user)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating dedicated admin group: %w", err)
	}
	return nil
}

// create idp config with self signed cert
func setupIDPConfig(ctx *TestingContext) error {
	routerSecret := &corev1.Secret{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "router-ca", Namespace: "openshift-ingress-operator"}, routerSecret); err != nil {
		return fmt.Errorf("error occurred while getting router ca: %w", err)
	}
	tlsCrt := routerSecret.Data["tls.crt"]

	idpConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("idp-ca-%s", testingIDPRealm),
			Namespace: "openshift-config",
		},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, idpConfigMap, func() error {
		if idpConfigMap == nil {
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
func addIDPToOauth(ctx *TestingContext, hasSelfSignedCerts bool) error {
	// setup identity provider ca name
	identityProviderCA := ""
	if hasSelfSignedCerts {
		identityProviderCA = fmt.Sprintf("idp-ca-%s", testingIDPRealm)
	}

	clusterOauth := &configv1.OAuth{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "cluster"}, clusterOauth); err != nil {
		return fmt.Errorf("error occurred while getting cluster oauth: %w", err)
	}

	keycloakRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "keycloak-edge", Namespace: fmt.Sprintf("%srhsso", NamespacePrefix)}, keycloakRoute); err != nil {
		return fmt.Errorf("error occurred while getting Keycloak Edge Route: %w ", err)
	}
	identityProviderIssuer := fmt.Sprintf("https://%s/auth/realms/testing-idp", keycloakRoute.Spec.Host)

	// check if identity providers is nil or contains testing IDP
	identityProviders := clusterOauth.Spec.IdentityProviders
	if identityProviders != nil {
		for _, providers := range identityProviders {
			if providers.Name == testingIDPRealm {
				return nil
			}
		}
	}

	// create identity provider
	testingIdentityProvider := &configv1.IdentityProvider{
		Name:          testingIDPRealm,
		MappingMethod: "claim",
		IdentityProviderConfig: configv1.IdentityProviderConfig{
			Type: configv1.IdentityProviderTypeOpenID,
			OpenID: &configv1.OpenIDIdentityProvider{
				ClientID: "openshift",
				ClientSecret: configv1.SecretNameReference{
					Name: fmt.Sprintf("idp-%s", testingIDPRealm),
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

	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, clusterOauth, func() error {
		if clusterOauth.Spec.IdentityProviders == nil {
			clusterOauth.Spec.IdentityProviders = []configv1.IdentityProvider{}
		}
		clusterOauth.Spec.IdentityProviders = append(clusterOauth.Spec.IdentityProviders, *testingIdentityProvider)
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating openshift cluster oauth: %w", err)
	}
	return nil
}

// creates secret to be used by keycloak client
func createClientSecret(ctx *TestingContext, clientSecret []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("idp-%s", testingIDPRealm),
			Namespace: "openshift-config",
		},
	}

	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, secret, func() error {
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
func createKeycloakUsers(ctx *TestingContext, keycloakClientName, keycloakClientNamespace string) error {
	if err := ensureKeycloakClientIsReady(ctx, keycloakClientName, keycloakClientNamespace); err != nil {
		return fmt.Errorf("error occurred while waiting on keycloak client: %w", err)
	}

	// populate users to be created
	var testUsers []TestUser
	postfix := 0
	// build rhmi developer users
	for postfix < defaultNumberOfTestUsers {
		user := TestUser{
			UserName:  fmt.Sprintf("%s-%d", defaultTestUserName, postfix),
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
			FirstName: "Test",
			LastName:  fmt.Sprintf("User %d", postfix),
		}
		testUsers = append(testUsers, user)
		postfix++
	}

	// create rhmi developer users from test users
	for _, user := range testUsers {
		keycloakUser := &v1alpha1.KeycloakUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", testingIDPRealm, user.UserName),
				Namespace: fmt.Sprintf("%srhsso", NamespacePrefix),
			},
		}
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, keycloakUser, func() error {
			keycloakUser.Spec = v1alpha1.KeycloakUserSpec{
				RealmSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"sso": testingIDPRealm,
					},
				},
				User: v1alpha1.KeycloakAPIUser{
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
		}); err != nil {
			return fmt.Errorf("error occurred while creating or updating keycloak user: %w", err)
		}
	}
	return nil
}

// polls the keycloak client until it is ready
func ensureKeycloakClientIsReady(ctx *TestingContext, keycloakClientName, keycloakClientNamespace string) error {
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		keycloakClient := &v1alpha1.KeycloakClient{}

		if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: keycloakClientName, Namespace: keycloakClientNamespace}, keycloakClient); err != nil {
			return true, fmt.Errorf("error occurred while getting keycloak client")
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
func createKeycloakClient(ctx *TestingContext, oauthURL, clientName, clientNamespace string) error {
	keycloakClient := &v1alpha1.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientName,
			Namespace: clientNamespace,
		},
	}

	keycloakSpec := v1alpha1.KeycloakClientSpec{
		RealmSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"sso": testingIDPRealm,
			},
		},
		Client: &v1alpha1.KeycloakAPIClient{
			ID:                      "openshift",
			ClientID:                "openshift",
			Enabled:                 true,
			ClientAuthenticatorType: "client-secret",
			Secret:                  defaultSecret,
			RootURL:                 fmt.Sprintf("https://%s", oauthURL),
			RedirectUris: []string{
				fmt.Sprintf("https://%s/oauth2callback/%s", oauthURL, testingIDPRealm),
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
	}

	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, keycloakClient, func() error {
		keycloakClient.Spec = keycloakSpec
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating keycloak client: %w", err)
	}
	return nil
}

// create keycloak realm
func createKeycloakRealm(ctx *TestingContext) error {
	keycloakRealm := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testingIDPRealm,
			Namespace: fmt.Sprintf("%srhsso", NamespacePrefix),
			Labels: map[string]string{
				"sso": testingIDPRealm,
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
			ID:          testingIDPRealm,
			Realm:       testingIDPRealm,
			Enabled:     true,
			DisplayName: "Testing IDP",
		},
	}

	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, keycloakRealm, func() error {
		keycloakRealm.Spec = keycloakRealmSpec
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating keycloak realm: %w", err)
	}
	return nil
}

// creates dedicated admin group
func setupDedicatedAdminGroup(ctx *TestingContext) error {
	dedicatedAdminGroup := &userv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-admins",
		},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, dedicatedAdminGroup, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while creating or updating dedicated admins group : %w", err)
	}
	return nil
}

// checks to see if a dedicated admin group exists
func hasDedicatedAdminGroup(ctx *TestingContext) bool {
	dedicatedAdminGroup := &userv1.Group{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "dedicated-admins"}, dedicatedAdminGroup); err != nil {
		return false
	}
	return true
}

// checks if cluster has self signed certs based on rhmi cr
func hasSelfSignedCerts(ctx *TestingContext) (bool, error) {
	rhmi, err := getRHMI(ctx)
	if err != nil {
		return false, fmt.Errorf("error occurred while getting rhmi cr: %w", err)
	}

	return rhmi.Spec.SelfSignedCerts, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
