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
		chromedp.Sleep(5 * time.Second), // allow redirects after login to complete
		chromedp.WaitReady(`body`),      // ensure document is stable before querying (avoids "No node with given id found")
		chromedp.ActionFunc(func(ctx context.Context) error {
			var html string
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

// assertConsoleLinksAction checks that the console links from the Application Launcher dropdown match expected RHOAM links.
// It tries multiple section selectors (OCP/PatternFly versions differ) and collects RHOAM links from all sections so it works for both developer and dedicated-admin views.
func assertConsoleLinksAction(t TestingTB, expectedConsoleLinks []ConsoleLinkAssertion, clusterVersion string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		sectionSelectors := sectionSelectorsForVersion(clusterVersion)

		var allSections []*cdp.Node
		var usedSelector string
		for _, sel := range sectionSelectors {
			var sections []*cdp.Node
			if err := chromedp.Nodes(sel, &sections, chromedp.ByQueryAll).Do(ctx); err != nil {
				continue
			}
			if len(sections) > 0 {
				allSections = sections
				usedSelector = sel
				break
			}
		}
		if len(allSections) == 0 {
			t.Fatal("unable to find sections containing console links (tried multiple selectors)")
		}

		if isBroadAppLauncherSectionSelector(usedSelector) {
			t.Logf("WARNING: console-links check used a broad section selector %q (narrower selectors matched nothing). "+
				"Links are still validated by exact URL; consider updating selectors if OCP/console DOM changed. clusterVersion=%q",
				usedSelector, clusterVersion)
		}

		// Collect all links with href containing rhoam or 3scale from ALL sections (RHOAM links may be in any section in 4.21)
		var filteredLinks []*cdp.Node
		for _, section := range allSections {
			var links []*cdp.Node
			if err := chromedp.Nodes(`a`, &links, chromedp.FromNode(section), chromedp.ByQueryAll).Do(ctx); err != nil {
				continue
			}
			for _, link := range links {
				href := link.AttributeValue("href")
				if strings.Contains(href, "rhoam") || strings.Contains(href, "3scale") {
					filteredLinks = append(filteredLinks, link)
				}
			}
		}

		if len(filteredLinks) < len(expectedConsoleLinks) {
			return fmt.Errorf("expected at least %d console links (3scale, Grafana, User SSO) but got %d (selector used: %s)", len(expectedConsoleLinks), len(filteredLinks), usedSelector)
		}

		// Match expected URLs (order may differ in DOM)
		matched := make([]bool, len(expectedConsoleLinks))
		for _, link := range filteredLinks {
			href := link.AttributeValue("href")
			for i, expected := range expectedConsoleLinks {
				if !matched[i] && href == expected.URL {
					matched[i] = true
					break
				}
			}
		}
		for i, m := range matched {
			if !m {
				return fmt.Errorf("expected console link %q not found in launcher", expectedConsoleLinks[i].URL)
			}
		}

		// Icons: collect from all sections
		var allIcons []*cdp.Node
		for _, section := range allSections {
			var icons []*cdp.Node
			if err := chromedp.Nodes(`img`, &icons, chromedp.FromNode(section), chromedp.ByQueryAll).Do(ctx); err != nil {
				continue
			}
			allIcons = append(allIcons, icons...)
		}
		if len(allIcons) < len(expectedConsoleLinks) {
			return fmt.Errorf("expected at least %d console icons but got %d", len(expectedConsoleLinks), len(allIcons))
		}
		totalRhoamIcons := 0
		for _, icon := range allIcons {
			src := icon.AttributeValue("src")
			for _, expected := range expectedConsoleLinks {
				if src == expected.Icon {
					totalRhoamIcons++
					break
				}
			}
		}
		if totalRhoamIcons < len(expectedConsoleLinks) {
			return fmt.Errorf("expected %d RHOAM console icons but got %d", len(expectedConsoleLinks), totalRhoamIcons)
		}

		return nil
	}
}

// isBroadAppLauncherSectionSelector reports selectors that may match non-launcher sections
// (e.g. any element with *menu__group or *__group in class). Prefer updating sectionSelectorsForVersion
// if tests log this warning on a supported OCP version.
func isBroadAppLauncherSectionSelector(sel string) bool {
	switch sel {
	case `section[class*="menu__group"]`, `section[class*="__group"]`:
		return true
	default:
		return false
	}
}

// sectionSelectorsForVersion returns CSS selectors for app launcher menu sections, in order of preference.
func sectionSelectorsForVersion(clusterVersion string) []string {
	match, _ := regexp.MatchString("4\\.1[567]\\.", clusterVersion)
	match1, _ := regexp.MatchString("4\\.18\\.", clusterVersion)
	match2, _ := regexp.MatchString("4\\.19\\.|4\\.20\\.|4\\.21\\.", clusterVersion)

	switch {
	case match:
		return []string{`section[class="pf-v5-c-app-launcher__group"]`, `section[class*="app-launcher__group"]`}
	case match1:
		return []string{`section[class="pf-v5-c-menu__group"]`, `section[class*="menu__group"]`}
	case match2:
		return []string{
			`section[class="pf-v6-c-menu__group"]`,
			`section[class*="pf-v6-c-menu__group"]`,
			`section[class*="menu__group"]`,
			`section[class*="__group"]`,
		}
	default:
		return []string{
			`section[class="pf-c-app-launcher__group"]`,
			`section[class*="app-launcher__group"]`,
			`section[class*="menu__group"]`,
			`section[class*="__group"]`,
		}
	}
}
