package resources

import (
	"context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	SSOLabelKey   = "sso"
	SSOLabelValue = "integreatly"
)

func CreateRHSSOClient(clientID string, clientSecret string, serverClient k8sclient.Client, configManager config.ConfigReadWriter, ctx context.Context, installation integreatlyv1alpha1.RHMI, log l.Logger) (integreatlyv1alpha1.StatusPhase, error) {

	rhssoConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	rhssoNamespace := rhssoConfig.GetNamespace()
	rhssoRealm := rhssoConfig.GetRealm()

	if rhssoNamespace == "" || rhssoRealm == "" {
		log.Warningf("Cannot configure SSO integration without SSO", l.Fields{"ns": rhssoNamespace, "realm": rhssoRealm})
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	kcClient := &keycloak.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientID,
			Namespace: rhssoNamespace,
		},
	}

	// keycloak-operator sets the spec.client.id, we need to preserve that value
	apiClientID := ""
	err = serverClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: rhssoNamespace,
		Name:      clientID,
	}, kcClient)
	if err == nil {
		apiClientID = kcClient.Spec.Client.ID
	}

	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcClient, func() error {
		kcClient.Spec = getKeycloakClientSpec(apiClientID, clientID, clientSecret, &installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create/update 3scale keycloak client: %w operation: %v", err, opRes)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getKeycloakClientSpec(apiClientID, clientID, clientSecret string, installation *integreatlyv1alpha1.RHMI) keycloak.KeycloakClientSpec {
	fullScopeAllowed := true
	var client *keycloak.KeycloakAPIClient

	protocolMappers := []keycloak.KeycloakProtocolMapper{
		{
			Name:            "given name",
			Protocol:        "openid-connect",
			ProtocolMapper:  "oidc-usermodel-property-mapper",
			ConsentRequired: true,
			ConsentText:     "${givenName}",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "firstName",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "given_name",
				"jsonType.label":       "String",
			},
		},
		{
			Name:            "email verified",
			Protocol:        "openid-connect",
			ProtocolMapper:  "oidc-usermodel-property-mapper",
			ConsentRequired: true,
			ConsentText:     "${emailVerified}",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "emailVerified",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "email_verified",
				"jsonType.label":       "String",
			},
		},
		{
			Name:            "full name",
			Protocol:        "openid-connect",
			ProtocolMapper:  "oidc-full-name-mapper",
			ConsentRequired: true,
			ConsentText:     "${fullName}",
			Config: map[string]string{
				"id.token.claim":     "true",
				"access.token.claim": "true",
			},
		},
		{
			Name:            "family name",
			Protocol:        "openid-connect",
			ProtocolMapper:  "oidc-usermodel-property-mapper",
			ConsentRequired: true,
			ConsentText:     "${familyName}",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "lastName",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "family_name",
				"jsonType.label":       "String",
			},
		},
		{
			Name:            "role list",
			Protocol:        "saml",
			ProtocolMapper:  "saml-role-list-mapper",
			ConsentRequired: false,
			ConsentText:     "${familyName}",
			Config: map[string]string{
				"single":               "false",
				"attribute.nameformat": "Basic",
				"attribute.name":       "Role",
			},
		},
		{
			Name:            "email",
			Protocol:        "openid-connect",
			ProtocolMapper:  "oidc-usermodel-property-mapper",
			ConsentRequired: true,
			ConsentText:     "${email}",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "email",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "email",
				"jsonType.label":       "String",
			},
		},
		{
			Name:            "org_name",
			Protocol:        "openid-connect",
			ProtocolMapper:  "oidc-usermodel-property-mapper",
			ConsentRequired: false,
			ConsentText:     "n.a.",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "org_name",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "org_name",
				"jsonType.label":       "String",
			},
		},
	}

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		client = &keycloak.KeycloakAPIClient{
			ID:                      apiClientID,
			ClientID:                clientID,
			Enabled:                 true,
			Secret:                  clientSecret,
			ClientAuthenticatorType: "client-secret",
			RedirectUris: []string{
				fmt.Sprintf("https://3scale-admin.%s/*", installation.Spec.RoutingSubdomain),
			},
			StandardFlowEnabled: true,
			RootURL:             fmt.Sprintf("https://3scale-admin.%s", installation.Spec.RoutingSubdomain),
			FullScopeAllowed:    &fullScopeAllowed,
			Access: map[string]bool{
				"view":      true,
				"configure": true,
				"manage":    true,
			},
			ProtocolMappers: protocolMappers,
		}
	} else {
		client = &keycloak.KeycloakAPIClient{
			ID:       apiClientID,
			ClientID: clientID,
			Enabled:  true,
			RedirectUris: []string{
				fmt.Sprintf("https://%s-admin.%s/*", clientID, installation.Spec.RoutingSubdomain),
			},
			StandardFlowEnabled: true,
			RootURL:             fmt.Sprintf("https://%s-admin.%s", clientID, installation.Spec.RoutingSubdomain),
			FullScopeAllowed:    &fullScopeAllowed,
			Access: map[string]bool{
				"view":      true,
				"configure": true,
				"manage":    true,
			},
			ProtocolMappers: protocolMappers,
		}
	}

	return keycloak.KeycloakClientSpec{
		RealmSelector: &metav1.LabelSelector{
			MatchLabels: getRHSSOInstanceLabels(),
		},
		Client: client,
	}
}

func getRHSSOInstanceLabels() map[string]string {
	return map[string]string{
		SSOLabelKey: SSOLabelValue,
	}
}