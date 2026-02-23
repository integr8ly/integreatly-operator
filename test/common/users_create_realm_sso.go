package common

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	rand "github.com/3scale/3scale-operator/pkg/crypto/rand"
	"github.com/chromedp/chromedp"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	logger "github.com/sirupsen/logrus"
)

func TestUsersCreateRealmSSO(t TestingTB, ctx *TestingContext) {
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

	userSSOConsoleUrl := fmt.Sprintf("%s%s", rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSOUser].Host, keyCloakAuthConsolePath)

	createRealmInUserSSO(t, userSSOConsoleUrl, dedicatedAdminUser)

	for _, developerUser := range developerUsers {
		createRealmInUserSSO(t, userSSOConsoleUrl, developerUser)
	}

}

func createRealmInUserSSO(t TestingTB, userSSOConsoleUrl, userName string) {
	ChromeDpTimeOutWithActions(t, 10*time.Minute, createRealmInUserSSOActions(t, userSSOConsoleUrl, userName)...)
}

// waitForKeycloakAdminUI waits for the Keycloak admin UI to be visible (old Angular or new React UI).
func waitForKeycloakAdminUI(t TestingTB) chromedp.ActionFunc {
	selectors := []string{
		`div[data-ng-controller="RealmTabCtrl"]`, // old Angular admin
		`#realm-selector`,                        // common id
		`[data-testid="realmSelector"]`,          // new Keycloak admin
		`.pf-c-page`,                             // PatternFly 4 page (new admin)
		`#view`,                                  // old admin container
	}
	return func(ctx context.Context) error {
		for _, sel := range selectors {
			subCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
			err := chromedp.WaitVisible(sel).Do(subCtx)
			cancel()
			if err == nil {
				t.Logf("Keycloak admin UI ready (selector: %s)", sel)
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		return fmt.Errorf("Keycloak admin UI did not appear (tried %d selectors)", len(selectors))
	}
}

// clickRealmSelectorDropdown opens the realm dropdown in the sidebar (tries several selectors).
func clickRealmSelectorDropdown(t TestingTB) chromedp.ActionFunc {
	selectors := []string{
		`div.realm-selector > h2 > i`,
		`div.realm-selector h2 i`,
		`.realm-selector i`,
		`div.realm-selector h2`, // click h2 if no i
		`div.realm-selector`,    // click whole realm-selector div
		`.sidebar-pf .realm-selector h2`,
		`.sidebar-pf-left .realm-selector`,
		`[class*="realm-selector"] h2`,
		`[class*="realm-selector"]`,
		`#view > div.col-sm-3.col-md-2.col-sm-pull-9.col-md-pull-10.sidebar-pf.sidebar-pf-left > div.realm-selector > h2:nth-child(1) > i`,
	}
	return func(ctx context.Context) error {
		// Try normal click first
		if err := tryClickSelectors(t, "realm selector dropdown", selectors)(ctx); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Fallback: JavaScript click in case element is covered or chromedp click fails
		for _, sel := range selectors {
			subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			script := fmt.Sprintf("(function(){ var e = document.querySelector(%q); if (e) { e.click(); return true; } return false; })()", sel)
			var clicked bool
			err := chromedp.Evaluate(script, &clicked).Do(subCtx)
			cancel()
			if err == nil && clicked {
				t.Logf("Clicked realm selector via JS (selector: %s)", sel)
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		return fmt.Errorf("could not click realm selector dropdown (tried %d selectors + JS fallback)", len(selectors))
	}
}

// waitAndClickCreateRealmLink waits for and clicks the "Create realm" link in the dropdown.
func waitAndClickCreateRealmLink(t TestingTB) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		selectors := []string{
			`#view > div.col-sm-3.col-md-2.col-sm-pull-9.col-md-pull-10.sidebar-pf.sidebar-pf-left > div.realm-selector > div > div > a`,
			`div.realm-selector div a`,
			`.realm-selector a`,
		}
		for _, sel := range selectors {
			subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := chromedp.WaitVisible(sel).Do(subCtx)
			cancel()
			if err != nil {
				continue
			}
			if err = chromedp.Click(sel).Do(ctx); err == nil {
				t.Logf("Clicked create realm link (selector: %s)", sel)
				return nil
			}
		}
		return fmt.Errorf("could not find or click create realm link")
	}
}

// clickCreateRealmSubmitButton submits the create-realm form.
func clickCreateRealmSubmitButton(t TestingTB) chromedp.ActionFunc {
	selectors := []string{
		`#view > div.col-sm-9.col-md-10.col-sm-push-3.col-md-push-2.ng-scope > form > div > div > button.ng-binding.btn.btn-primary`,
		`form button.btn-primary`,
		`button.ng-binding.btn.btn-primary`,
		`button.btn-primary`,
	}
	return tryClickSelectors(t, "create realm submit", selectors)
}

func tryClickSelectors(t TestingTB, name string, selectors []string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		for _, sel := range selectors {
			subCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			err := chromedp.Click(sel).Do(subCtx)
			cancel()
			if err == nil {
				t.Logf("Clicked %s (selector: %s)", name, sel)
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		return fmt.Errorf("could not click %s (tried %d selectors)", name, len(selectors))
	}
}

func createRealmInUserSSOActions(t TestingTB, userSSOConsoleUrl, userName string) []chromedp.Action {
	logger.Infof("Attempting to create realm in User SSO: %s for user: %s", userSSOConsoleUrl, userName)

	// Random name so the same user can run the test repeatedly (each run creates a new realm)
	realmName := rand.HexadecimalString(6)

	return []chromedp.Action{
		chromedp.Navigate(userSSOConsoleUrl),
		chromedp.WaitVisible(`html[data-test-id="login"]`), // Wait to allow page to redirect to oauth page
		chromedp.Click(`a[title="Log in with testing-idp"]`),
		chromedp.SendKeys(`//input[@name="username"]`, userName),
		chromedp.SendKeys(`//input[@name="password"]`, TestingIdpPassword),
		chromedp.Submit(`#kc-form-login`),
		chromedp.Sleep(5 * time.Second), // allow redirect after login
		chromedp.WaitReady(`body`),      // ensure document is stable before waiting for admin UI
		waitForKeycloakAdminUI(t),       // old Angular (RealmTabCtrl) or new React admin
		chromedp.Sleep(2 * time.Second), // let realm selector render
		clickRealmSelectorDropdown(t),   // open realm dropdown (try several selectors)
		waitAndClickCreateRealmLink(t),
		chromedp.WaitVisible(`#name`),
		chromedp.WaitEnabled(`#name`),
		chromedp.SendKeys(`#name`, realmName),
		chromedp.Sleep(5 * time.Second),
		clickCreateRealmSubmitButton(t),
		chromedp.Sleep(5 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			html, err := chromeDPGetHtml(ctx)
			if err != nil {
				return err
			}
			logger.Infof("Test passed as user: %s was able to create a Realm. HTML: %s", userName, html)
			return nil
		}),
	}
}
