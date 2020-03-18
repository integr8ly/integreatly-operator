package resources

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"

	"io/ioutil"
	"net/http"
)

// returns all pods in a namesapce
func DoOpenshiftGetPodsForNamespacePods(masterUrl, namespace, token string) (*corev1.PodList, error) {
	path := fmt.Sprintf("/api/kubernetes/api/v1/namespaces/%s/pods", namespace)
	resp, err := DoOpenshiftGetRequest(masterUrl, path, token)
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
func DoOpenshiftGetProjects(masterUrl, token string) (*projectv1.ProjectList, error) {
	resp, err := DoOpenshiftGetRequest(masterUrl, PathProjects, token)
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
func DoOpenshiftGetRequest(masterUrl string, path string, token string) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", masterUrl, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating new http request : %w", err)
	}
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", OpenshiftTokenName, token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while performing http request : %w", err)
	}

	return resp, nil
}
