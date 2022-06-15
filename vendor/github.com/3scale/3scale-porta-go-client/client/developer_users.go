package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const (
	developerUserActivateEndpoint          = "/admin/api/accounts/%d/users/%d/activate.json"
	developerUserListResourceEndpoint      = "/admin/api/accounts/%d/users.json"
	developerUserResourceEndpoint          = "/admin/api/accounts/%d/users/%d.json"
	developerUserMemberResourceEndpoint    = "/admin/api/accounts/%d/users/%d/member.json"
	developerUserAdminResourceEndpoint     = "/admin/api/accounts/%d/users/%d/admin.json"
	developerUserSuspendResourceEndpoint   = "/admin/api/accounts/%d/users/%d/suspend.json"
	developerUserUnsuspendResourceEndpoint = "/admin/api/accounts/%d/users/%d/unsuspend.json"
)

func (c *ThreeScaleClient) ListDeveloperUsers(accountID int64, filterParams Params) (*DeveloperUserList, error) {
	endpoint := fmt.Sprintf(developerUserListResourceEndpoint, accountID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	for k, v := range filterParams {
		values.Add(k, v)
	}
	req.URL.RawQuery = values.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	userList := &DeveloperUserList{}
	err = handleJsonResp(resp, http.StatusOK, userList)
	return userList, err
}

func (c *ThreeScaleClient) DeveloperUser(accountID, userID int64) (*DeveloperUser, error) {
	endpoint := fmt.Sprintf(developerUserResourceEndpoint, accountID, userID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	obj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, obj)
	return obj, err
}

// UpdateDeveloperUser Update existing developer user
func (c *ThreeScaleClient) UpdateDeveloperUser(accountID int64, user *DeveloperUser) (*DeveloperUser, error) {
	if user == nil {
		return nil, errors.New("UpdateDeveloperUser requires not-nil DeveloperUser pointer")
	}

	if user.Element.ID == nil {
		return nil, errors.New("UpdateDeveloperUser requires not-nil DeveloperUser ID")
	}

	endpoint := fmt.Sprintf(developerUserResourceEndpoint, accountID, *user.Element.ID)

	bodyArr, err := json.Marshal(user.Element)
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

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// DeleteDeveloperUser Delete existing developerUser
func (c *ThreeScaleClient) DeleteDeveloperUser(accountID, userID int64) error {
	endpoint := fmt.Sprintf(developerUserResourceEndpoint, accountID, userID)

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

// ActivateDeveloperUser activates user of a given account from pending state to active
func (c *ThreeScaleClient) ActivateDeveloperUser(accountID, userID int64) (*DeveloperUser, error) {
	endpoint := fmt.Sprintf(developerUserActivateEndpoint, accountID, userID)

	req, err := c.buildUpdateJSONReq(endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// CreateDeveloperUser creates a new developer user
// username and email are unique fields for the entire provider account
// role attribute ["member", "admin"] cannot be set. All new developer users have member role
// state attribute ["pending", "active", "suspended"] cannot be set. All new developer users are in "pending" state
func (c *ThreeScaleClient) CreateDeveloperUser(accountID int64, user *DeveloperUser) (*DeveloperUser, error) {
	if user == nil {
		return nil, errors.New("CreateDeveloperUser requires not-nil DeveloperUser pointer")
	}

	endpoint := fmt.Sprintf(developerUserListResourceEndpoint, accountID)

	bodyArr, err := json.Marshal(user.Element)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bodyArr)

	req, err := c.buildPostJSONReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusCreated, respObj)
	return respObj, err
}

// ChangeRoleToMemberDeveloperUser sets user of a given account to member role
func (c *ThreeScaleClient) ChangeRoleToMemberDeveloperUser(accountID, userID int64) (*DeveloperUser, error) {
	endpoint := fmt.Sprintf(developerUserMemberResourceEndpoint, accountID, userID)

	req, err := c.buildUpdateJSONReq(endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// ChangeRoleToAdminDeveloperUser sets user of a given account to member role
func (c *ThreeScaleClient) ChangeRoleToAdminDeveloperUser(accountID, userID int64) (*DeveloperUser, error) {
	endpoint := fmt.Sprintf(developerUserAdminResourceEndpoint, accountID, userID)

	req, err := c.buildUpdateJSONReq(endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// SuspendDeveloperUser suspends the user of a given account
func (c *ThreeScaleClient) SuspendDeveloperUser(accountID, userID int64) (*DeveloperUser, error) {
	endpoint := fmt.Sprintf(developerUserSuspendResourceEndpoint, accountID, userID)

	req, err := c.buildUpdateJSONReq(endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}

// UnsuspendDeveloperUser unsuspends the user of a given account
func (c *ThreeScaleClient) UnsuspendDeveloperUser(accountID, userID int64) (*DeveloperUser, error) {
	endpoint := fmt.Sprintf(developerUserUnsuspendResourceEndpoint, accountID, userID)

	req, err := c.buildUpdateJSONReq(endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respObj := &DeveloperUser{}
	err = handleJsonResp(resp, http.StatusOK, respObj)
	return respObj, err
}
