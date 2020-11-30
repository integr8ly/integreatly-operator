package common

import (
	"context"
	goctx "context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	brow "github.com/headzoo/surf/browser"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TestAuthThreeScaleUsername = "testauth-threescale"
	threeScaleDashboardURI     = "p/admin/dashboard"
	threeScaleOnboardingURI    = "p/admin/onboarding/wizard/intro"
)

func TestAuthDelayFirstBrokerLogin(t *testing.T, ctx *TestingContext) {

	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	testUser, err := getRandomKeycloakUser(ctx, rhmi.Name)

	if err != nil {
		t.Fatalf("error getting test user: %v", err)
	}

	tsHost := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.Product3Scale].Host
	if tsHost == "" {
		tsHost = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	t.Logf("Three scale admin host %s", tsHost)

	err = ensureKeycloakUserIsReconciled(goctx.TODO(), ctx.Client, testUser.UserName)
	if err != nil {
		t.Fatalf("error occurred while waiting on keycloak user to be reconciled: %v", err)
	}

	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = loginToThreeScale(t, tsHost, testUser.UserName, DefaultPassword, TestingIDPRealm, httpClient)
	if err != nil {
		dumpAuthResources(ctx.Client, t)
		// t.Skip("Skipping due to known flaky behavior, reported in Jira: https://issues.redhat.com/browse/INTLY-10087")
		t.Fatalf("[%s] error logging in to three scale: %v", getTimeStampPrefix(), err)
	}
}

func getRandomKeycloakUser(ctx *TestingContext, installationName string) (*TestUser, error) {
	// create random keycloak user
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	userNamePostfix := r1.Intn(100000)
	testUsers := []TestUser{
		{
			FirstName: TestAuthThreeScaleUsername,
			LastName:  fmt.Sprintf("User %d", userNamePostfix),
			UserName:  fmt.Sprintf("%s-%d", TestAuthThreeScaleUsername, userNamePostfix),
		},
	}
	err := createOrUpdateKeycloakUserCR(goctx.TODO(), ctx.Client, testUsers, installationName)
	if err != nil {
		return nil, fmt.Errorf("error creating test user: %v", err)
	}

	var adminUsers = []string{
		fmt.Sprintf("%s-%d", TestAuthThreeScaleUsername, userNamePostfix),
	}
	err = createOrUpdateDedicatedAdminGroupCR(goctx.TODO(), ctx.Client, adminUsers)
	if err != nil {
		return nil, fmt.Errorf("error adding user to dedicated admin group: %v", err)
	}

	return &testUsers[0], nil
}

// polls the keycloak user until it is ready
func ensureKeycloakUserIsReconciled(ctx context.Context, client dynclient.Client, keycloakUsername string) error {
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		keycloakUser := &keycloak.KeycloakUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", TestingIDPRealm, keycloakUsername),
				Namespace: fmt.Sprintf("%srhsso", NamespacePrefix),
			},
		}

		if err := client.Get(ctx, types.NamespacedName{Name: keycloakUser.GetName(), Namespace: keycloakUser.GetNamespace()}, keycloakUser); err != nil {
			return true, fmt.Errorf("error occurred while getting keycloak user")
		}
		if keycloakUser.Status.Phase == "reconciled" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error occurred while polling keycloak user: %w", err)
	}
	return nil
}

func loginToThreeScale(t *testing.T, tsHost, username, password string, idp string, client *http.Client) error {

	// const variable to validate authentication
	const (
		provisioningAccountTxt = "Your account is being provisioned"
	)

	parsedURL, err := url.Parse(tsHost)
	if err != nil {
		return fmt.Errorf("failed to parse three scale url %s: %s", parsedURL, err)
	}

	if parsedURL.Scheme == "" {
		tsHost = fmt.Sprintf("https://%s", tsHost)
	}

	tsLoginURL := fmt.Sprintf("%v/p/login", tsHost)
	tsDashboardURL := fmt.Sprintf("%v/%s", tsHost, threeScaleDashboardURI)

	t.Logf("Attempting to open threescale login page with url: %s", tsLoginURL)

	browser := surf.NewBrowser()
	browser.SetCookieJar(client.Jar)
	browser.SetTransport(client.Transport)
	browser.SetAttribute(brow.FollowRedirects, true)

	// open threescale login page
	err = browser.Open(tsLoginURL)
	if err != nil {
		return fmt.Errorf("failed to open browser url: %w", err)
	}

	// check if user is already authenticated
	if isUserAuthenticated(browser, tsDashboardURL) {
		return nil
	}

	t.Logf("User is not authenticated yet, going to authenticate through rhsso url  %s", browser.Url().String())

	// click on authenticate through rhsso
	err = authenticateThroughRHSSO(browser)
	if err != nil {
		t.Logf("response %s", browser.Body())
		return err
	}

	// check if user is already authenticated through rhsso
	if isUserAuthenticated(browser, tsDashboardURL) {
		return nil
	}

	t.Logf("User is not authenticated through rhsso yet, going to select the IDP %s", browser.Url().String())

	selectIDPURL := browser.Url().String()
	// select the IDP to authenticate through RHSSO
	err = selectRHSSOIDP(browser, idp)
	if err != nil {
		return err
	}

	// check if user is already authenticated after selecting the IDP
	if isUserAuthenticated(browser, tsDashboardURL) {
		return nil
	}

	t.Logf("User is not authenticated after selecting the IDP yet, going to submit the form to authenticate the user through rhsso %s", browser.Url().String())

	// submit openshift oauth login form
	err = browser.Open(selectIDPURL)
	if err != nil {
		return fmt.Errorf("failed to open selectIDPURL url: %w", err)
	}
	err = resources.OpenshiftClientSubmitForm(browser, username, password, idp, t)
	if err != nil {
		return fmt.Errorf("failed to submit the openshift oauth login: %w", err)
	}

	// check if user is authenticated after submiting rhsso login form
	if isUserAuthenticated(browser, tsDashboardURL) {
		return nil
	}

	// waits until the account is provisioned and user is authenticated in three scale
	err = wait.Poll(time.Second*5, time.Minute*5, func() (done bool, err error) {
		t.Logf("browser URL first %s - URL with requestURI %s, status code %v", browser.Url().String(), browser.Url().RequestURI(), browser.StatusCode())

		// checks if an error happened in the login
		if browser.StatusCode() == 502 {
			t.Logf("Unexpected error, User already authenticated: URL - %s | Request status code - %v", browser.Url().String(), browser.StatusCode())

			err := browser.Open(tsDashboardURL)
			t.Logf("Opened dashboard URL to validate user authentication: URL - %s | DashboardURL - %s", browser.Url().String(), tsDashboardURL)
			if err != nil {
				t.Logf("failed to open dashboard url: %w", err)
				return false, fmt.Errorf("failed to open dashboard url: %w", err)
			}

			// if user is redirected to the login page try again
			if browser.Url().String() == tsLoginURL {
				// click on authenticate through rhsso
				err = authenticateThroughRHSSO(browser)
				if err != nil {
					return false, err
				}

				// check if user is already authenticated through rhsso
				if isUserAuthenticated(browser, tsDashboardURL) {
					return true, nil
				}

				selectIDPURL = browser.Url().String()
				// select the IDP to authenticate through RHSSO
				err = selectRHSSOIDP(browser, idp)
				if err != nil {
					return false, err
				}

				// check if user is already authenticated after selecting the IDP
				if isUserAuthenticated(browser, tsDashboardURL) {
					return true, nil
				}
				t.Logf("authenticate after click on idp url: %s", browser.Url().String())
			}
		}

		if isUserAuthenticated(browser, tsDashboardURL) {
			return true, nil
		}

		browser.Find(fmt.Sprintf("h1:contains('%s')", provisioningAccountTxt)).Each(func(index int, s *goquery.Selection) {

			// gets the refresh url from the meta tag
			browser.Dom().Find("meta").Each(func(index int, s *goquery.Selection) {
				val, exist := s.Attr("content")
				if exist {
					contentValue := strings.Split(val, ";")
					if len(contentValue) > 0 {
						browser.Open(contentValue[1])
						t.Logf("open new url after creating user in rhsso url: %s", browser.Url().String())
					}
				}
			})
		})

		t.Logf("request status code %v , browser response", browser.StatusCode())

		return isUserAuthenticated(browser, tsDashboardURL), nil
	})

	if err != nil {
		errLogin := browser.Open(tsDashboardURL)
		if errLogin != nil {
			t.Logf("failed to open dashboard url: %w", err)
		}
		if !isUserAuthenticated(browser, tsDashboardURL) {
			return fmt.Errorf("Account was not provisioned: %w", err)
		}
	}

	return nil
}

// checks if user is authenticated according to the browser url
func isUserAuthenticated(browser *browser.Browser, tsDashboardURL string) bool {

	// Check if the request returns 302
	// which means that the user is already authenticated
	if browser.StatusCode() == 302 {
		// opens landing page
		err := browser.Open(tsDashboardURL)
		if err != nil {
			return false
		}
	}

	// checks if user is redirected to the lading page
	return strings.Contains(browser.Url().RequestURI(), threeScaleDashboardURI) ||
		strings.Contains(browser.Url().RequestURI(), threeScaleOnboardingURI)

}

func authenticateThroughRHSSO(browser *browser.Browser) error {
	// click on authenticate throught rhsso
	err := browser.Click("a.authorizeLink")
	if err != nil {
		return fmt.Errorf("failed to click on a.authorizeLink to authenticate throught rhsso: %w", err)
	}

	return nil
}

func selectRHSSOIDP(browser *browser.Browser, idp string) error {
	browser.Find("noscript").Each(func(i int, selection *goquery.Selection) {
		selection.SetHtml(selection.Text())
	})
	if err := browser.Click(fmt.Sprintf("a:contains('%s')", idp)); err != nil {
		return fmt.Errorf("failed to click testing-idp identity provider in oauth proxy login, ensure the identity provider exists on the cluster: %w", err)
	}

	return nil
}
