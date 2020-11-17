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
	t.Log("skipping first broker login test due to flakiness")
	t.SkipNow()

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

	err = loginToThreeScale(t, tsHost, testUser.UserName, DefaultPassword, TestingIDPRealm, ctx.HttpClient)
	if err != nil {
		dumpAuthResources(ctx.Client, t)
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

	// check if logged in already
	if browser.StatusCode() == 302 {
		// opens landing page
		browser.Open(tsDashboardURL)

		if isUserAuthenticated(browser.Url().RequestURI()) {
			return nil
		}
	}

	// click on authenticate through rhsso
	err = browser.Click("a.authorizeLink")
	if err != nil {
		return fmt.Errorf("failed to click authenticate throught rhsso: %w", err)
	}

	// submit openshift oauth login form
	err = resources.OpenshiftClientSubmitForm(browser, username, password, idp, t)
	if err != nil {
		return fmt.Errorf("failed to submit the openshift oauth login: %w", err)
	}

	// waits until the account is provisioned and user is authenticated in three scale
	err = wait.Poll(time.Second*5, time.Minute*5, func() (done bool, err error) {
		t.Logf("browser URL first %s%s", browser.Url(), browser.Url().RequestURI())

		browser.Find(fmt.Sprintf("h1:contains('%s')", provisioningAccountTxt)).Each(func(index int, s *goquery.Selection) {

			// gets the refresh url from the meta tag
			browser.Dom().Find("meta").Each(func(index int, s *goquery.Selection) {
				val, exist := s.Attr("content")
				if exist {
					contentValue := strings.Split(val, ";")
					if len(contentValue) > 0 {
						browser.Open(contentValue[1])
					}
				}
			})
		})

		// checks if an error happened in the login
		if browser.StatusCode() == 502 {
			browser.Open(tsDashboardURL)

			// if redirected to the login page try again
			if browser.Url().String() == tsLoginURL {
				// click on authenticate throught rhsso
				err = browser.Click("a.authorizeLink")
				if err != nil {
					return false, fmt.Errorf("failed to click authenticate throught rhsso: %w", err)
				}
			}
		}

		return isUserAuthenticated(browser.Url().RequestURI()), nil
	})

	if err != nil {
		return fmt.Errorf("Account was not provisioned: %w", err)
	}

	return nil
}

func isUserAuthenticated(tsURL string) bool {
	return strings.Contains(tsURL, threeScaleDashboardURI) ||
		strings.Contains(tsURL, threeScaleOnboardingURI)
}
