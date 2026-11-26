package common

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
		chromedp.WaitReady(`body`),
		chromedp.WaitVisible(`button`),
		chromedp.Click(`button`), // Click "login with openshift" button
	}

	actions = append(actions, chromeDPLoginIDPActions(userName)...)
	actions = append(actions, chromedp.Sleep(5*time.Second), // Wait for redirect
		chromedp.WaitReady(`body`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			html, err := chromeDPGetHtml(ctx)
			if err != nil {
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

// ChromeDpTimeOutWithActions runs chromedp actions with a timeout. Uses a desktop viewport
// (1920x1080) so that CI/headless and remote clusters get the same layout as local (e.g.
// OpenShift console application launcher visible in header instead of hidden on small viewports).
func ChromeDpTimeOutWithActions(t TestingTB, timeOut time.Duration, actions ...chromedp.Action) {
	// Chrome in OpenShift/K8s: arbitrary UID, small /dev/shm, no setuid sandbox.
	userDataDir, err := os.MkdirTemp("", "chromedp-*")
	if err != nil {
		t.Errorf("ChromeDP: could not create user data dir: %v", err)
		return
	}
	defer func() { _ = os.RemoveAll(userDataDir) }()

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1920, 1080),
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer allocCancel()
	ctxC, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel := context.WithTimeout(ctxC, timeOut)
	defer cancel()

	// When CHROMEDP_DEBUG=1, run each action with a log line so timeout shows which step failed
	if os.Getenv("CHROMEDP_DEBUG") == "1" {
		for i, a := range actions {
			t.Logf("[chromedp] step %d/%d", i+1, len(actions))
			if err := chromedp.Run(ctx, a); err != nil {
				t.Errorf("ChromeDP test failed at step %d/%d: %s", i+1, len(actions), err)
				return
			}
		}
		return
	}

	err = chromedp.Run(ctx, actions...)
	if err != nil {
		t.Errorf("ChromeDP test failed with error: %s (set CHROMEDP_DEBUG=1 to see which step fails)", err)
	}
}

func loginToUserSSOActions(t TestingTB, userSSOConsoleUrl string, userName string) []chromedp.Action {
	t.Logf("Trying to log into User SSO: %s for user: %s", userSSOConsoleUrl, userName)

	actions := []chromedp.Action{
		chromedp.Navigate(userSSOConsoleUrl),
	}

	actions = append(actions, chromeDPLoginIDPActions(userName)...)
	// Same settle + admin-UI detection as B08B: RHSSO/Keycloak may serve Angular or React admin.
	actions = append(actions,
		chromedp.Sleep(5*time.Second),
		chromedp.WaitReady(`body`),
		waitForKeycloakAdminUI(t),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var location string
			if err := chromedp.Location(&location).Do(ctx); err != nil {
				return err
			}
			// Accept either classic Angular admin URL (.../admin/master/console/#/realms/master)
			// or newer admin console paths that still land under /admin after IDP login.
			if !strings.Contains(location, "/admin") {
				return fmt.Errorf("failed to sign into User SSO for user %s: unexpected location %s", userName, location)
			}
			t.Logf("User %s successfully signed into User SSO console (%s)", userName, location)
			return nil
		}),
	)

	return actions
}

func loginToRHSSOActions(t TestingTB, rhssoConsoleUrl string, userName string) []chromedp.Action {
	t.Logf("Trying to log into RHSSO: %s for user: %s", rhssoConsoleUrl, userName)

	return []chromedp.Action{
		chromedp.Navigate(rhssoConsoleUrl),
		chromedp.WaitReady(`body`),
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
		chromeDPWaitForLoginOrTestingIDP(),
		chromeDPClickTestingIDP(),
		chromedp.SendKeys(`//input[@name="username"]`, userName),
		chromedp.SendKeys(`//input[@name="password"]`, TestingIdpPassword),
		chromedp.Submit(`#kc-form-login`),
	}
}

// chromeDPWaitForLoginOrTestingIDP waits for the OpenShift IDP picker or the
// Keycloak testing-idp form (auto-redirect when testing-idp is the only IDP).
func chromeDPWaitForLoginOrTestingIDP() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		deadline := time.Now().Add(time.Minute)
		for time.Now().Before(deadline) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if chromeDPAlreadyOnTestingIDP(ctx) {
				return nil
			}
			var onPicker bool
			_ = chromedp.Evaluate(`!!document.querySelector('html[data-test-id="login"]')`, &onPicker).Do(ctx)
			if onPicker {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
		return fmt.Errorf("timed out waiting for OpenShift login or Keycloak testing-idp form")
	}
}

func chromeDPAlreadyOnTestingIDP(ctx context.Context) bool {
	var location string
	_ = chromedp.Location(&location).Do(ctx)
	if strings.Contains(location, "/realms/testing-idp/") {
		return true
	}
	var onForm bool
	_ = chromedp.Evaluate(`!!document.querySelector('#kc-form-login')`, &onForm).Do(ctx)
	return onForm
}

// chromeDPClickTestingIDP clicks testing-idp on the OpenShift login page.
// No-op if OpenShift already redirected to the Keycloak testing-idp form.
func chromeDPClickTestingIDP() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if chromeDPAlreadyOnTestingIDP(ctx) {
				return nil
			}
			for _, sel := range []string{`a[href*="idp=testing-idp"]`, `a[title*="testing-idp"]`} {
				subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := chromedp.Click(sel).Do(subCtx)
				cancel()
				if err == nil {
					return nil
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		return fmt.Errorf("could not click testing-idp on OpenShift login page")
	}
}
