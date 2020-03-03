package common

import (
	goctx "context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"net/http"
	"net/url"
	"strings"

	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

const (
	openshiftAuthenticationNamespace = "openshift-authentication"
	openshiftOAuthRouteName          = "oauth-openshift"
	keycloakRouteName                = "keycloak-edge"
	keycloakNamespace                = "rhsso"

	defaultCodeReadyRouteName = "codeready"
	defaultCodeReadyNamespace = "codeready-workspaces"

	// a token to indicate we are not a browser that got tricked into requesting basic auth
	CSRFTokenHeader = "X-CSRF-Token"
)

func TestIntegreatlyUserPermissions(t *testing.T, ctx *TestingContext) {
	// get the RHMI custom resource to check what storage type is being used
	rhmi := &v1alpha1.RHMI{}
	ns := fmt.Sprintf("%soperator", namespacePrefix)
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: ns}, rhmi); err != nil {
		t.Fatal("error getting RHMI CR:", err)
	}

	// get routes
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: openshiftOAuthRouteName, Namespace: openshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}
	keycloakRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: keycloakRouteName, Namespace: fmt.Sprintf("%s%s", namespacePrefix, keycloakNamespace)}, keycloakRoute); err != nil {
		t.Fatal("error getting KeyCloak Route: ", err)
	}
	codereadyRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: defaultCodeReadyRouteName, Namespace: fmt.Sprintf("%s%s", namespacePrefix, defaultCodeReadyNamespace)}, codereadyRoute); err != nil {
		t.Fatal("error occured trying to get codeready route: ", err)
	}

	// declare keycloak edge url
	requestURL := fmt.Sprintf("https://%s/auth/realms/testing-idp/protocol/openid-connect/token", keycloakRoute.Spec.Host)
	requestHeaders := http.Header{}

	// declare user values
	adminUserValues := buildUserValues("customer-admin01")
	testUserOneValues := buildUserValues("test-user01")
	testUserTwoValues := buildUserValues("test-user02")

	// get user tokens
	adminUserToken, err := requestToken(requestHeaders, adminUserValues, requestURL)
	if err != nil {
		t.Fatal("error occurred trying to auth admin user: ", err)
	}
	testUserOneToken, err := requestToken(requestHeaders, testUserOneValues, requestURL)
	if err != nil {
		t.Fatal("error occurred trying to auth test user 1: ", err)
	}
	testUserTwoToken, err := requestToken(requestHeaders, testUserTwoValues, requestURL)
	if err != nil {
		t.Fatal("error occurred trying to auth test user 2: ", err)
	}

	fmt.Println("dedicated admin user -> ", adminUserToken)
	fmt.Println("test user 1 -> ", testUserOneToken)
	fmt.Println("test user 2 -> ", testUserTwoToken)

	if err := getCodeReadyWorkspaces(fmt.Sprintf("http://%s/api/workspace", codereadyRoute.Spec.Host), testUserOneToken); err != nil {
		t.Fatal("error occurred getting code ready workspaces: ", err)
	}

	// return fatal to print logs during development
	t.Fatal("rEtuRN f0r ouTpUTz")
}

func getCodeReadyWorkspaces(requestURL string, bearerToken string) error {
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("error occurred on new request: %w", err)
	}

	// add authorization header to the req
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

	// declare default transport
	rt := http.DefaultTransport.(*http.Transport).Clone()
	rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// make request round trip
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("error occurred on round trip: %w", err)
	}

	fmt.Println("resp -> ", resp)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("error occurred decoding response: %w", err)
	}

	fmt.Println("result -> ", result)

	return nil
}

func requestToken(requestHeaders http.Header, requestValues url.Values, requestURL string) (string, error) {
	// encode request values
	requestParams := strings.NewReader(requestValues.Encode())

	// create new http request
	req, err := http.NewRequest("POST", requestURL, requestParams)
	if err != nil {
		return "", fmt.Errorf("error occurred on new request: %w", err)
	}

	// add potential headers
	for k, v := range requestHeaders {
		req.Header[k] = v
	}

	// add required headers
	req.Header.Set(CSRFTokenHeader, "1")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// declare default transport
	rt := http.DefaultTransport.(*http.Transport).Clone()
	rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// make request round trip
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return "", fmt.Errorf("error occurred on round trip: %w", err)
	}

	// decode result
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("error occurred decoding response: %w", err)
	}

	// nil check and return access token
	if result["access_token"] != nil {
		return fmt.Sprintf("%s", result["access_token"]), nil
	}
	return "", fmt.Errorf("access token is nil")
}

func buildUserValues(name string) url.Values {
	return url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"password":   {"Password1"},
		"username":   {name},
	}
}
