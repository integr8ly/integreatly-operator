package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	apicastPolicyRegistryEndpoint = "/admin/api/registry/policies.json"
	apicastPolicyEndpoint         = "/admin/api/registry/policies/%d.json"
)

// ListAPIcastPolicies List existing apicast policies in the registry for the client provider account
func (c *ThreeScaleClient) ListAPIcastPolicies() (*APIcastPolicyRegistry, error) {
	req, err := c.buildGetReq(apicastPolicyRegistryEndpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	obj := &APIcastPolicyRegistry{}
	err = handleJsonResp(resp, http.StatusOK, obj)
	return obj, err
}

// ReadAPIcastPolicy Reads 3scale apicast policy from registry
func (c *ThreeScaleClient) ReadAPIcastPolicy(id int64) (*APIcastPolicy, error) {
	endpoint := fmt.Sprintf(apicastPolicyEndpoint, id)

	req, err := c.buildGetJSONReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	obj := &APIcastPolicy{}
	err = handleJsonResp(resp, http.StatusOK, obj)
	return obj, err
}

// CreateAPIcastPolicy Create 3scale apicast policy in the registry
func (c *ThreeScaleClient) CreateAPIcastPolicy(item *APIcastPolicy) (*APIcastPolicy, error) {
	bodyArr, err := json.Marshal(item.Element)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bodyArr)

	req, err := c.buildPostJSONReq(apicastPolicyRegistryEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &APIcastPolicy{}
	err = handleJsonResp(resp, http.StatusCreated, respObj)
	return respObj, err
}

// UpdateAPIcastPolicy Update existing apicast policy in the registry
func (c *ThreeScaleClient) UpdateAPIcastPolicy(item *APIcastPolicy) (*APIcastPolicy, error) {
	if item == nil {
		return nil, errors.New("UpdateAPIcastPolicy needs not nil pointer")
	}

	if item.Element.ID == nil {
		return nil, errors.New("UpdateAPIcastPolicy needs not nil ID")
	}

	endpoint := fmt.Sprintf(apicastPolicyEndpoint, *item.Element.ID)

	bodyArr, err := json.Marshal(item.Element)
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

	respObj := &APIcastPolicy{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// DeleteAPIcastPolicy Delete existing apicast policy in the registry
func (c *ThreeScaleClient) DeleteAPIcastPolicy(id int64) error {
	endpoint := fmt.Sprintf(apicastPolicyEndpoint, id)

	req, err := c.buildDeleteReq(endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handleJsonResp(resp, http.StatusOK, nil)
}
