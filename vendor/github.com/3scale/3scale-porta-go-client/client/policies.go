package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	policiesResourceEndpoint = "/admin/api/services/%d/proxy/policies.json"
)

// Policies fetches 3scale product policy chain
func (c *ThreeScaleClient) Policies(productID int64) (*PoliciesConfigList, error) {
	endpoint := fmt.Sprintf(policiesResourceEndpoint, productID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	policies := &PoliciesConfigList{}
	err = handleJsonResp(resp, http.StatusOK, policies)
	return policies, err
}

// UpdatePolicies Update 3scale product policy chain
func (c *ThreeScaleClient) UpdatePolicies(productID int64, policies *PoliciesConfigList) (*PoliciesConfigList, error) {
	endpoint := fmt.Sprintf(policiesResourceEndpoint, productID)

	bodyArr, err := json.Marshal(policies)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bodyArr)
	req, err := c.buildUpdateJSONReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	items := &PoliciesConfigList{}
	err = handleJsonResp(resp, http.StatusOK, items)
	return items, err
}
