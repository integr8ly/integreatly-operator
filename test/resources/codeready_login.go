package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
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
}

func NewCodereadyLoginClient(httpClient *http.Client, client k8sclient.Client, masterUrl, idp, username, password string) *CodereadyLoginClient {
	return &CodereadyLoginClient{
		HttpClient: httpClient,
		Client:     client,
		MasterUrl:  masterUrl,
		IdpName:    idp,
		Username:   username,
		Password:   password,
	}
}

// Login user to openshift and wait for the user to be reconciled to the openshift realm in keycloak
func (c *CodereadyLoginClient) OpenshiftLogin(namespacePrefix string) error {
	authUrl := fmt.Sprintf("%s/auth/login", c.MasterUrl)
	if err := DoAuthOpenshiftUser(authUrl, c.Username, c.Password, c.HttpClient, c.IdpName); err != nil {
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

	code := strings.Split(response.Request.URL.RawQuery, "&")[1]
	tokenUrl := fmt.Sprintf(codereadySSOTokenEndpoint, keycloakHost)

	formValues := url.Values{
		"grant_type":   []string{"authorization_code"},
		"code":         []string{strings.Split(code, "=")[1]},
		"client_id":    []string{clientId},
		"redirect_uri": []string{redirectUrl},
	}

	response, err = c.HttpClient.PostForm(tokenUrl, formValues)
	if err != nil {
		return "", fmt.Errorf("failed to request %s: %s", u, err)
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

func dumpResponse(r *http.Response) string {
	msg := "> Request\n"
	bytes, err := httputil.DumpRequestOut(r.Request, false)
	if err != nil {
		msg += fmt.Sprintf("failed to dump the request: %s", err)
	} else {
		msg += string(bytes)
	}
	msg += "\n"

	msg += "< Response\n"
	bytes, err = httputil.DumpResponse(r, true)
	if err != nil {
		msg += fmt.Sprintf("failed to dump the response: %s", err)
	} else {
		msg += string(bytes)
	}
	msg += "\n"

	return msg
}

func errorWithResponseDump(r *http.Response, err error) error {
	return fmt.Errorf("%s\n\n%s", err, dumpResponse(r))
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

func resolveRelativeURL(r *http.Response, relativeURL string) (string, error) {
	u, err := url.Parse(relativeURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse the url %s: %s", relativeURL, err)
	}

	u = r.Request.URL.ResolveReference(u)

	return u.String(), nil
}

func findElement(d *goquery.Document, selector string) (*goquery.Selection, error) {
	e := d.Find(selector)
	if e.Length() == 0 {
		return nil, fmt.Errorf("failed to find an element matching the selector %s", selector)
	}
	if e.Length() > 1 {
		return nil, fmt.Errorf("multiple element founded matching the selector %s", selector)
	}

	return e, nil
}

func getAttribute(element *goquery.Selection, name string) (string, error) {
	v, ok := element.Attr(name)
	if !ok {
		e, err := element.Html()
		if err != nil {
			e = fmt.Sprintf("failed to get the html content: %s", err)
		}

		return "", fmt.Errorf("the element '%s' doesn't have the %s attribute", e, name)
	}
	return v, nil
}
