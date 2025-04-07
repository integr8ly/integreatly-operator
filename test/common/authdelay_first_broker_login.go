package common

import (
	"context"
	goctx "context"
	"crypto/rand"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	brow "github.com/headzoo/surf/browser"

	. "github.com/onsi/ginkgo/v2"
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

func TestAuthDelayFirstBrokerLogin(t TestingTB, ctx *TestingContext) {

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

	err = loginToThreeScale(t, tsHost, testUser.UserName, TestingIdpPassword, TestingIDPRealm, httpClient)
	if err != nil {
		t.Fatalf("[%s] error logging in to three scale: %v ", getTimeStampPrefix(), err)
	}
}

func getRandomKeycloakUser(ctx *TestingContext, installationName string) (*TestUser, error) {
	// create random keycloak user
	r1, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return nil, fmt.Errorf("error generating random username postfix")
	}
	userNamePostfix := r1.Int64()

	testUsers := []TestUser{
		{
			FirstName: TestAuthThreeScaleUsername,
			LastName:  fmt.Sprintf("User %d", userNamePostfix),
			UserName:  fmt.Sprintf("%s-%d", TestAuthThreeScaleUsername, userNamePostfix),
		},
	}
	err = createOrUpdateKeycloakUserCR(goctx.TODO(), ctx.Client, testUsers, installationName)
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
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*5, true, func(ctx2 context.Context) (done bool, err error) {
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

func loginToThreeScale(t TestingTB, tsHost, username, password string, idp string, client *http.Client) error {
	By("Login to 3Scale")
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

	clintBrowser := surf.NewBrowser()
	clintBrowser.SetCookieJar(client.Jar)
	clintBrowser.SetTransport(client.Transport)
	clintBrowser.SetAttribute(brow.FollowRedirects, true)

	// open threescale login page
	By("Open 3Scale login page")
	err = clintBrowser.Open(tsLoginURL)
	if err != nil {
		return fmt.Errorf("failed to open browser url: %w", err)
	}

	// check if user is already authenticated
	By("Check if user authenticated")
	if isUserAuthenticated(clintBrowser, tsDashboardURL) {
		return nil
	}

	t.Logf("User is not authenticated yet, going to authenticate through rhsso url  %s", clintBrowser.Url().String())

	// click on authenticate through rhsso
	By("Authenticate through RHSSO")
	err = authenticateThroughRHSSO(clintBrowser)
	if err != nil {
		t.Logf("response %s", clintBrowser.Body())
		return err
	}

	// check if user is already authenticated through rhsso
	By("Check if user authenticated")
	if isUserAuthenticated(clintBrowser, tsDashboardURL) {
		return nil
	}

	t.Logf("User is not authenticated through rhsso yet, going to select the IDP %s", clintBrowser.Url().String())

	By("Select the testing IDP to authenticate through RHSSO")
	selectIDPURL := clintBrowser.Url().String()
	// select the IDP to authenticate through RHSSO
	err = selectRHSSOIDP(clintBrowser, idp)
	if err != nil {
		return err
	}

	// check if user is already authenticated after selecting the IDP
	By("Check if user authenticated")
	if isUserAuthenticated(clintBrowser, tsDashboardURL) {
		return nil
	}

	t.Logf("User is not authenticated after selecting the IDP yet, going to submit the form to authenticate the user through rhsso %s", clintBrowser.Url().String())

	By("Submit login form for testing IDP")
	// submit openshift oauth login form
	err = clintBrowser.Open(selectIDPURL)
	if err != nil {
		return fmt.Errorf("failed to open selectIDPURL url: %w", err)
	}
	err = resources.OpenshiftClientSubmitForm(clintBrowser, username, password, idp, t)
	if err != nil {
		return fmt.Errorf("failed to submit the openshift oauth login: %w", err)
	}

	// check if user is authenticated after submitting rhsso login form
	By("Check if user authenticated after submitting RHSSO login form")
	if isUserAuthenticated(clintBrowser, tsDashboardURL) {
		return nil
	}

	By("Wait until account is provisioned and user is authenticated in 3Scale")
	// waits until the account is provisioned and user is authenticated in three scale
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*8, false, func(ctx context.Context) (done bool, err error) {
		t.Logf("\nbrowser URL first:\n %s\nURL with requestURI:\n %s\n", clintBrowser.Url().String(), clintBrowser.Url().RequestURI())

		// checks if an error happened in the login
		if clintBrowser.StatusCode() == 502 {
			t.Logf("Unexpected error: \nURL - %s \nRequest status code - %v", clintBrowser.Url().String(), clintBrowser.StatusCode())

			err := clintBrowser.Open(tsDashboardURL)
			t.Logf("Opened dashboard URL to validate user authentication: URL - %s | DashboardURL - %s", clintBrowser.Url().String(), tsDashboardURL)
			if err != nil {
				t.Logf("failed to open dashboard url: %w", err)
				return false, fmt.Errorf("failed to open dashboard url: %w", err)
			}

			// if user is redirected to the login page try again
			if clintBrowser.Url().String() == tsLoginURL {
				// click on authenticate through rhsso
				err = authenticateThroughRHSSO(clintBrowser)
				if err != nil {
					return false, err
				}

				// check if user is already authenticated through rhsso
				if isUserAuthenticated(clintBrowser, tsDashboardURL) {
					return true, nil
				}

				selectIDPURL = clintBrowser.Url().String()
				// select the IDP to authenticate through RHSSO
				err = selectRHSSOIDP(clintBrowser, idp)
				if err != nil {
					return false, err
				}

				// check if user is already authenticated after selecting the IDP
				if isUserAuthenticated(clintBrowser, tsDashboardURL) {
					return true, nil
				}
				t.Logf("authenticate after click on idp url: %s", clintBrowser.Url().String())
			}
		}

		if isUserAuthenticated(clintBrowser, tsDashboardURL) {
			return true, nil
		}

		clintBrowser.Find(fmt.Sprintf("h1:contains('%s')", provisioningAccountTxt)).Each(func(index int, s *goquery.Selection) {
			// gets the refresh url from the meta tag
			clintBrowser.Dom().Find("meta").Each(func(index int, s *goquery.Selection) {
				val, exist := s.Attr("content")
				if exist {
					contentValue := strings.Split(val, ";")
					if len(contentValue) > 0 {
						err = clintBrowser.Open(contentValue[1])
						if err != nil {
							t.Logf("open new URL error", err)
						}
						t.Logf("open new url after creating user in rhsso url: %s", clintBrowser.Url().String())
					}
				}
			})
		})

		if clintBrowser.StatusCode() != 200 {
			t.Logf("request status code %v , browser response", clintBrowser.StatusCode())
		}

		if isUserAuthenticated(clintBrowser, tsDashboardURL) {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		By("Try open 3Scale dashboard")
		errLogin := clintBrowser.Open(tsDashboardURL)
		if errLogin != nil {
			t.Logf("failed to open dashboard url: %w", err)
		}
		if !isUserAuthenticated(clintBrowser, tsDashboardURL) {
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
	rhssoUrl, err := retrieveRHSSOLink(browser)
	if err != nil {
		return fmt.Errorf("failed to authenticate through rhsso: %w", err)
	}

	err = browser.Open(rhssoUrl)
	if err != nil {
		return fmt.Errorf("failed to open RHSSO link: %w", err)
	}

	return nil
}

func selectRHSSOIDP(browser *browser.Browser, idp string) error {
	browser.Find("noscript").Each(func(i int, selection *goquery.Selection) {
		selection.SetHtml(selection.Text())
	})
	if err := browser.Click(fmt.Sprintf("a:contains('%s')", idp)); err != nil {
		return fmt.Errorf("selectRHSSOIDP(): failed to click testing-idp identity provider in oauth proxy login, ensure the identity provider exists on the cluster: %w", err)
	}

	return nil
}

func retrieveRHSSOLink(browser *browser.Browser) (string, error) {
	// Get body from HTML response
	htmlBody, err := browser.Find("body").Html()
	if err != nil {
		return "", err
	}

	// Split body on Unicode Decimal code for double quote, &#34;
	data := strings.Split(htmlBody, "&#34;")

	// Find RHSSO authorization link
	for _, d := range data {
		if strings.Contains(d, "rhsso") {
			// Decodes string
			decoded, err := strconv.Unquote(`"` + d + `"`)
			if err != nil {
				return "", fmt.Errorf("failed to decode string: %w", err)
			}
			return decoded, nil
		}
	}

	return "", fmt.Errorf("failed to retrieve RHSSO link")
}
