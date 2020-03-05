package common

import (
	"bytes"
	goctx "context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	projectv1 "github.com/openshift/api/project/v1"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	openshiftTokenName = "openshift-session-token"
	defaultIDP         = "testing-idp"

	openshiftAuthenticationNamespace = "openshift-authentication"
	openshiftOAuthRouteName          = "oauth-openshift"

	pathProjects = "/api/kubernetes/apis/project.openshift.io/v1/projects"
	pathFusePods = "/api/kubernetes/api/v1/namespaces/redhat-rhmi-fuse/pods"
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

// struct used to create query string for fuse logs endpoint
type LogOptions struct {
	Container string `url:"container"`
	Follow    string `url:"follow"`
	TailLines string `url:"tailLines"`
}

// returns all pods in a namesapce
func doOpenshiftGetNamespacePods(masterUrl, path, token string) (*corev1.PodList, error) {
	resp, err := doOpenshiftGetRequest(masterUrl, path, token)
	if err != nil {
		return nil, fmt.Errorf("error occured durning oc request : %w", err)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	foundPods := &corev1.PodList{}
	if err = json.Unmarshal(respBody, &foundPods); err != nil {
		return nil, fmt.Errorf("error occured while unmarshalling pod list: %w", err)
	}

	return foundPods, nil
}

// returns all projects
func doOpenshiftGetProjects(masterUrl, token string) (*projectv1.ProjectList, error) {
	resp, err := doOpenshiftGetRequest(masterUrl, pathProjects, token)
	if err != nil {
		return nil, fmt.Errorf("error occured durning oc request : %w", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	foundProjects := &projectv1.ProjectList{}
	if err = json.Unmarshal(respBody, &foundProjects); err != nil {
		return nil, fmt.Errorf("error occured while unmarshalling project list: %w", err)
	}

	return foundProjects, nil
}

// makes a get request, expects master url, a path and a token
func doOpenshiftGetRequest(masterUrl string, path string, token string) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", masterUrl, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error occured while creating new requests : %w", err)
	}
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", openshiftTokenName, token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occured while making http request : %w", err)
	}

	return resp, nil
}

// doAuthOpenshiftUser this function expects users and IDP to be created via `./scripts/setup-sso-idp.sh`
func doAuthOpenshiftUser(oauthUrl string, masterURL string, idp string, username string, password string) (string, error) {
	// declare transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// declare new cookie jar om nom nom
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return "", fmt.Errorf("error occured creating a new cookie jar : %w", err)
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

	// check response for token
	for _, c := range resp.Cookies() {
		if c.Name == openshiftTokenName {
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
	state := parsedQuery["state"][0]
	if state == "" {
		return "", errors.New("failed to find state during parse")
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

func execToPod(command string, podName string, namespace string, container string, ctx *TestingContext) (string, error) {
	req := ctx.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(ctx.KubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("error in Stream: %v", err)
	}
	return stdout.String(), nil
}

// difference one-way diff that return strings in sliceSource that are not in sliceTarget
func difference(sliceSource, sliceTarget []string) []string {
	// create an empty lookup map with keys from sliceTarget
	diffSourceLookupMap := make(map[string]struct{}, len(sliceTarget))
	for _, item := range sliceTarget {
		diffSourceLookupMap[item] = struct{}{}
	}
	// use the lookup map to find items in sliceSource that are not in sliceTarget
	// and store them in a diff slice
	var diff []string
	for _, item := range sliceSource {
		if _, found := diffSourceLookupMap[item]; !found {
			diff = append(diff, item)
		}
	}
	return diff
}

// Is the cluster using on cluster or external storage
func isClusterStorage(ctx *TestingContext) (bool, error) {
	rhmi := &integreatlyv1alpha1.RHMI{}

	// get the RHMI custom resource to check what storage type is being used
	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: InstallationName, Namespace: rhmiOperatorNamespace}, rhmi)
	if err != nil {
		return true, fmt.Errorf("error getting RHMI CR: %v", err)
	}

	if rhmi.Spec.UseClusterStorage == "true" {
		return true, nil
	}
	return false, nil
}

// returns rhmi
func getRHMI(ctx *TestingContext) (*integreatlyv1alpha1.RHMI, error) {
	// get the RHMI custom resource to check what storage type is being used
	rhmi := &integreatlyv1alpha1.RHMI{}
	ns := fmt.Sprintf("%soperator", namespacePrefix)
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: InstallationName, Namespace: ns}, rhmi); err != nil {
		return nil, fmt.Errorf("error getting RHMI CR: %w", err)
	}
	return rhmi, nil
}
