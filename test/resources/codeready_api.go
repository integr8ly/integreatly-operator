package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	pingUrl             = "%v/dashboard/"
	createWorkspacePath = "%v/api/workspace/devfile"
	getWorkspacePath    = "%v/api/workspace/%v"
	stopWorkspacePath   = "%v/api/workspace/%v/runtime"
	deleteWorkspacePath = "%v/api/workspace/%v"
)

type CodereadyApiResult struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
}

type CodereadyApiClient interface {
	Ping() error
	GetWorkspace() (*CodereadyApiResult, error)
	CreateWorkspace() (string, error)
	StopWorkspace() error
	DeleteWorkspace() error
}

type CodereadyApiClientImpl struct {
	host        string
	httpClient  *http.Client
	accessToken string
}

func NewCodereadyApiClient(httpClient *http.Client, host string, accessToken string) *CodereadyApiClientImpl {
	return &CodereadyApiClientImpl{
		host:        host,
		httpClient:  httpClient,
		accessToken: accessToken,
	}
}

func (c *CodereadyApiClientImpl) Ping() error {
	url := fmt.Sprintf(pingUrl, c.host)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create a new request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.accessToken))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("expected response status %v but got %v", http.StatusOK, res.StatusCode)
	}

	return nil
}

/*
Sends a request to /api/workspace/devfile endpoint to create a workspace and starts it as soon as it is created.
Returns the workspace ID upon successful workspace creation, otherwise it returns an error.
*/
func (c *CodereadyApiClientImpl) CreateWorkspace(devfile []byte) (string, error) {
	url := fmt.Sprintf(createWorkspacePath, c.host)
	url = fmt.Sprintf("%s?start-after-create=true", url)

	payload := bytes.NewBufferString(string(devfile))

	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return "", fmt.Errorf("failed to create a new request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("expected response status %v but got %v: %v", http.StatusCreated, res.StatusCode, string(data))
	}

	result := &CodereadyApiResult{}
	if err := json.Unmarshal(data, result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response %v: %v", err, string(data))
	}

	return result.ID, nil
}

// Gets a workspace by id
func (c *CodereadyApiClientImpl) GetWorkspace(id string) (*CodereadyApiResult, error) {
	url := fmt.Sprintf(getWorkspacePath, c.host, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.accessToken))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected response status %v but got %v: %v", http.StatusOK, res.StatusCode, string(data))
	}

	result := &CodereadyApiResult{}
	if err := json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result, nil
}

// Stops a workspace specified by id
func (c *CodereadyApiClientImpl) StopWorkspace(id string) error {
	url := fmt.Sprintf(stopWorkspacePath, c.host, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create a new request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.accessToken))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %v", err)
		}
		return fmt.Errorf("expected response status %v but got %v: %v", http.StatusNoContent, res.StatusCode, string(data))
	}

	return nil
}

// Deletes a workspace specified by id
func (c *CodereadyApiClientImpl) DeleteWorkspace(id string) error {
	url := fmt.Sprintf(deleteWorkspacePath, c.host, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create a new request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.accessToken))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %v", err)
		}
		return fmt.Errorf("expected response status %v but got %v: %v", http.StatusNoContent, res.StatusCode, string(data))
	}

	return nil
}
