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

func createRealmInUserSSOActions(t TestingTB, userSSOConsoleUrl, userName string) []chromedp.Action {
	logger.Infof("Attempting to create realm in User SSO: %s for user: %s", userSSOConsoleUrl, userName)

	randomName := rand.HexadecimalString(6)

	return []chromedp.Action{
		chromedp.Navigate(userSSOConsoleUrl),
		chromedp.WaitVisible(`html[data-test-id="login"]`), // Wait to allow page to redirect to oauth page
		chromedp.Click(`a[title="Log in with testing-idp"]`),
		chromedp.SendKeys(`//input[@name="username"]`, userName),
		chromedp.SendKeys(`//input[@name="password"]`, TestingIdpPassword),
		chromedp.Submit(`#kc-form-login`),
		chromedp.WaitVisible(`div[data-ng-controller="RealmTabCtrl"]`),
		chromedp.Click(`#view > div.col-sm-3.col-md-2.col-sm-pull-9.col-md-pull-10.sidebar-pf.sidebar-pf-left > div.realm-selector > h2:nth-child(1) > i`),
		chromedp.WaitVisible(`#view > div.col-sm-3.col-md-2.col-sm-pull-9.col-md-pull-10.sidebar-pf.sidebar-pf-left > div.realm-selector > div > div > a`),
		chromedp.Click(`#view > div.col-sm-3.col-md-2.col-sm-pull-9.col-md-pull-10.sidebar-pf.sidebar-pf-left > div.realm-selector > div > div > a`),
		chromedp.WaitVisible(`#name`),
		chromedp.WaitEnabled(`#name`),
		chromedp.SendKeys(`#name`, randomName),
		chromedp.Sleep(5 * time.Second),
		chromedp.Click(`#view > div.col-sm-9.col-md-10.col-sm-push-3.col-md-push-2.ng-scope > form > div > div > button.ng-binding.btn.btn-primary`),
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
