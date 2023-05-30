package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	proxyGetUpdate            = "/admin/api/services/%s/proxy.xml"
	proxyConfigGet            = "/admin/api/services/%s/proxy/configs/%s/%s.json"
	proxyConfigList           = "/admin/api/services/%s/proxy/configs/%s.json"
	proxyConfigLatestGet      = "/admin/api/services/%s/proxy/configs/%s/latest.json"
	proxyConfigPromote        = "/admin/api/services/%s/proxy/configs/%s/%s/promote.json"
	accountProxyConfigGet     = "/admin/api/account/proxy_configs/%s.json"
	PROXYCONFIGS_PER_PAGE int = 500
)

// ReadProxy - Returns the Proxy for a specific Service.
// Deprecated - Use ProductProxy function instead
func (c *ThreeScaleClient) ReadProxy(svcID string) (Proxy, error) {
	var p Proxy

	endpoint := fmt.Sprintf(proxyGetUpdate, svcID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return p, httpReqError
	}

	values := url.Values{}
	req.URL.RawQuery = values.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()

	err = handleXMLResp(resp, http.StatusOK, &p)
	return p, err
}

// GetProxyConfig - Returns the Proxy Configs of a Service
// Supports invoking client callback upon response from 3scale
func (c *ThreeScaleClient) GetProxyConfig(svcId string, env string, version string) (ProxyConfigElement, error) {
	endpoint := fmt.Sprintf(proxyConfigGet, svcId, env, version)
	return c.getProxyConfig(endpoint)
}

// GetLatestProxyConfig - Returns the latest Proxy Config
// Supports invoking client callback upon response from 3scale
func (c *ThreeScaleClient) GetLatestProxyConfig(svcId string, env string) (ProxyConfigElement, error) {
	endpoint := fmt.Sprintf(proxyConfigLatestGet, svcId, env)
	return c.getProxyConfig(endpoint)
}

// UpdateProxy - Changes the Proxy settings.
// This will create a new APIcast configuration version for the Staging environment with the updated settings.
func (c *ThreeScaleClient) UpdateProxy(svcId string, params Params) (Proxy, error) {
	var p Proxy

	endpoint := fmt.Sprintf(proxyGetUpdate, svcId)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(endpoint, body)
	if err != nil {
		return p, httpReqError
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return p, err
	}

	defer resp.Body.Close()

	err = handleXMLResp(resp, http.StatusOK, &p)
	return p, err
}

// ListProxyConfig - Returns the Proxy Configs of a Service
// env parameter should be one of 'sandbox', 'production'
func (c *ThreeScaleClient) ListProxyConfig(svcId string, env string) (ProxyConfigList, error) {
	var pc ProxyConfigList

	endpoint := fmt.Sprintf(proxyConfigList, svcId, env)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return pc, httpReqError
	}
	req.Header.Set("Accept", "application/json")

	values := url.Values{}
	req.URL.RawQuery = values.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pc, err
	}

	defer resp.Body.Close()

	err = handleJsonResp(resp, http.StatusOK, &pc)
	return pc, err
}

// PromoteProxyConfig - Promotes a Proxy Config from one environment to another environment.
func (c *ThreeScaleClient) PromoteProxyConfig(svcId string, env string, version string, toEnv string) (ProxyConfigElement, error) {
	var pe ProxyConfigElement
	endpoint := fmt.Sprintf(proxyConfigPromote, svcId, env, version)

	values := url.Values{}
	values.Add("to", toEnv)

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(endpoint, body)
	if err != nil {
		return pe, httpReqError
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pe, err
	}

	defer resp.Body.Close()

	err = handleJsonResp(resp, http.StatusCreated, &pe)
	return pe, err
}

func (c *ThreeScaleClient) getProxyConfig(endpoint string) (ProxyConfigElement, error) {
	var pc ProxyConfigElement
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return pc, httpReqError
	}

	values := url.Values{}
	req.URL.RawQuery = values.Encode()
	req.Header.Set("accept", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pc, err
	}
	timeTaken := time.Since(start)
	if c.afterResponse != nil {
		c.afterResponse(resp.StatusCode, timeTaken)
	}
	defer resp.Body.Close()

	err = handleJsonResp(resp, http.StatusOK, &pc)
	return pc, err
}

// ListProxyConfig - Returns the Proxy Configs of a Service
// env parameter should be one of 'sandbox', 'production'
func (c *ThreeScaleClient) ListAccountProxyConfigs(env string, version, host *string) (*ProxyConfigList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	configList := &ProxyConfigList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		pageList, err := c.ListAccountProxyConfigsPerPage(env, version, host, currentPage, PROXYCONFIGS_PER_PAGE)
		if err != nil {
			return nil, err
		}

		configList.ProxyConfigs = append(configList.ProxyConfigs, pageList.ProxyConfigs...)

		allResultsPerPage = len(pageList.ProxyConfigs) == PROXYCONFIGS_PER_PAGE
		currentPage += 1
	}

	return configList, nil
}

// ListAccountProxyConfigsPerPage List existing proxy configs in a single page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListAccountProxyConfigsPerPage(env string, version, host *string, paginationValues ...int) (*ProxyConfigList, error) {
	var pc ProxyConfigList

	endpoint := fmt.Sprintf(accountProxyConfigGet, env)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, httpReqError
	}
	req.Header.Set("Accept", "application/json")

	values := url.Values{}
	if version != nil {
		values.Add("version", *version)
	}
	if host != nil {
		values.Add("host", *host)
	}
	if len(paginationValues) > 0 {
		values.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		values.Add("per_page", strconv.Itoa(paginationValues[1]))
	}
	req.URL.RawQuery = values.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = handleJsonResp(resp, http.StatusOK, &pc)
	if err != nil {
		return nil, err
	}

	return &pc, nil
}
