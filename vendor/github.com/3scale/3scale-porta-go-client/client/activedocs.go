package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	activeDocListEndpoint = "/admin/api/active_docs.json"
	activeDocEndpoint     = "/admin/api/active_docs/%d.json"
)

// ListActiveDocs List existing activedocs for the client provider account
func (c *ThreeScaleClient) ListActiveDocs() (*ActiveDocList, error) {
	req, err := c.buildGetReq(activeDocListEndpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	activeDocList := &ActiveDocList{}
	err = handleJsonResp(resp, http.StatusOK, activeDocList)
	return activeDocList, err
}

// ActiveDoc Reads 3scale Activedoc
func (c *ThreeScaleClient) ActiveDoc(id int64) (*ActiveDoc, error) {
	endpoint := fmt.Sprintf(activeDocEndpoint, id)

	req, err := c.buildGetJSONReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	activeDoc := &ActiveDoc{}
	err = handleJsonResp(resp, http.StatusOK, activeDoc)
	return activeDoc, err
}

// CreateActiveDoc Create 3scale activedoc
func (c *ThreeScaleClient) CreateActiveDoc(activeDoc *ActiveDoc) (*ActiveDoc, error) {
	bodyArr, err := json.Marshal(activeDoc.Element)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bodyArr)

	req, err := c.buildPostJSONReq(activeDocListEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &ActiveDoc{}
	err = handleJsonResp(resp, http.StatusCreated, respObj)
	return respObj, err
}

// UpdateActiveDoc Update existing activedoc
func (c *ThreeScaleClient) UpdateActiveDoc(activeDoc *ActiveDoc) (*ActiveDoc, error) {
	if activeDoc == nil {
		return nil, errors.New("UpdateActiveDoc needs not nil pointer")
	}

	if activeDoc.Element.ID == nil {
		return nil, errors.New("UpdateActiveDoc needs not nil ID")
	}

	endpoint := fmt.Sprintf(activeDocEndpoint, *activeDoc.Element.ID)

	bodyArr, err := json.Marshal(activeDoc.Element)
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

	respObj := &ActiveDoc{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// DeleteActiveDoc Delete existing activedoc
func (c *ThreeScaleClient) DeleteActiveDoc(id int64) error {
	endpoint := fmt.Sprintf(activeDocEndpoint, id)

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

// UnbindActiveDocFromProduct removes product relationship from activedoc object
func (c *ThreeScaleClient) UnbindActiveDocFromProduct(id int64) (*ActiveDoc, error) {
	endpoint := fmt.Sprintf(activeDocEndpoint, id)

	data := struct {
		ID        int64  `json:"id"`
		ServiceID *int64 `json:"service_id"`
	}{
		id,
		nil,
	}

	bodyArr, err := json.Marshal(data)
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

	respObj := &ActiveDoc{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}
