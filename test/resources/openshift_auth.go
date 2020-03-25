package resources

import (
	"crypto/tls"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf/errors"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/headzoo/surf.v1"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

const (
	OpenshiftAuthenticationNamespace = "openshift-authentication"
	OpenshiftOAuthRouteName          = "oauth-openshift"

	PathProjects = "/api/kubernetes/apis/project.openshift.io/v1/projects"
	PathFusePods = "/api/kubernetes/api/v1/namespaces/redhat-rhmi-fuse/pods"
)

// User used to create url user query
type User struct {
	Username string `url:"username"`
	Password string `url:"password"`
}

// LoginOptions used to create and parse url login options
type LoginOptions struct {
	Client string `url:"client_id"`
	IDP    string `url:"idp"`
}

// CallbackOptions used to create and parse url callback options
type CallbackOptions struct {
	Response string `url:"response_type"`
	Scope    string `url:"scope"`
	State    string `url:"state"`
}

// doAuthOpenshiftUser this function expects users and IDP to be created via `./scripts/setup-sso-idp.sh`
func DoAuthOpenshiftUser(masterURL string, username string, password string) (*http.Client, error) {
	// declare transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// declare new cookie jar om nom nom
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("error occurred creating a new cookie jar: %w", err)
	}

	// declare http client
	client := &http.Client{
		Transport: tr,
		Jar:       jar,
	}

	if err = openshiftClientSetup(fmt.Sprintf("https://%s/auth/login", masterURL), username, password, client); err != nil {
		return nil, fmt.Errorf("error occurred during oauth login: %w", err)
	}
	return client, nil
}

func OpenshiftIDPCheck(url string, client *http.Client) (bool, error) {
	browser := surf.NewBrowser()
	browser.SetTransport(client.Transport)
	if err := browser.Open(url); err != nil {
		return false, fmt.Errorf("failed to open browser url: %w", err)
	}
	browser.Find("noscript").Each(func(i int, selection *goquery.Selection) {
		selection.SetHtml(selection.Text())
	})
	if err := browser.Click("a:contains('testing-idp')"); err != nil {
		if _, ok := err.(errors.ElementNotFound); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to get idp anchor tag element: %w", err)
	}
	return true, nil
}

//openshiftOAuthProxyLogin Retrieve a cookie by logging in through the OpenShift OAuth Proxy
func openshiftClientSetup(url, username, password string, client *http.Client) error {
	//oauth proxy-specific constants
	const (
		openshiftOauthSubdomain = "oauth-openshift."
	)
	//follow the oauth proxy flow
	browser := surf.NewBrowser()
	browser.SetCookieJar(client.Jar)
	browser.SetTransport(client.Transport)
	if err := browser.Open(url); err != nil {
		return fmt.Errorf("failed to open browser url: %w", err)
	}
	browser.Find("noscript").Each(func(i int, selection *goquery.Selection) {
		selection.SetHtml(selection.Text())
	})
	if err := browser.Click("a:contains('testing-idp')"); err != nil {
		return fmt.Errorf("failed to click testing-idp identity provider in oauth proxy login, ensure the identity provider exists on the cluster: %w", err)
	}
	loginForm, err := browser.Form("#kc-form-login")
	if err != nil {
		return fmt.Errorf("failed to get login form from oauth proxy screen: %w", err)
	}
	if err = loginForm.Input("username", username); err != nil {
		return fmt.Errorf("failed to set username on oauth proxy form: %w", err)
	}
	if err = loginForm.Input("password", password); err != nil {
		return fmt.Errorf("failed to set password on oauth proxy form: %w", err)
	}
	if err = loginForm.Submit(); err != nil {
		return fmt.Errorf("failed to submit login form on oauth proxy screen: %w", err)
	}
	//sometimes we'll reach an accept permissions page for the user if they haven't accepted these scope requests before.
	if strings.Contains(browser.Url().Host, openshiftOauthSubdomain) {
		permissionsForm, err := browser.Form("[action=approve]")
		if err != nil {
			return fmt.Errorf("failed to get permissions form: %w", err)
		}
		if err = permissionsForm.Submit(); err != nil {
			return fmt.Errorf("failed to submit acceptance button for permissions: %w", err)
		}
	}
	return nil
}
