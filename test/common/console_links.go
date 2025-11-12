package common

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/cluster"
	"github.com/integr8ly/integreatly-operator/utils"
)

type ConsoleLinkAssertion struct {
	URL  string
	Icon string
}

// TestConsoleLinks tests the console links are the same as in the RHMI CR status
// Logins is covered by TestProductLogins
func TestConsoleLinks(t TestingTB, ctx *TestingContext) {
	if err := createTestingIDP(t, context.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	consoleRoute, err := utils.GetConsoleRouteCR(context.TODO(), ctx.Client)
	if err != nil {
		t.Fatal(err)
	}

	clusterVersionCR, err := cluster.GetClusterVersionCR(context.TODO(), ctx.Client)
	if err != nil {
		t.Fatalf("error getting ClusterVersion CR: %w", err)
	}
	clusterVersion, err := cluster.GetClusterVersion(clusterVersionCR)
	if err != nil {
		t.Fatalf("error getting cluster version from ClusterVersion CR: %w", err)
	}

	const expectedIcon = "data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI1LjIuMCwgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAzNyAzNyIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMzcgMzc7IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KCS5zdDB7ZmlsbDojRUUwMDAwO30KCS5zdDF7ZmlsbDojRkZGRkZGO30KPC9zdHlsZT4KPGc+Cgk8cGF0aCBkPSJNMjcuNSwwLjVoLTE4Yy00Ljk3LDAtOSw0LjAzLTksOXYxOGMwLDQuOTcsNC4wMyw5LDksOWgxOGM0Ljk3LDAsOS00LjAzLDktOXYtMThDMzYuNSw0LjUzLDMyLjQ3LDAuNSwyNy41LDAuNUwyNy41LDAuNXoiCgkJLz4KCTxnPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yNSwyMi4zN2MtMC45NSwwLTEuNzUsMC42My0yLjAyLDEuNWgtMS44NVYyMS41YzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYycy0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDIuNDhjMC4yNywwLjg3LDEuMDcsMS41LDIuMDIsMS41YzEuMTcsMCwyLjEyLTAuOTUsMi4xMi0yLjEyUzI2LjE3LDIyLjM3LDI1LDIyLjM3eiBNMjUsMjUuMzcKCQkJYy0wLjQ4LDAtMC44OC0wLjM5LTAuODgtMC44OHMwLjM5LTAuODgsMC44OC0wLjg4czAuODgsMC4zOSwwLjg4LDAuODhTMjUuNDgsMjUuMzcsMjUsMjUuMzd6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTIwLjUsMTYuMTJjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTIuMzhoMS45MWMwLjMyLDAuNzcsMS4wOCwxLjMxLDEuOTYsMS4zMQoJCQljMS4xNywwLDIuMTItMC45NSwyLjEyLTIuMTJzLTAuOTUtMi4xMi0yLjEyLTIuMTJjLTEuMDIsMC0xLjg4LDAuNzMtMi4wOCwxLjY5SDIwLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJQzE5Ljg3LDE1Ljg1LDIwLjE2LDE2LjEyLDIwLjUsMTYuMTJ6IE0yNSwxMS40M2MwLjQ4LDAsMC44OCwwLjM5LDAuODgsMC44OHMtMC4zOSwwLjg4LTAuODgsMC44OHMtMC44OC0wLjM5LTAuODgtMC44OAoJCQlTMjQuNTIsMTEuNDMsMjUsMTEuNDN6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTEyLjEyLDE5Ljk2di0wLjg0aDIuMzhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJzLTAuMjgtMC42Mi0wLjYyLTAuNjJoLTIuMzh2LTAuOTEKCQkJYzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYyaC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYzYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDNDMTEuODQsMjAuNTksMTIuMTIsMjAuMzEsMTIuMTIsMTkuOTYKCQkJeiBNMTAuODcsMTkuMzRIOS4xMnYtMS43NWgxLjc1VjE5LjM0eiIvPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yOC41LDE2LjM0aC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYwLjkxSDIyLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYyczAuMjgsMC42MiwwLjYyLDAuNjJoMi4zOAoJCQl2MC44NGMwLDAuMzUsMC4yOCwwLjYyLDAuNjIsMC42MmgzYzAuMzQsMCwwLjYyLTAuMjgsMC42Mi0wLjYydi0zQzI5LjEyLDE2LjYyLDI4Ljg0LDE2LjM0LDI4LjUsMTYuMzR6IE0yNy44NywxOS4zNGgtMS43NXYtMS43NQoJCQloMS43NVYxOS4zNHoiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwyMC44N2MtMC4zNCwwLTAuNjMsMC4yOC0wLjYzLDAuNjJ2Mi4zOGgtMS44NWMtMC4yNy0wLjg3LTEuMDctMS41LTIuMDItMS41CgkJCWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMmMwLjk1LDAsMS43NS0wLjYzLDIuMDItMS41aDIuNDhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTMKCQkJQzE3LjEyLDIxLjE1LDE2Ljg0LDIwLjg3LDE2LjUsMjAuODd6IE0xMiwyNS4zN2MtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4CgkJCVMxMi40OCwyNS4zNywxMiwyNS4zN3oiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwxMS44N2gtMi40MmMtMC4yLTAuOTctMS4wNi0xLjY5LTIuMDgtMS42OWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMgoJCQljMC44OCwwLDEuNjQtMC41NCwxLjk2LTEuMzFoMS45MXYyLjM4YzAsMC4zNSwwLjI4LDAuNjIsMC42MywwLjYyczAuNjItMC4yOCwwLjYyLTAuNjJ2LTNDMTcuMTIsMTIuMTUsMTYuODQsMTEuODcsMTYuNSwxMS44N3oKCQkJIE0xMiwxMy4xOGMtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4UzEyLjQ4LDEzLjE4LDEyLDEzLjE4eiIvPgoJPC9nPgoJPHBhdGggY2xhc3M9InN0MSIgZD0iTTE4LjUsMjIuNjJjLTIuMjcsMC00LjEzLTEuODUtNC4xMy00LjEyczEuODUtNC4xMiw0LjEzLTQuMTJzNC4xMiwxLjg1LDQuMTIsNC4xMlMyMC43NywyMi42MiwxOC41LDIyLjYyegoJCSBNMTguNSwxNS42MmMtMS41OCwwLTIuODgsMS4yOS0yLjg4LDIuODhzMS4yOSwyLjg4LDIuODgsMi44OHMyLjg4LTEuMjksMi44OC0yLjg4UzIwLjA4LDE1LjYyLDE4LjUsMTUuNjJ6Ii8+CjwvZz4KPC9zdmc+Cg=="

	// Expected console links in the order expected from the Application Launcher
	expectedConsoleLinks := []ConsoleLinkAssertion{
		{
			URL:  fmt.Sprintf("%s/auth/rhsso/bounce", rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Host),
			Icon: expectedIcon,
		},
		{
			URL:  rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductGrafana].Host,
			Icon: expectedIcon,
		},
		{
			URL:  rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSOUser].Host,
			Icon: expectedIcon,
		},
	}

	consoleUrl := fmt.Sprintf("https://%s", consoleRoute.Spec.Host)

	// test console links for developer user
	testConsoleLinksForUser(t, consoleUrl, "test-user01", expectedConsoleLinks, clusterVersion)
	// test console links for dedicated admin
	testConsoleLinksForUser(t, consoleUrl, "customer-admin01", expectedConsoleLinks, clusterVersion)
}

func testConsoleLinksForUser(t TestingTB, consoleUrl, userName string, expectedConsoleLinks []ConsoleLinkAssertion, clusterVersion string) {
	var applicationLauncherSelector = `button[data-test-id="application-launcher"]`

	// Navigate to OSD landing page
	actions := []chromedp.Action{
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Navigate(consoleUrl).Do(ctx)
		}),
	}
	// Login via IDP
	actions = append(actions, chromeDPLoginIDPActions(userName)...)
	// Actions after logging in for user
	actions = append(actions, []chromedp.Action{
		chromedp.ActionFunc(func(ctx context.Context) error {
			var html string
			// This OuterHTML action implicitly waits for the page to load the HTML
			if err := chromedp.OuterHTML(`html`, &html).Do(ctx); err != nil {
				return err
			}

			// Tour was already skipped or irrelevant for user
			if !strings.Contains(html, "Welcome to the Developer Perspective!") {
				return nil
			}

			// Skip the tour
			if err := chromedp.Click(`button[id="tour-step-footer-secondary"]`).Do(ctx); err != nil {
				t.Logf("!! FAILED to click skip tour button: %v", err)
				return err
			}
			return nil
		}),

		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.WaitReady(applicationLauncherSelector).Do(ctx); err != nil {
				t.Logf("!! FAILED waiting for launcher to be ready: %v", err)
				return err
			}

			if err := chromedp.Sleep(500 * time.Millisecond).Do(ctx); err != nil {
				return err
			}

			// Use Evaluate to execute a direct JavaScript click() call
			clickScript := fmt.Sprintf(`document.querySelector('%s').click()`, applicationLauncherSelector)
			if err := chromedp.Evaluate(clickScript, nil).Do(ctx); err != nil {
				t.Logf("!! FAILED forcing JS click on launcher: %v", err)
				return err
			}

			return nil
		}),

		assertConsoleLinksAction(t, expectedConsoleLinks, clusterVersion),
	}...)

	ChromeDpTimeOutWithActions(t, 2*time.Minute, actions...)
}

// assertConsoleLinksAction checks that the console links from
func assertConsoleLinksAction(t TestingTB, expectedConsoleLinks []ConsoleLinkAssertion, clusterVersion string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// Get all the sections from the dropdown as there could be multiple depending on developer user and dedicated admin
		var sections []*cdp.Node

		sectionSelector := `section[class="pf-c-app-launcher__group"]`
		match, err := regexp.MatchString("4\\.1[567]\\.", clusterVersion)
		if err != nil {
			t.Fatal(err)
		}
		// Check if the clusterVersion matches 4.18
		match1, err := regexp.MatchString("4\\.18\\.", clusterVersion)
		if err != nil {
			t.Fatal(err)
		}
		// Check if the clusterVersion matches 4.19 or 4.20
		match2, err := regexp.MatchString("4\\.19\\.|4\\.20\\.", clusterVersion)
		if err != nil {
			t.Fatal(err)
		}

		if match {
			sectionSelector = `section[class="pf-v5-c-app-launcher__group"]`
		} else if match1 {
			// For versions 4.18:
			sectionSelector = `section[class="pf-v5-c-menu__group"]`
		} else if match2 {
			// For versions 4.19 or 4.20:
			sectionSelector = `section[class="pf-v6-c-menu__group"]`
		}

		if err := chromedp.Nodes(sectionSelector, &sections, chromedp.ByQueryAll).Do(ctx); err != nil {
			return err
		}

		if len(sections) == 0 {
			t.Fatal("unable to find sections containing console links")
		}

		// Use the last element in the slice as a dedicated-admin have 2 console link sections
		// Managed service links are in the last section (at the time of writing) for both developer and dedicated admin users
		lastElement := sections[len(sections)-1]

		// Get all the links from this section
		var links []*cdp.Node

		err = chromedp.Nodes(`a`, &links, chromedp.FromNode(lastElement), chromedp.ByQueryAll).Do(ctx)

		if err != nil {
			t.Fatal(err)
		}

		// Remove non-RHOAM links from the list
		var filteredLinks []*cdp.Node
		for _, link := range links {
			if strings.Contains(link.AttributeValue("href"), "rhoam") {
				filteredLinks = append(filteredLinks, link)
			} else if strings.Contains(link.AttributeValue("href"), "3scale") {
				filteredLinks = append(filteredLinks, link)
			}
		}
		links = filteredLinks

		// Assert number of links is as expected
		if len(links) != len(expectedConsoleLinks) {
			return fmt.Errorf("expected %d console links but got %d", len(expectedConsoleLinks), len(links))
		}

		// Assert the links itself are as expected
		for idx := range links {
			consoleLinkHref := links[idx].AttributeValue("href")
			expectedLinkHref := expectedConsoleLinks[idx].URL
			if consoleLinkHref != expectedLinkHref {
				return fmt.Errorf("expected %s as a console link url but got: %s", expectedLinkHref, consoleLinkHref)
			}
		}

		// get all the icons
		var icons []*cdp.Node
		err = chromedp.Nodes(`img`, &icons, chromedp.FromNode(lastElement), chromedp.ByQueryAll).Do(ctx)

		// Assert number of icons is as expected, can be more if other add-ons are installed
		if len(icons) < len(expectedConsoleLinks) {
			return fmt.Errorf("expected at least %d console icons but got %d", len(expectedConsoleLinks), len(icons))
		}

		// Assert the icons itself are as expected
		totalRhoamIcons := 0
		for idx := range links {
			consoleLinkIcon := icons[idx].AttributeValue("src")
			expectedLinkIcon := expectedConsoleLinks[idx].Icon
			if consoleLinkIcon == expectedLinkIcon {
				totalRhoamIcons += 1
			}
		}
		if totalRhoamIcons != len(expectedConsoleLinks) {
			return fmt.Errorf("expected %d RHOAM console icons but got %d", len(expectedConsoleLinks), totalRhoamIcons)
		}

		return nil
	}
}
