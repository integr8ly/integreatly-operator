package common

import (
	"bytes"
	goctx "context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/resources"
	v1 "github.com/openshift/api/route/v1"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/cookiejar"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CheDevFile struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty"`
	Commands          []DevFileCommand   `yaml:"commands,omitempty"`
	Components        []DevFileComponent `yaml:"components,omitempty"`
	Projects          []DevFileProject   `yaml:"projects,omitempty"`
}

type DevFileCommand struct {
	Name    string   `yaml:"name,omitempty"`
	Actions []Action `yaml:"actions,omitempty"`
}

type Action struct {
	Command          string      `yaml:"command,omitempty"`
	Component        string      `yaml:"component,omitempty"`
	Type             string      `yaml:"type,omitempty"`
	Workdir          string      `yaml:"workdir,omitempty"`
	ReferenceContent interface{} `yaml:"referenceContent,omitempty"`
}

type DevFileComponent struct {
	Alias        string            `yaml:"alias,omitempty"`
	ID           string            `yaml:"id,omitempty"`
	Endpoints    []Endpoint        `yaml:"endpoints,omitempty"`
	Env          []Env             `yaml:"env,omitempty"`
	MemoryLimit  string            `yaml:"memoryLimit,omitempty"`
	MountSources bool              `yaml:"mountSources,omitempty"`
	Preferences  map[string]string `yaml:"preferences,omitempty"`
	Type         string            `yaml:"type,omitempty"`
}

type Endpoint struct {
	Name string `yaml:"name,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

type Env struct {
	Name  string `yaml:"name,omitempty"`
	Value string `yaml:"value,omitempty"`
}

type DevFileProject struct {
	ClonePath string        `yaml:"clonePath,omitempty"`
	Name      string        `yaml:"name,omitempty"`
	Source    ProjectSource `yaml:"source,omitempty"`
}

type ProjectSource struct {
	Type     string `yaml:"type,omitempty"`
	Location string `yaml:"location,omitempty"`
}

var (
	goDevFileLocation = fmt.Sprintf("https://raw.githubusercontent.com/redhat-developer/codeready-workspaces/%s.GA/dependencies/che-devfile-registry/devfiles/05_go/devfile.yaml", integreatlyv1alpha1.VersionCodeReadyWorkspaces)
)

func TestCodereadyCrudlPermisssions(t *testing.T, ctx *TestingContext) {
	t.Log("Test codeready workspace creation")

	rhmi, err := getRHMI(ctx)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}

	// Get the fuse host url from the rhmi status
	cheHost := rhmi.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductCodeReadyWorkspaces].Host
	keycloakHost := rhmi.Status.Stages[integreatlyv1alpha1.AuthenticationStage].Products[integreatlyv1alpha1.ProductRHSSO].Host


	redirectUrl := fmt.Sprintf("%v/dashboard/", cheHost)
	client, token, err := resources.AuthProductClient(oauthRoute.Spec.Host, masterURL, redirectUrl, keycloakHost, "che-client", "test-user01", DefaultPassword)
	if err != nil {
		t.Fatal(err)
	}

	queryWorkspaceUrl := fmt.Sprintf("%v/api/workspace?skipCount=0&maxItems=30", cheHost)
	req, err := http.NewRequest(http.MethodGet, queryWorkspaceUrl, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(data)

	// TODO: Get auth token (Sync up with Ciaran Roche on this) [Blocked until PR is merged: for now get auth token via oc cli]
	// TODO: Get CodeReady access token
	// Login to openshift first
	// Login to codeready
	// Get auth code from the location/request url of the last call
	// Get access token

	t.Log("test: ")

	// Get Codeready Go dev file. This will be used as a payload to create a workspace
	// goDevfile, err := getDevFile(httpClient, goDevFileLocation)
	// if err != nil {
	// 	t.Fatalf("failed to get che devfile: %s", err.Error())
	// }

	// TODO: Create workspace via /api/workspace/devfile endpoint (NOTE: https://codeready-redhat-rhmi-codeready-workspaces.apps.jbriones.w8u6.s1.devshift.org/swagger/#/)

	// TODO: Check all resources are up and running (NOTE: All resources are labeled with the workspace id)

	// t.Fatal("reTuRn f0r 0uT9uTz")
}

// creates an http client
func createHttpClient() (*http.Client, error) {
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
	return &http.Client{
		Transport: tr,
		Jar:       jar,
	}, nil
}

func getContentFromURL(httpClient *http.Client, url string) ([]byte, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get file content from %s. Status: %d", url, response.StatusCode)
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func getDevFile(httpClient *http.Client, url string) ([]byte, error) {
	fileContent, err := getContentFromURL(httpClient, url)
	if err != nil {
		return nil, fmt.Errorf("failed to file content from url %s: %s", url, err.Error())
	}

	devFile := &CheDevFile{}
	if err := yaml.Unmarshal(fileContent, devFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal devfile content: %s", err.Error())
	}

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // Ensures that characters such as '&' does not get converted to unicode
	if err := encoder.Encode(devFile); err != nil {
		return nil, fmt.Errorf("failed to encode: %s", err.Error())
	}

	return buffer.Bytes(), nil
}
