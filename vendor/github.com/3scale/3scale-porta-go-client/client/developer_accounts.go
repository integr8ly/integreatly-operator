package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	developerAccountListResourceEndpoint = "/admin/api/accounts.json"
	developerAccountResourceEndpoint     = "/admin/api/accounts/%d.json"
	signupResourceEndpoint               = "/admin/api/signup.json"
)

func (c *ThreeScaleClient) ListDeveloperAccounts() (*DeveloperAccountList, error) {
	req, err := c.buildGetReq(developerAccountListResourceEndpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	accountList := &DeveloperAccountList{}
	err = handleJsonResp(resp, http.StatusOK, accountList)
	return accountList, err
}

// Account fetches 3scale developer account
func (c *ThreeScaleClient) DeveloperAccount(accountID int64) (*DeveloperAccount, error) {
	endpoint := fmt.Sprintf(developerAccountResourceEndpoint, accountID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	obj := &DeveloperAccount{}
	err = handleJsonResp(resp, http.StatusOK, obj)
	return obj, err
}

// Signup will create an Account, an Admin User for the account, and optionally an Application with its keys.
// If the plan_id is not passed, the default plan will be used instead.
func (c *ThreeScaleClient) Signup(params Params) (*DeveloperAccount, error) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())

	req, err := c.buildPostReq(signupResourceEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperAccount{}
	err = handleJsonResp(resp, http.StatusCreated, respObj)
	return respObj, err
}

// UpdateDeveloperAccount Update existing developer account
func (c *ThreeScaleClient) UpdateDeveloperAccount(account *DeveloperAccount) (*DeveloperAccount, error) {
	if account == nil {
		return nil, errors.New("UpdateDeveloperAccount needs not nil pointer")
	}

	if account.Element.ID == nil {
		return nil, errors.New("UpdateDeveloperAccount needs not nil ID")
	}

	endpoint := fmt.Sprintf(developerAccountResourceEndpoint, *account.Element.ID)

	bodyArr, err := json.Marshal(account.Element)
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

	respObj := &DeveloperAccount{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// DeleteDeveloperAccount Delete existing developerAccount
func (c *ThreeScaleClient) DeleteDeveloperAccount(id int64) error {
	endpoint := fmt.Sprintf(developerAccountResourceEndpoint, id)

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
