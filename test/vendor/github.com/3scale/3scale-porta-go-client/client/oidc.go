package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	oidcResourceEndpoint = "/admin/api/services/%d/proxy/oidc_configuration.json"
)

// OIDCConfiguration fetches 3scale product oidc configuration
func (c *ThreeScaleClient) OIDCConfiguration(productID int64) (*OIDCConfiguration, error) {
	endpoint := fmt.Sprintf(oidcResourceEndpoint, productID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	oidcConf := &OIDCConfiguration{}
	err = handleJsonResp(resp, http.StatusOK, oidcConf)
	return oidcConf, err
}

// UpdateOIDCConfiguration Update 3scale product oidc configuration
func (c *ThreeScaleClient) UpdateOIDCConfiguration(productID int64, oidcConf *OIDCConfiguration) (*OIDCConfiguration, error) {
	endpoint := fmt.Sprintf(oidcResourceEndpoint, productID)

	bodyArr, err := json.Marshal(oidcConf)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bodyArr)
	req, err := c.buildPatchJSONReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	newConf := &OIDCConfiguration{}
	err = handleJsonResp(resp, http.StatusOK, newConf)
	return newConf, err
}
