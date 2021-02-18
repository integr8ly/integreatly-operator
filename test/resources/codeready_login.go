package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	codereadySSOAuthEndpoint  = "%v/auth/realms/openshift/protocol/openid-connect/auth?client_id=%v&redirect_uri=%v&response_type=code&scope=openid"
	codereadySSOTokenEndpoint = "%v/auth/realms/openshift/protocol/openid-connect/token"
	clientId                  = "che-client"
)

type CodereadyLoginClient struct {
	HttpClient *http.Client
	Client     k8sclient.Client
	MasterUrl  string
	IdpName    string
	Username   string
	Password   string
	Logger     SimpleLogger
}

func NewCodereadyLoginClient(httpClient *http.Client, client k8sclient.Client, masterUrl, idp, username, password string, logger SimpleLogger) *CodereadyLoginClient {
	return &CodereadyLoginClient{
		HttpClient: httpClient,
		Client:     client,
		MasterUrl:  masterUrl,
		IdpName:    idp,
		Username:   username,
		Password:   password,
		Logger:     logger,
	}
}

// Login user to openshift and wait for the user to be reconciled to the openshift realm in keycloak
func (c *CodereadyLoginClient) OpenshiftLogin(namespacePrefix string) error {
	authUrl := fmt.Sprintf("%s/auth/login", c.MasterUrl)
	if err := DoAuthOpenshiftUser(authUrl, c.Username, c.Password, c.HttpClient, c.IdpName, c.Logger); err != nil {
		return err
	}

	openshiftClient := NewOpenshiftClient(c.HttpClient, c.MasterUrl)
	if err := OpenshiftUserReconcileCheck(openshiftClient, c.Client, namespacePrefix, c.Username); err != nil {
		return err
	}
	return nil
}

// Log user into codeready. Returns an access token to be used for codeready API calls
func (c *CodereadyLoginClient) CodereadyLogin(keycloakHost, redirectUrl string) (string, error) {
	u := fmt.Sprintf(codereadySSOAuthEndpoint, keycloakHost, clientId, redirectUrl)

	response, err := c.HttpClient.Get(u)
	if err != nil {
		return "", fmt.Errorf("failed to get %v: %v", u, err)
	}
	if response.StatusCode != http.StatusOK {
		return "", errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// Select the testing IDP
	document, err := parseResponse(response)
	if err != nil {
		return "", errorWithResponseDump(response, err)
	}

	// find the link to the testing IDP
	link, err := findElement(document, fmt.Sprintf("a:contains('%s')", c.IdpName))
	if err != nil {
		return "", errorWithResponseDump(response, err)
	}

	// get the url from the
	href, err := getAttribute(link, "href")
	if err != nil {
		return "", errorWithResponseDump(response, err)
	}

	u, err = resolveRelativeURL(response, href)
	if err != nil {
		return "", err
	}
	response, err = c.HttpClient.Get(u)
	if err != nil {
		return "", fmt.Errorf("failed to get %v: %v", u, err)
	}
	if response.StatusCode != http.StatusOK {
		return "", errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}
	// t.Logf("Response: raw query = %s", response.Request.URL.RawQuery)
	code := strings.Split(response.Request.URL.RawQuery, "&")[1]
	tokenUrl := fmt.Sprintf(codereadySSOTokenEndpoint, keycloakHost)
	// t.Logf("Received code: %s", code)
	// t.Logf("Token url: %s", tokenUrl)

	formValues := url.Values{
		"grant_type":   []string{"authorization_code"},
		"code":         []string{strings.Split(code, "=")[1]},
		"client_id":    []string{clientId},
		"redirect_uri": []string{redirectUrl},
	}

	response, err = c.HttpClient.PostForm(tokenUrl, formValues)
	if err != nil {
		return "", fmt.Errorf("failed to request %s: %s", tokenUrl, err)
	}

	postBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	tokenResponse := struct {
		AccessToken string `json:"access_token"`
	}{}

	json.Unmarshal(postBody, &tokenResponse)

	if tokenResponse.AccessToken == "" {
		return "", fmt.Errorf("failed to get access token: %v", string(postBody))
	}
	return tokenResponse.AccessToken, nil
}

func parseResponse(r *http.Response) (*goquery.Document, error) {
	// Clone the body while reading it so that in case of errors
	// we can dump the response with the body
	var clone bytes.Buffer
	body := io.TeeReader(r.Body, &clone)
	r.Body = ioutil.NopCloser(&clone)

	d, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("failed to create the document: %s", err)
	}

	// <noscript> bug workaround
	// https://github.com/PuerkitoBio/goquery/issues/139#issuecomment-517526070
	d.Find("noscript").Each(func(i int, s *goquery.Selection) {
		s.SetHtml(s.Text())
	})

	return d, nil
}
