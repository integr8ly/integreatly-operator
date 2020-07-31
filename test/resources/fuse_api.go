package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	testingIDP                  = "testing-idp"
	fuseApiPingUrl              = "%v/api/v1/public/environments"
	fuseApiIntegrationsUrl      = "%v/api/v1/integrations"
	fuseApiDeleteIntegrationUrl = "%v/api/v1/integrations/%v"
)

type FuseApiClient interface {
	Ping() error
	CountIntegrations() (int, error)
	CreateIntegration(schema string) (string, error)
	DeleteIntegration(id string) error
}

type FuseApiClientImpl struct {
	host   string
	client *http.Client
}

func NewFuseApiClient(host string, client *http.Client) FuseApiClient {
	return &FuseApiClientImpl{
		host:   host,
		client: client,
	}
}

// Check fuse connectivity by calling a public endpoint
func (r *FuseApiClientImpl) Ping() error {
	url := fmt.Sprintf(fuseApiPingUrl, r.host)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("fuse ping: expected status 200 but got %v", resp.StatusCode))
	}

	return nil
}

// Creates a simple integration using timer and logger
func (r *FuseApiClientImpl) CreateIntegration(schema string) (string, error) {
	url := fmt.Sprintf(fuseApiIntegrationsUrl, r.host)

	payload := bytes.NewBuffer([]byte(schema))
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	// The fuse API expects this, otherwise a 403 is returned because of a missing
	// csrf token
	req.Header.Set("SYNDESIS-XSRF-TOKEN", "awesome")
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	apiResult := struct {
		ID string `json:"id"`
	}{}

	err = json.Unmarshal(data, &apiResult)

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("error creating fuse integration, expected status 201 but received %v", resp.StatusCode))
	}

	return apiResult.ID, nil
}

// Deletes an integration by id
func (r *FuseApiClientImpl) DeleteIntegration(id string) error {
	url := fmt.Sprintf(fuseApiDeleteIntegrationUrl, r.host, id)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	// The fuse API expects this, otherwise a 403 is returned because of a missing
	// csrf token
	req.Header.Set("SYNDESIS-XSRF-TOKEN", "awesome")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.New(fmt.Sprintf("error deleting fuse integration, expected status 204 but received %v", resp.StatusCode))
	}

	return nil
}

// Returns the number of existing integrations
func (r *FuseApiClientImpl) CountIntegrations() (int, error) {
	url := fmt.Sprintf(fuseApiIntegrationsUrl, r.host)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New(fmt.Sprintf("fuse api: expected status 200 but got %v", resp.StatusCode))
	}

	apiResult := struct {
		TotalCount int `json:"totalCount"`
	}{}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal(data, &apiResult)
	if err != nil {
		return 0, err
	}

	return apiResult.TotalCount, nil
}
