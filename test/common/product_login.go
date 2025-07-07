package common

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestProductLogins Test logins to RHOAM products as dedicated admin and developer user
// Test login for 3scale is covered by Test3ScaleUserPromotion
func TestProductLogins(t TestingTB, ctx *TestingContext) {
	// To run this testcase for multiple test-users, set USER_NUMBERS to a string
	// in a "1,2,3" format
	userNumbers := os.Getenv("USER_NUMBERS")
	testUserNumbers := strings.Split(userNumbers, ",")

	if userNumbers == "" {
		t.Logf("env var USER_NUMBERS was not set, defaulting to 1")
		testUserNumbers = []string{"1"}
	}
	var (
		developerUsers     []string
		dedicatedAdminUser = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
	)
	for _, numberString := range testUserNumbers {
		number, err := strconv.Atoi(numberString)
		if err != nil {
			t.Fatalf("`USER_NUMBERS` env variable doesn't have the proper format (e.g. '1,2,3') %v", err)
		}
		developerUsers = append(developerUsers, fmt.Sprintf("%v%02d", DefaultTestUserName, number))
	}

	const keyCloakAuthConsolePath = "/auth/admin"

	if err := createTestingIDP(t, context.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	grafanaHost := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductGrafana].Host
	userSSOConsoleUrl := fmt.Sprintf("%s%s", rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSOUser].Host, keyCloakAuthConsolePath)
	rhssoConsoleUrl := fmt.Sprintf("%s%s", rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSO].Host, keyCloakAuthConsolePath)
	for _, developerUser := range developerUsers {
		// Login to Grafana
		testLoginToCustomerGrafanaForUser(t, grafanaHost, developerUser, assertGrafanaLoginUnAuthorized(t, developerUser))

		// Login to User SSO
		testLoginToUserSSOForUser(t, userSSOConsoleUrl, developerUser)

		// Login to RHSSO
		testLoginToRHSSOForUser(t, rhssoConsoleUrl, developerUser)
	}

	//Login to the services as customer-admin
	testLoginToCustomerGrafanaForUser(t, grafanaHost, dedicatedAdminUser, assertGrafanaLoginAuthorized(t, dedicatedAdminUser))
	testLoginToUserSSOForUser(t, userSSOConsoleUrl, dedicatedAdminUser)
	testLoginToRHSSOForUser(t, rhssoConsoleUrl, dedicatedAdminUser)
}

func testLoginToCustomerGrafanaForUser(t TestingTB, grafanaConsoleUrl string, userName string, assertAction chromedp.Action) {
	ChromeDpTimeOutWithActions(t, 2*time.Minute, loginToGrafanaActions(t, grafanaConsoleUrl, userName, assertAction)...)
}

func loginToGrafanaActions(t TestingTB, grafanaConsoleUrl, userName string, assertAction chromedp.Action) []chromedp.Action {
	t.Logf("Trying to log into Customer Grafana: %s for user: %s", grafanaConsoleUrl, userName)

	actions := []chromedp.Action{
		chromedp.Navigate(grafanaConsoleUrl),
		chromedp.Click(`button`), // Click "login with openshift" button
	}

	actions = append(actions, chromeDPLoginIDPActions(userName)...)
	actions = append(actions, chromedp.Sleep(5*time.Second), // Wait for redirect
		chromedp.ActionFunc(func(ctx context.Context) error {
			var html string
			if err := chromedp.OuterHTML(`html`, &html).Do(ctx); err != nil {
				return err
			}

			if !strings.Contains(html, "Authorize Access") { // Permissions already accepted
				return nil
			}

			return chromedp.Click(`input[name='approve']`).Do(ctx) // Approve permissions
		}),
		assertAction)

	return actions
}

func assertGrafanaLoginUnAuthorized(t TestingTB, userName string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		html, err := chromeDPGetHtml(ctx)
		if err != nil {
			return err
		}
		if !strings.Contains(html, "Authorize") {
			t.Errorf("expected authorization prompt but did not find it in HTML: %s", html)
		}

		t.Logf("User %s was not authorized to log into customer grafana", userName)
		return nil
	})
}

func assertGrafanaLoginAuthorized(t TestingTB, userName string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Wait a bit to allow page contents to load
		var html string
		err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 1*time.Minute, true, func(ctx2 context.Context) (done bool, err error) {
			html, err = chromeDPGetHtml(ctx)
			if err != nil {
				return false, nil
			}

			if !strings.Contains(html, "Grafana") {
				t.Logf("User not at Grafana home yet")
				return false, nil
			}

			t.Logf("User %s successfully logged into customer grafana", userName)
			return true, nil
		})

		if err != nil {
			t.Logf("Expected login to Grafana to be successful but didnt: %s: err: %s", html, err)
		}

		return err
	})
}

func testLoginToUserSSOForUser(t TestingTB, userSSOConsoleUrl string, userName string) {
	ChromeDpTimeOutWithActions(t, 2*time.Minute, loginToUserSSOActions(t, userSSOConsoleUrl, userName)...)
}

func testLoginToRHSSOForUser(t TestingTB, rhssoConsoleUrl string, userName string) {
	ChromeDpTimeOutWithActions(t, 2*time.Minute, loginToRHSSOActions(t, rhssoConsoleUrl, userName)...)
}

func ChromeDpTimeOutWithActions(t TestingTB, timeOut time.Duration, actions ...chromedp.Action) {
	// Create chromedp context
	ctxC, cancel := chromedp.NewContext(context.TODO())

	defer cancel()
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctxC, timeOut)
	defer cancel()

	err := chromedp.Run(ctx, actions...)
	if err != nil {
		t.Errorf("ChromeDP test failed with error: %s", err)
	}
}

func loginToUserSSOActions(t TestingTB, userSSOConsoleUrl string, userName string) []chromedp.Action {
	t.Logf("Trying to log into User SSO: %s for user: %s", userSSOConsoleUrl, userName)

	actions := []chromedp.Action{
		chromedp.Navigate(userSSOConsoleUrl),
	}

	actions = append(actions, chromeDPLoginIDPActions(userName)...)
	actions = append(actions, chromedp.WaitVisible(`div[data-ng-controller="RealmTabCtrl"]`), // Wait to allow page to redirect to realm console
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}

			var successfulLoginUrl = fmt.Sprintf("%s/master/console/#/realms/master", userSSOConsoleUrl)
			if node.BaseURL != successfulLoginUrl {
				return fmt.Errorf("failed to sign to User SSO for user: %s", userName)
			}

			t.Logf("User %s successfully signed into User SSO console", userName)
			return nil
		}),
	)

	return actions
}

func loginToRHSSOActions(t TestingTB, rhssoConsoleUrl string, userName string) []chromedp.Action {
	t.Logf("Trying to log into RHSSO: %s for user: %s", rhssoConsoleUrl, userName)

	return []chromedp.Action{
		chromedp.Navigate(rhssoConsoleUrl),
		chromedp.WaitVisible(`#kc-page-title`), // Wait for redirect to kc login page
		chromedp.SendKeys(`//input[@name="username"]`, userName),
		chromedp.SendKeys(`//input[@name="password"]`, TestingIdpPassword),
		chromedp.Submit(`#kc-form-login`),
		chromedp.WaitVisible(`#input-error`), // Wait to expect login error
		chromedp.ActionFunc(func(ctx context.Context) error {
			html, err := chromeDPGetHtml(ctx)
			if err != nil {
				return err
			}

			if !strings.Contains(html, "Invalid username or password") {
				return fmt.Errorf("expected sign in error for RHSSO but didnt for user %s: %s", userName, html)
			}

			t.Logf("Test passed as user %s was unauthorized to RHSSO", userName)

			return nil
		}),
	}
}

func chromeDPGetHtml(ctx context.Context) (string, error) {
	var html string
	if err := chromedp.OuterHTML(`html`, &html).Do(ctx); err != nil {
		return "", fmt.Errorf("failed to get HTML: %s", err)
	}
	return html, nil
}

func chromeDPLoginIDPActions(userName string) []chromedp.Action {
	return []chromedp.Action{
		chromedp.WaitVisible(`html[data-test-id="login"]`), // Wait to allow page to redirect to oauth page
		chromedp.Click(`a[title="Log in with testing-idp"]`),
		chromedp.SendKeys(`//input[@name="username"]`, userName),
		chromedp.SendKeys(`//input[@name="password"]`, TestingIdpPassword),
		chromedp.Submit(`#kc-form-login`),
	}
}
