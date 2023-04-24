package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	appRead                    = "/admin/api/accounts/%d/applications/%d.json"
	appCreate                  = "/admin/api/accounts/%s/applications.json"
	appList                    = "/admin/api/accounts/%d/applications.json"
	appUpdate                  = "/admin/api/accounts/%d/applications/%d.json"
	appDelete                  = "/admin/api/accounts/%d/applications/%d.json"
	appChangePlan              = "/admin/api/accounts/%d/applications/%d/change_plan.json"
	appCreatePlanCustomization = "/admin/api/accounts/%d/applications/%d/customize_plan.json"
	appDeletePlanCustomization = "/admin/api/accounts/%d/applications/%d/decustomize_plan.json"
	appSuspend                 = "/admin/api/accounts/%d/applications/%d/suspend.json"
	appResume                  = "/admin/api/accounts/%d/applications/%d/resume.json"
	listAllApplications        = "/admin/api/applications.json"
)

// CreateApp - Create an application.
// The application object can be extended with Fields Definitions in the Admin Portal where you can add/remove fields
func (c *ThreeScaleClient) CreateApp(accountId, planId, name, description string) (Application, error) {
	var app Application
	endpoint := fmt.Sprintf(appCreate, accountId)

	values := url.Values{}
	values.Add("account_id", accountId)
	values.Add("plan_id", planId)
	values.Add("name", name)
	values.Add("description", description)

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(endpoint, body)
	if err != nil {
		return app, httpReqError
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return app, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationElem{}
	err = handleJsonResp(resp, http.StatusCreated, apiResp)
	if err != nil {
		return app, err
	}
	return apiResp.Application, nil
}

// ListApplications - List of applications for a given account.
func (c *ThreeScaleClient) ListApplications(accountID int64) (*ApplicationList, error) {
	endpoint := fmt.Sprintf(appList, accountID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	applicationList := &ApplicationList{}
	err = handleJsonResp(resp, http.StatusOK, applicationList)
	return applicationList, err
}

// DeleteApplication Delete existing application
func (c *ThreeScaleClient) DeleteApplication(accountID, id int64) error {
	applicationEndpoint := fmt.Sprintf(appDelete, accountID, id)

	req, err := c.buildDeleteReq(applicationEndpoint, nil)
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

func (c *ThreeScaleClient) UpdateApplication(accountID, id int64, params Params) (*Application, error) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	applicationEndpoint := fmt.Sprintf(appUpdate, accountID, id)

	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(applicationEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationElem{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	return &apiResp.Application, err
}

func (c *ThreeScaleClient) ChangeApplicationPlan(accountID, id, planId int64) (*Application, error) {
	values := url.Values{}
	values.Add("plan_id", strconv.FormatInt(planId, 10))

	applicationEndpoint := fmt.Sprintf(appChangePlan, accountID, id)

	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(applicationEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationElem{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	return &apiResp.Application, err
}

func (c *ThreeScaleClient) CreateApplicationCustomPlan(accountId, id int64) (*ApplicationPlanItem, error) {
	endpoint := fmt.Sprintf(appCreatePlanCustomization, accountId, id)

	req, err := c.buildUpdateReq(endpoint, nil)
	if err != nil {
		return nil, httpReqError
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationPlan{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	if err != nil {
		return nil, err
	}
	return &apiResp.Element, nil
}

func (c *ThreeScaleClient) DeleteApplicationCustomPlan(accountID, id int64) error {
	applicationEndpoint := fmt.Sprintf(appDeletePlanCustomization, accountID, id)

	req, err := c.buildUpdateReq(applicationEndpoint, nil)
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

func (c *ThreeScaleClient) ApplicationSuspend(accountId, id int64) (*Application, error) {
	endpoint := fmt.Sprintf(appSuspend, accountId, id)

	req, err := c.buildUpdateReq(endpoint, nil)
	if err != nil {
		return nil, httpReqError
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationElem{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	if err != nil {
		return nil, err
	}
	return &apiResp.Application, nil
}

func (c *ThreeScaleClient) ApplicationResume(accountId, id int64) (*Application, error) {
	endpoint := fmt.Sprintf(appResume, accountId, id)

	req, err := c.buildUpdateReq(endpoint, nil)
	if err != nil {
		return nil, httpReqError
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationElem{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	if err != nil {
		return nil, err
	}
	return &apiResp.Application, nil
}

func (c *ThreeScaleClient) Application(accountId, id int64) (*Application, error) {
	endpoint := fmt.Sprintf(appRead, accountId, id)

	req, err := c.buildGetJSONReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationElem{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	return &apiResp.Application, err
}

func (c *ThreeScaleClient) ListAllApplications() (*ApplicationList, error) {
	endpoint := fmt.Sprintf(listAllApplications)

	req, err := c.buildGetJSONReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResp := &ApplicationList{}
	err = handleJsonResp(resp, http.StatusOK, apiResp)
	return apiResp, err
}
