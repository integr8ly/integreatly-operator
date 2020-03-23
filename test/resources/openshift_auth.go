package resources

import (
	goctx "context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf/errors"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"gopkg.in/headzoo/surf.v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OpenshiftAuthenticationNamespace = "openshift-authentication"
	OpenshiftOAuthRouteName          = "oauth-openshift"

	PathProjectRequests = "/apis/project.openshift.io/v1/projectrequests"
	PathProjects        = "/api/kubernetes/apis/apis/project.openshift.io/v1/projects"
	PathFusePods        = "/api/kubernetes/api/v1/namespaces/redhat-rhmi-fuse/pods"
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
func DoAuthOpenshiftUser(authPageURL string, username string, password string, httpClient *http.Client, idp string) error {
	parsedURL, err := url.Parse(authPageURL)
	if err != nil {
		return fmt.Errorf("failed to parse url %s: %w", authPageURL, err)
	}
	if parsedURL.Scheme == "" {
		authPageURL = fmt.Sprintf("https://%s", authPageURL)
	}
	if err := openshiftClientSetup(authPageURL, username, password, httpClient, idp); err != nil {
		return fmt.Errorf("error occurred during oauth login: %w", err)
	}
	return nil
}

func OpenshiftIDPCheck(url string, client *http.Client, idp string) (bool, error) {
	browser := surf.NewBrowser()
	browser.SetTransport(client.Transport)
	if err := browser.Open(url); err != nil {
		return false, fmt.Errorf("failed to open browser url: %w", err)
	}
	browser.Find("noscript").Each(func(i int, selection *goquery.Selection) {
		selection.SetHtml(selection.Text())
	})
	if err := browser.Click(fmt.Sprintf("a:contains('%s')", idp)); err != nil {
		if _, ok := err.(errors.ElementNotFound); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to get idp anchor tag element: %w", err)
	}
	return true, nil
}

/*
Checks if openshift user has been reconciled to the openshift realm in keycloak
*/
func OpenshiftUserReconcileCheck(openshiftClient *OpenshiftClient, k8sclient dynclient.Client, namespacePrefix, username string) error {
	userSyncRetryInterval := time.Second * 30
	userSyncTimeout := time.Minute * 5

	return wait.Poll(userSyncRetryInterval, userSyncTimeout, func() (done bool, err error) {

		fuseNamespace := fmt.Sprintf("%v%v", namespacePrefix, integreatlyv1alpha1.ProductFuse)
		// ensure the fuse project can be seen by the user
		if _, err := openshiftClient.GetProject(fuseNamespace); err != nil {
			// fuse project not available to the user yet
			if strings.Contains(err.Error(), "forbidden") {
				return false, nil
			}
			return false, fmt.Errorf("failed to get fuse project: %v", err)
		}

		// ensure that a generated keycloak user cr has been created for the user
		generatedKeycloakUsers := &keycloak.KeycloakUserList{}
		labelSelector, err := labels.Parse("sso=integreatly")
		if err != nil {
			return false, fmt.Errorf("failed to parse label selector: %v", err)
		}
		rhssoNamespace := fmt.Sprintf("%s%s", namespacePrefix, "rhsso")
		if err := k8sclient.List(goctx.TODO(), generatedKeycloakUsers, &dynclient.ListOptions{LabelSelector: labelSelector, Namespace: rhssoNamespace}); err != nil {
			return false, fmt.Errorf("failed to list keycloak users: %v", err)
		}

		for _, user := range generatedKeycloakUsers.Items {
			if user.Spec.User.UserName == username {
				return true, nil
			}
		}
		return false, nil
	})
}

//openshiftOAuthProxyLogin Retrieve a cookie by logging in through the OpenShift OAuth Proxy
func openshiftClientSetup(url, username, password string, client *http.Client, idp string) error {
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
	if err := browser.Click(fmt.Sprintf("a:contains('%s')", idp)); err != nil {
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
