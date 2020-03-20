package common

import (
	"bytes"
	goctx "context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testingIDP       = "testing-idp"
	testUser         = "test-user01"
	testUserPassword = "Password1"
)

func TestCreateAddressSpace(t *testing.T, ctx *TestingContext) {

	h, err := getAMQOnlineHost(ctx.Client)
	if err != nil {
		t.Fatal(err)
	}

	client, err := porxyOAuth(h, testUser, testUserPassword)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", client.Jar)
}

func getAMQOnlineHost(client dynclient.Client) (string, error) {

	var r v1.Route
	err := client.Get(goctx.TODO(), types.NamespacedName{Namespace: "redhat-rhmi-amq-online", Name: "console"}, &r)
	if err != nil {
		return "", err
	}

	return r.Spec.Host, nil
}

func porxyOAuth(host string, username string, password string) (*http.Client, error) {

	// Create the http client with a cookie jar
	j, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initalize the cookie jar: %s", err)
	}

	client := &http.Client{
		Jar:           j,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return nil },
	}

	// Start the authentication
	u := fmt.Sprintf("https://%s/oauth/start", host)
	response, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return nil, errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// Select the testing IDP
	document, err := parseResponse(response)
	if err != nil {
		return nil, errorWithResponseDump(response, err)
	}

	// find the link to the testing IDP
	link, err := findElement(document, fmt.Sprintf("a:contains('%s')", testingIDP))
	if err != nil {
		return nil, errorWithResponseDump(response, err)
	}

	// get the url from the
	href, err := getAttribute(link, "href")
	if err != nil {
		return nil, errorWithResponseDump(response, err)
	}

	u, err = resolveRelativeURL(response, href)
	if err != nil {
		return nil, err
	}

	response, err = client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to request %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return nil, errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// Submit the username and password
	document, err = parseResponse(response)
	if err != nil {
		return nil, errorWithResponseDump(response, err)
	}

	// find the form for the login
	form, err := findElement(document, "#kc-form-login")
	if err != nil {
		return nil, errorWithResponseDump(response, err)
	}

	// retrieve the action of the form
	action, err := getAttribute(form, "action")
	if err != nil {
		return nil, errorWithResponseDump(response, err)
	}

	u, err = resolveRelativeURL(response, action)
	if err != nil {
		return nil, err
	}

	// subbmit the form with the username and password
	v := url.Values{"username": []string{username}, "password": []string{password}}
	response, err = client.PostForm(u, v)
	if err != nil {
		return nil, fmt.Errorf("failed to request %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return nil, errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}
	// if the login succeded the request should be redirected to the starting location
	if response.Request.URL.Host != host {
		return nil, errorWithResponseDump(response, fmt.Errorf("the request wasn't redirect back to %s", host))
	}

	return client, nil
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
