package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	developerAccountListResourceEndpoint     = "/admin/api/accounts.json"
	developerAccountResourceEndpoint         = "/admin/api/accounts/%d.json"
	signupResourceEndpoint                   = "/admin/api/signup.json"
	DEVELOPERACCOUNTS_PER_PAGE           int = 500
)

func (c *ThreeScaleClient) ListDeveloperAccounts() (*DeveloperAccountList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	list := &DeveloperAccountList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		tmpList, err := c.ListDeveloperAccountsPerPage(currentPage, DEVELOPERACCOUNTS_PER_PAGE)
		if err != nil {
			return nil, err
		}

		list.Items = append(list.Items, tmpList.Items...)

		allResultsPerPage = len(tmpList.Items) == DEVELOPERACCOUNTS_PER_PAGE
		currentPage += 1
	}

	return list, nil
}

// ListDeveloperAccountsPerPage List existing developer accounts for a given page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListDeveloperAccountsPerPage(paginationValues ...int) (*DeveloperAccountList, error) {
	queryValues := url.Values{}

	if len(paginationValues) > 0 {
		queryValues.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		queryValues.Add("per_page", strconv.Itoa(paginationValues[1]))
	}

	req, err := c.buildGetReq(developerAccountListResourceEndpoint)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = queryValues.Encode()

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
