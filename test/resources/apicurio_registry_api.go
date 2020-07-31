package resources

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	Avro ArtifactType = "AVRO"
)

type ArtifactType string

type ApicurioRegistryApiClient interface {
	CreateArtifact(id string, artifactType ArtifactType, data string) error
	ReadArtifact(id string) (string, error)
	DeleteArtifact(id string) error
}

type ApicurioRegistryApiClientImpl struct {
	host       string
	httpClient *http.Client
}

func NewApicurioRegistryApiClient(host string, httpClient *http.Client) ApicurioRegistryApiClient {
	return &ApicurioRegistryApiClientImpl{
		host:       host,
		httpClient: httpClient,
	}
}

func (r *ApicurioRegistryApiClientImpl) CreateArtifact(id string, artifactType ArtifactType, data string) error {
	url := fmt.Sprintf("http://%v/api/artifacts", r.host)
	body := bytes.NewBuffer([]byte(data))

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}

	switch artifactType {
	case Avro:
		req.Header.Set("Content-Type", fmt.Sprintf("application/json; artifactType=%v", Avro))
	}

	req.Header.Set("X-Registry-ArtifactId", id)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(fmt.Sprintf("expected status 200 but received %v", resp.StatusCode))
	}

	return nil
}

func (r *ApicurioRegistryApiClientImpl) ReadArtifact(id string) (string, error) {
	url := fmt.Sprintf("http://%v/api/artifacts/%v", r.host, id)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(fmt.Sprintf("expected status 200 but received %v", resp.StatusCode))
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	data := string(bytes)

	return data, nil
}

func (r *ApicurioRegistryApiClientImpl) DeleteArtifact(id string) error {
	url := fmt.Sprintf("http://%v/api/artifacts/%v", r.host, id)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf(fmt.Sprintf("expected status %v but received %v", http.StatusNoContent, resp.StatusCode))
	}

	return nil
}
