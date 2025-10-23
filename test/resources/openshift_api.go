package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	projectv1 "github.com/openshift/api/project/v1"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"net/http"
)

/* #nosec G101 -- This is a false positive */
const (
	OpenshiftPathListProjects = "/api/kubernetes/apis/project.openshift.io/v1/projects"
	OpenshiftPathGetProject   = "/api/kubernetes/apis/project.openshift.io/v1/projects/%v"
	OpenshiftPathListPods     = "/api/kubernetes/api/v1/namespaces/%v/pods"
	OpenshiftPathGetSecret    = "/api/kubernetes/api/v1/namespaces/%s/secrets"
	PathListRHMI              = "/apis/integreatly.org/v1alpha1/namespaces/%s/rhmis"
	PathGetRHMI               = "/apis/integreatly.org/v1alpha1/namespaces/%s/rhmis/%s"
	PathGetRoute              = "/apis/route.openshift.io/v1/namespaces/%s/routes/%s"
)

// API server direct paths (used when authenticating with Bearer tokens)
const (
	ApiPathListProjects = "/apis/project.openshift.io/v1/projects"
	ApiPathGetProject   = "/apis/project.openshift.io/v1/projects/%v"
	ApiPathListPods     = "/api/v1/namespaces/%v/pods"
)

type OpenshiftClient struct {
	HTTPClient *http.Client
	MasterUrl  string
	ApiUrl     string
	// Optional: impersonate a different user/groups when sending requests
	ImpersonateUser   string
	ImpersonateGroups []string
}

func NewOpenshiftClient(httpClient *http.Client, masterUrl string) *OpenshiftClient {
	openshiftAPIURL := strings.Replace(masterUrl, "console-openshift-console.apps.", "api.", 1) + ":6443"

	return &OpenshiftClient{
		MasterUrl:  masterUrl,
		HTTPClient: httpClient,
		ApiUrl:     openshiftAPIURL,
	}
}

// WithImpersonation returns a shallow copy of the client that will impersonate the given user and groups.
func (oc *OpenshiftClient) WithImpersonation(user string, groups []string) *OpenshiftClient {
	copy := *oc
	copy.ImpersonateUser = user
	copy.ImpersonateGroups = append([]string(nil), groups...)
	return &copy
}

// returns all pods in a namesapce
func (oc *OpenshiftClient) ListPods(projectName string) (*corev1.PodList, error) {
	path := fmt.Sprintf(ApiPathListPods, projectName)
	resp, err := oc.DoOpenshiftGetRequest(path)
	if err != nil {
		return nil, fmt.Errorf("error occurred performing oc get request : %w", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	foundPods := &corev1.PodList{}
	if err = json.Unmarshal(respBody, &foundPods); err != nil {
		return nil, fmt.Errorf("error occurred while unmarshalling pod list: %w", err)
	}

	return foundPods, nil
}

// returns a project
func (oc *OpenshiftClient) GetProject(projectName string) (*projectv1.Project, error) {
	path := fmt.Sprintf(ApiPathGetProject, projectName)
	resp, err := oc.DoOpenshiftGetRequest(path)
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
	if err != nil {
		return nil, err
	}
	project := &projectv1.Project{}
	if err := json.Unmarshal(respBody, project); err != nil {
		return nil, fmt.Errorf("error occurred while unmarshalling project: %v", err)
	}

	return project, nil
}

// returns all projects
func (oc *OpenshiftClient) ListProjects() (*projectv1.ProjectList, error) {
	resp, err := oc.DoOpenshiftGetRequest(ApiPathListProjects)
	if err != nil {
		return nil, fmt.Errorf("error occurred performing oc get request : %w", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
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

// makes a post request, expects a path and the body
func (oc *OpenshiftClient) DoOpenshiftPostRequest(path string, data []byte) (*http.Response, error) {
	requestUrl := fmt.Sprintf("https://%s%s", oc.ApiUrl, path)
	req, err := http.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("error reading request: %w", err)
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
	token, err := getOAuthToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if oc.ImpersonateUser != "" {
		req.Header.Set("Impersonate-User", oc.ImpersonateUser)
		for _, g := range oc.ImpersonateGroups {
			req.Header.Add("Impersonate-Group", g)
		}
	}

	// Perform http request
	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while performing http request : %w", err)
	}

	return resp, nil
}

var (
	tokenOnce   sync.Once
	cachedToken string
	cachedErr   error
)

func getOAuthToken() (string, error) {
	tokenOnce.Do(func() {
		t := strings.TrimSpace(os.Getenv("OAUTH_BEARER_TOKEN"))
		if t == "" {
			out, err := exec.Command("oc", "whoami", "-t").Output()
			if err != nil {
				cachedErr = fmt.Errorf("failed to run oc whoami -t: %w", err)
				return
			}
			t = strings.TrimSpace(string(out))
			_ = os.Setenv("OAUTH_BEARER_TOKEN", t)
		}
		cachedToken = t
	})
	if cachedErr != nil || cachedToken == "" {
		return "", fmt.Errorf("OAUTH_BEARER_TOKEN not available: %w", cachedErr)
	}
	return cachedToken, nil
}

// LoginOcAndSetBearerToken logs in to the cluster as the provided user using the oc CLI
// (against the API server derived from the provided master URL), retrieves a bearer token,
// sets it into the OAUTH_BEARER_TOKEN env var, and returns it.
func LoginOcAndSetBearerToken(masterURL, username, password string) (string, error) {
	// Derive API server URL from console master URL
	apiHost := strings.Replace(masterURL, "console-openshift-console.apps.", "api.", 1) + ":6443"
	serverURL := fmt.Sprintf("https://%s", apiHost)

	kubeconfigFile, err := os.CreateTemp("", "oc-kubeconfig-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp kubeconfig: %w", err)
	}
	// Ensure temporary kubeconfig is cleaned up
	defer os.Remove(kubeconfigFile.Name())
	_ = kubeconfigFile.Close()

	// Perform non-interactive oc login using a dedicated kubeconfig
	loginCmd := exec.Command("oc", "login", serverURL, "-u", username, "-p", password, "--insecure-skip-tls-verify=true", "--kubeconfig", kubeconfigFile.Name())
	if out, err := loginCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to oc login: %w, output: %s", err, string(out))
	}

	// Retrieve the token for that session
	whoamiCmd := exec.Command("oc", "whoami", "-t", "--kubeconfig", kubeconfigFile.Name())
	out, err := whoamiCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run oc whoami -t: %w", err)
	}
	token := strings.TrimSpace(string(out))

	if err := os.Setenv("OAUTH_BEARER_TOKEN", token); err != nil {
		return "", fmt.Errorf("failed to set OAUTH_BEARER_TOKEN: %w", err)
	}

	return token, nil
}
