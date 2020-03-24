package resources

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const (
	OpenshiftTokenName = "openshift-session-token"
	DefaultIDP         = "testing-idp"

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
func DoAuthOpenshiftUser(oauthUrl string, masterURL string, idp string, username string, password string) (string, error) {
	// declare transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// declare new cookie jar om nom nom
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return "", fmt.Errorf("error occurred creating a new cookie jar: %w", err)
	}

	// declare http client
	client := &http.Client{
		Transport: tr,
		Jar:       jar,
	}

	// get state
	state, err := getOpenshiftState(client, fmt.Sprintf("https://%s/auth/login", masterURL))
	if err != nil {
		return "", fmt.Errorf("error occured while getting state, %w", err)
	}

	// get auth url
	authURL, err := getOpenshiftAuthUrl(client, oauthUrl, masterURL, idp, state)
	if err != nil {
		return "", fmt.Errorf("error occured while getting auth url, %w", err)
	}

	user := User{
		Username: username,
		Password: password,
	}

	// get openshift token
	openshiftToken, err := getOpenshiftToken(client, user, authURL)
	if err != nil {
		return "", fmt.Errorf("error occured while trying to get openshift token, %w", err)
	}

	return openshiftToken, nil
}

// auth user returning openshift token
func getOpenshiftToken(client *http.Client, user User, authUrl string) (string, error) {
	// create query string
	u, err := query.Values(user)
	if err != nil {
		return "", fmt.Errorf("error occured while parsing values, %w", err)
	}

	// create post to openshift auth
	req, err := http.NewRequest("POST", authUrl, strings.NewReader(u.Encode()))
	if err != nil {
		return "", fmt.Errorf("error occured during new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// handle request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error occured during request: %w", err)
	}
	defer resp.Body.Close()

	// check response for token
	for _, c := range resp.Cookies() {
		if c.Name == OpenshiftTokenName {
			return c.Value, nil
		}
	}
	return "", fmt.Errorf("no openshift session token found")
}

// state is needed throughout the auth process, this call returns state
func getOpenshiftState(client *http.Client, stateUrl string) (string, error) {
	// create new request
	req, err := http.NewRequest("GET", stateUrl, nil)
	if err != nil {
		return "", fmt.Errorf("error occured creating request: %w", err)
	}

	// handle request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error occured at client do: %w", err)
	}

	// parse url query
	parsedQuery, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return "", fmt.Errorf("error occured parsing query: %w", err)
	}

	// return state query
	if parsedQuery == nil {
		return "", errors.New(fmt.Sprintf("parsed query is nil from response host : %s", resp.Request.URL.Host))
	}
	state := parsedQuery["state"][0]
	if state == "" {
		return "", errors.New(fmt.Sprintf("expected to find 'state' value in : %s", resp.Request.URL.RawQuery))
	}
	return state, nil
}

// a session is needed throughout the auth process, this call parses the openshift idp login page to return the correct url with the session query string
func getOpenshiftAuthUrl(client *http.Client, oauthUrl string, masterUrl string, idp string, state string) (string, error) {
	// create login query options
	loginOpt := LoginOptions{"console", idp}
	lv, _ := query.Values(loginOpt)

	// create callback options
	callbackOpt := CallbackOptions{"code", "user:full", state}
	cv, _ := query.Values(callbackOpt)

	// create callback url
	callbackUrl := "https%3A%2F%2F" + masterUrl + "%2Fauth%2Fcallback&" + cv.Encode()
	// create request url, not this url needed to be created in this format to avoid any unwanted escape character
	reqUrl := "https://" + oauthUrl + "/oauth/authorize?" + lv.Encode() + "&redirect_uri=" + callbackUrl
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return "", fmt.Errorf("error occured while creating a new request, %w", err)
	}

	// perform request, handling redirects
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error occured at client do: %w", err)
	}

	// parse response body
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error occured parsing http response: %w", err)
	}

	// find expected form to return auth url
	var f func(*html.Node)
	var authURL string
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			for _, a := range n.Attr {
				if a.Key == "action" {
					authURL = a.Val
					continue
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	if authURL == "" {
		return "", errors.New("auth url not found in response, token can not be retrieved")
	}

	return authURL, nil
}
