package resources

import (
	"encoding/json"
	"fmt"
	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"

	"io/ioutil"
	"net/http"
)

type OpenshiftClient struct {
	HTTPClient *http.Client
}

// returns all pods in a namesapce
func (oc *OpenshiftClient) DoOpenshiftGetPodsForNamespacePods(masterUrl, namespace string) (*corev1.PodList, error) {
	path := fmt.Sprintf("/api/kubernetes/api/v1/namespaces/%s/pods", namespace)
	resp, err := oc.DoOpenshiftGetRequest(masterUrl, path)
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

// returns all projects
func (oc *OpenshiftClient) DoOpenshiftGetProjects(masterUrl string) (*projectv1.ProjectList, error) {
	resp, err := oc.DoOpenshiftGetRequest(masterUrl, PathProjects)
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
