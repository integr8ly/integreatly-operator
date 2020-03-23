package resources

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	OpenshiftPathListProjects = "/api/kubernetes/apis/project.openshift.io/v1/projects"
	OpenshiftPathGetProject   = "/api/kubernetes/apis/project.openshift.io/v1/projects/%v"
	OpenshiftPathListPods     = "/api/kubernetes/api/v1/namespaces/%v/pods"
	OpenshiftPathGetSecret    = "/api/kubernetes/api/v1/namespaces/%s/secrets"
)

type OpenshiftClient struct {
	HTTPClient *http.Client
	MasterUrl  string
}

func NewOpenshiftClient(httpClient *http.Client, masterUrl string) *OpenshiftClient {
	return &OpenshiftClient{
		MasterUrl:  masterUrl,
		HTTPClient: httpClient,
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

func DoOpenshiftCreateProject(apiURL string, token string, projectCR *projectv1.ProjectRequest) error {

	projectJson, err := json.Marshal(projectCR)
	if err != nil {
		return fmt.Errorf("failed to marshal projectCR: %w", err)
	}

	response, err := DoOpenshiftPostRequest(apiURL, PathProjectRequests, token, projectJson)
	if err != nil {
		return fmt.Errorf("error occured durning oc request : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("request to %s failed with code %d", PathProjectRequests, response.StatusCode)
	}

	return nil
}

func DoOpenshiftCreateServiceInANamespace(openshiftAPIURL string, token string, namespace string, serviceCR *corev1.Service) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/services", namespace)
	serviceJSON, err := json.Marshal(serviceCR)
	if err != nil {
		return fmt.Errorf("failed to marshal serviceCR: %w", err)
	}

	response, err := DoOpenshiftPostRequest(openshiftAPIURL, path, token, serviceJSON)
	if err != nil {
		return fmt.Errorf("error occured durning oc request : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("request to %s failed with code %d", path, response.StatusCode)
	}

	return nil
}

func DoOpenshiftCreatePodInANamespace(apiURL string, token string, namespace string, podCR *corev1.Pod) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods", namespace)
	podJSON, err := json.Marshal(podCR)
	if err != nil {
		return fmt.Errorf("failed to marshal serviceCR: %w", err)
	}

	response, err := DoOpenshiftPostRequest(apiURL, path, token, podJSON)
	if err != nil {
		return fmt.Errorf("error occured durning oc request : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("request to %s failed with code %d", path, response.StatusCode)
	}
	return nil
}

// makes a get request, expects master url, a path and a token
func (oc *OpenshiftClient) DoOpenshiftGetRequest(masterUrl string, path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", masterUrl, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating new http request : %w", err)
	}
	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while performing http request : %w", err)
	}
	return resp, nil
}

// makes a post request, expects openshift api url, a path and a token
func DoOpenshiftPostRequest(openshiftAPIURL string, path string, token string, data []byte) (*http.Response, error) {

	// openshift api url required for POST requests
	apiURL := fmt.Sprintf("https://%s%s", openshiftAPIURL, path)
	req, err := http.NewRequest(
		http.MethodPost,
		apiURL,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("Error reading request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error reading response: %w", err)
	}

	return resp, nil
}
