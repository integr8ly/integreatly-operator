package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"io/ioutil"
	"net/http"
	"net/url"

	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	OpenshiftPathListProjects     = "/api/kubernetes/apis/project.openshift.io/v1/projects"
	OpenshiftPathGetProject       = "/api/kubernetes/apis/project.openshift.io/v1/projects/%v"
	OpenshiftPathListPods         = "/api/kubernetes/api/v1/namespaces/%v/pods"
	OpenshiftPathGetSecret        = "/api/kubernetes/api/v1/namespaces/%s/secrets"
	PathListRHMIConfig            = "/apis/integreatly.org/v1alpha1/namespaces/%s/rhmiconfigs"
	PathGetRHMIConfig             = "/apis/integreatly.org/v1alpha1/namespaces/%s/rhmiconfigs/%s"
	PathGetRoute                  = "/apis/route.openshift.io/v1/namespaces/%s/routes/%s"
	PathListStandardInfraConfig   = "/apis/admin.enmasse.io/v1beta1/namespaces/%s/standardinfraconfigs"
	PathGetStandardInfraConfig    = "/apis/admin.enmasse.io/v1beta1/namespaces/%s/standardinfraconfigs/%s"
	PathListBrokeredInfraConfig   = "/apis/admin.enmasse.io/v1beta1/namespaces/%s/brokeredinfraconfigs"
	PathGetBrokeredInfraConfig    = "/apis/admin.enmasse.io/v1beta1/namespaces/%s/brokeredinfraconfigs/%s"
	PathListAddressSpacePlan      = "/apis/admin.enmasse.io/v1beta2/namespaces/%s/addressspaceplans"
	PathGetAddressSpacePlan       = "/apis/admin.enmasse.io/v1beta2/namespaces/%s/addressspaceplans/%s"
	PathListAddressPlan           = "/apis/admin.enmasse.io/v1beta2/namespaces/%s/addressplans"
	PathGetAddressPlan            = "/apis/admin.enmasse.io/v1beta2/namespaces/%s/addressplans/%s"
	PathListAuthenticationService = "/apis/admin.enmasse.io/v1beta1/namespaces/%s/authenticationservices"
	PathGetAuthenticationService  = "/apis/admin.enmasse.io/v1beta1/namespaces/%s/authenticationservices/%s"
)

type OpenshiftClient struct {
	HTTPClient *http.Client
	MasterUrl  string
	ApiUrl     string
}

func NewOpenshiftClient(httpClient *http.Client, masterUrl string) *OpenshiftClient {
	openshiftAPIURL := strings.Replace(masterUrl, "console-openshift-console.apps.", "api.", 1) + ":6443"

	return &OpenshiftClient{
		MasterUrl:  masterUrl,
		HTTPClient: httpClient,
		ApiUrl:     openshiftAPIURL,
	}
}

// returns all pods in a namesapce
func (oc *OpenshiftClient) ListPods(projectName string) (*corev1.PodList, error) {
	path := fmt.Sprintf(OpenshiftPathListPods, projectName)
	resp, err := oc.GetRequest(path)
	if err != nil {
		return nil, fmt.Errorf("error occurred performing oc get request : %w", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	foundPods := &corev1.PodList{}
	if err = json.Unmarshal(respBody, &foundPods); err != nil {
		return nil, fmt.Errorf("error occurred while unmarshalling pod list: %w", err)
	}

	return foundPods, nil
}

// returns a project
func (oc *OpenshiftClient) GetProject(projectName string) (*projectv1.Project, error) {
	path := fmt.Sprintf(OpenshiftPathGetProject, projectName)
	resp, err := oc.GetRequest(path)
	if err != nil {
		return nil, fmt.Errorf("error occurred performing oc get request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("expected status %v but got %v: forbidden to access %v project", http.StatusOK, resp.StatusCode, projectName)
		}
		return nil, fmt.Errorf("expected status %v but got %v", http.StatusOK, resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	project := &projectv1.Project{}
	if err := json.Unmarshal(respBody, project); err != nil {
		return nil, fmt.Errorf("error occurred while unmarshalling project: %v", err)
	}

	return project, nil
}

// returns all projects
func (oc *OpenshiftClient) ListProjects() (*projectv1.ProjectList, error) {
	resp, err := oc.GetRequest(OpenshiftPathListProjects)
	if err != nil {
		return nil, fmt.Errorf("error occurred performing oc get request : %w", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	foundProjects := &projectv1.ProjectList{}
	if err = json.Unmarshal(respBody, &foundProjects); err != nil {
		return nil, fmt.Errorf("error occured while unmarshalling project list: %w", err)
	}

	return foundProjects, nil
}

// makes a get request to the specified openshift api endpoint
func (oc *OpenshiftClient) GetRequest(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", oc.MasterUrl, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating new http request : %w", err)
	}
	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while performing http request : %w", err)
	}
	return resp, nil
}

func (oc *OpenshiftClient) DoOpenshiftCreateProject(projectCR *projectv1.ProjectRequest) error {

	projectJson, err := json.Marshal(projectCR)
	if err != nil {
		return fmt.Errorf("failed to marshal projectCR: %w", err)
	}

	response, err := oc.DoOpenshiftPostRequest(PathProjectRequests, projectJson)
	if err != nil {
		return fmt.Errorf("error occured durning oc request : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("request to %s failed with code %d", PathProjectRequests, response.StatusCode)
	}

	return nil
}

func (oc *OpenshiftClient) DoOpenshiftCreateServiceInANamespace(namespace string, serviceCR *corev1.Service) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/services", namespace)
	serviceJSON, err := json.Marshal(serviceCR)
	if err != nil {
		return fmt.Errorf("failed to marshal serviceCR: %w", err)
	}

	response, err := oc.DoOpenshiftPostRequest(path, serviceJSON)
	if err != nil {
		return fmt.Errorf("error occured durning oc request : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("request to %s failed with code %d", path, response.StatusCode)
	}

	return nil
}

func (oc *OpenshiftClient) DoOpenshiftCreatePodInANamespace(namespace string, podCR *corev1.Pod) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods", namespace)
	podJSON, err := json.Marshal(podCR)
	if err != nil {
		return fmt.Errorf("failed to marshal serviceCR: %w", err)
	}

	response, err := oc.DoOpenshiftPostRequest(path, podJSON)
	if err != nil {
		return fmt.Errorf("error occured durning oc request : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("request to %s failed with code %d", path, response.StatusCode)
	}
	return nil
}

// makes a get request, expects a path
func (oc *OpenshiftClient) DoOpenshiftGetRequest(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", oc.ApiUrl, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating new http request : %w", err)
	}

	return oc.PerformRequest(req)
}

// get Oauth token from cookie
func getOauthTokenFromCookie(masterURL string, client *http.Client) (string, error) {
	clusterURL, err := url.Parse(fmt.Sprintf("https://%s/api/", masterURL))
	if err != nil {
		return "", fmt.Errorf("unable to parse the url: %w", err)
	}

	var token string = ""
	for _, cookie := range client.Jar.Cookies(clusterURL) {
		if cookie.Name == "openshift-session-token" {
			token = cookie.Value
			break
		}
	}

	if token == "" {
		return "", fmt.Errorf("token not found: %w", err)
	}

	return token, nil
}

// makes a post request, expects a path and the body
func (oc *OpenshiftClient) DoOpenshiftPostRequest(path string, data []byte) (*http.Response, error) {
	requestUrl := fmt.Sprintf("https://%s%s", oc.ApiUrl, path)
	req, err := http.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Error reading request: %w", err)
	}

	return oc.PerformRequest(req)
}

// makes a put request, expects a path and the body
func (oc *OpenshiftClient) DoOpenshiftPutRequest(path string, data []byte) (*http.Response, error) {
	requestUrl := fmt.Sprintf("https://%s%s", oc.ApiUrl, path)
	req, err := http.NewRequest(http.MethodPut, requestUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("error reading request: %w", err)
	}

	return oc.PerformRequest(req)
}

// make a delete request, expects a path
func (oc *OpenshiftClient) DoOpenshiftDeleteRequest(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("https://%s%s", oc.ApiUrl, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating new http request : %w", err)
	}

	return oc.PerformRequest(req)
}

// Common function for setting auth headers on request and performing the request
func (oc *OpenshiftClient) PerformRequest(req *http.Request) (*http.Response, error) {
	// Set auth headers from current user session
	token, err := getOauthTokenFromCookie(oc.MasterUrl, oc.HTTPClient)
	if err != nil {
		return nil, fmt.Errorf("error getting the oauth token client: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Perform http request
	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while performing http request : %w", err)
	}

	return resp, nil
}
