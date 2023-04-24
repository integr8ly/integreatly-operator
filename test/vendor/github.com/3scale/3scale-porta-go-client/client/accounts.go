package client

import (
	"net/http"
	"net/url"
)

const (
	accountList = "/admin/api/accounts.json"
	findAccount = "/admin/api/accounts/find.json"
)

// Deprecated: Use ListDeveloperAccounts instead
func (c *ThreeScaleClient) ListAccounts() (*AccountList, error) {
	req, err := c.buildGetReq(accountList)
	if err != nil {
		return nil, err
	}

	urlValues := url.Values{}
	req.URL.RawQuery = urlValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	accountList := &AccountList{}
	err = handleJsonResp(resp, http.StatusOK, accountList)
	return accountList, err
}

func (c *ThreeScaleClient) FindAccount(username string) (*Account, error) {
	req, err := c.buildGetReq(findAccount)
	if err != nil {
		return nil, err
	}

	urlValues := url.Values{}
	urlValues.Add("username", username)
	req.URL.RawQuery = urlValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &AccountElem{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	return &apiResp.Account, err
}
