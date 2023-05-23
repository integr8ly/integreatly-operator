package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	backendListResourceEndpoint           = "/admin/api/backend_apis.json"
	backendResourceEndpoint               = "/admin/api/backend_apis/%d.json"
	backendMethodListResourceEndpoint     = "/admin/api/backend_apis/%d/metrics/%d/methods.json"
	backendMethodResourceEndpoint         = "/admin/api/backend_apis/%d/metrics/%d/methods/%d.json"
	backendMetricListResourceEndpoint     = "/admin/api/backend_apis/%d/metrics.json"
	backendMetricResourceEndpoint         = "/admin/api/backend_apis/%d/metrics/%d.json"
	backendMRListResourceEndpoint         = "/admin/api/backend_apis/%d/mapping_rules.json"
	backendMRResourceEndpoint             = "/admin/api/backend_apis/%d/mapping_rules/%d.json"
	backendUsageListResourceEndpoint      = "/admin/api/services/%d/backend_usages.json"
	backendUsageResourceEndpoint          = "/admin/api/services/%d/backend_usages/%d.json"
	BACKENDS_PER_PAGE                 int = 500
	BACKEND_METRICS_PER_PAGE          int = 500
	BACKEND_MAPPINGRULES_PER_PAGE     int = 500
)

// ListBackends List existing backends
func (c *ThreeScaleClient) ListBackendApis() (*BackendApiList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	backendList := &BackendApiList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		tmpBackendList, err := c.ListBackendApisPerPage(currentPage, BACKENDS_PER_PAGE)
		if err != nil {
			return nil, err
		}

		backendList.Backends = append(backendList.Backends, tmpBackendList.Backends...)

		allResultsPerPage = len(tmpBackendList.Backends) == BACKENDS_PER_PAGE
		currentPage += 1
	}

	return backendList, nil
}

// ListBackendApisPerPage List existing backends for a given page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListBackendApisPerPage(paginationValues ...int) (*BackendApiList, error) {
	queryValues := url.Values{}

	if len(paginationValues) > 0 {
		queryValues.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		queryValues.Add("per_page", strconv.Itoa(paginationValues[1]))
	}

	req, err := c.buildGetReq(backendListResourceEndpoint)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	backendList := &BackendApiList{}
	err = handleJsonResp(resp, http.StatusOK, backendList)
	return backendList, err
}

// CreateBackendApi Create 3scale Backend
func (c *ThreeScaleClient) CreateBackendApi(params Params) (*BackendApi, error) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(backendListResourceEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	backendApi := &BackendApi{}
	err = handleJsonResp(resp, http.StatusCreated, backendApi)
	return backendApi, err
}

// DeleteBackendApi Delete existing backend
func (c *ThreeScaleClient) DeleteBackendApi(id int64) error {
	backendEndpoint := fmt.Sprintf(backendResourceEndpoint, id)

	req, err := c.buildDeleteReq(backendEndpoint, nil)
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

// BackendApi Read 3scale Backend
func (c *ThreeScaleClient) BackendApi(id int64) (*BackendApi, error) {
	backendEndpoint := fmt.Sprintf(backendResourceEndpoint, id)

	req, err := c.buildGetReq(backendEndpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	backendAPI := &BackendApi{}
	err = handleJsonResp(resp, http.StatusOK, backendAPI)
	return backendAPI, err
}

// UpdateBackendApi Update 3scale Backend
func (c *ThreeScaleClient) UpdateBackendApi(id int64, params Params) (*BackendApi, error) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	backendEndpoint := fmt.Sprintf(backendResourceEndpoint, id)

	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(backendEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	backendAPI := &BackendApi{}
	err = handleJsonResp(resp, http.StatusOK, backendAPI)
	return backendAPI, err
}

// ListBackendapiMethods List existing backend methods
func (c *ThreeScaleClient) ListBackendapiMethods(backendapiID, hitsID int64) (*MethodList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	methodList := &MethodList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		tmpList, err := c.ListBackendapiMethodsPerPage(backendapiID, hitsID, currentPage, BACKEND_METRICS_PER_PAGE)
		if err != nil {
			return nil, err
		}

		methodList.Methods = append(methodList.Methods, tmpList.Methods...)

		allResultsPerPage = len(tmpList.Methods) == BACKEND_METRICS_PER_PAGE
		currentPage += 1
	}

	return methodList, nil
}

// ListBackendapiMethodsPerPage List existing backend methods for a given page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListBackendapiMethodsPerPage(backendapiID, hitsID int64, paginationValues ...int) (*MethodList, error) {
	queryValues := url.Values{}

	if len(paginationValues) > 0 {
		queryValues.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		queryValues.Add("per_page", strconv.Itoa(paginationValues[1]))
	}

	endpoint := fmt.Sprintf(backendMethodListResourceEndpoint, backendapiID, hitsID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := &MethodList{}
	err = handleJsonResp(resp, http.StatusOK, list)
	return list, err
}

// CreateBackendApiMethod Create 3scale Backend method
func (c *ThreeScaleClient) CreateBackendApiMethod(backendapiID, hitsID int64, params Params) (*Method, error) {
	endpoint := fmt.Sprintf(backendMethodListResourceEndpoint, backendapiID, hitsID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &Method{}
	err = handleJsonResp(resp, http.StatusCreated, item)
	return item, err
}

// DeleteBackendApiMethod Delete 3scale Backend method
func (c *ThreeScaleClient) DeleteBackendApiMethod(backendapiID, hitsID, methodID int64) error {
	endpoint := fmt.Sprintf(backendMethodResourceEndpoint, backendapiID, hitsID, methodID)

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

// BackendApiMethod Read 3scale Backend method
func (c *ThreeScaleClient) BackendApiMethod(backendapiID, hitsID, methodID int64) (*Method, error) {
	endpoint := fmt.Sprintf(backendMethodResourceEndpoint, backendapiID, hitsID, methodID)

	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &Method{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// UpdateBackendApiMethod Update 3scale Backend method
func (c *ThreeScaleClient) UpdateBackendApiMethod(backendapiID, hitsID, methodID int64, params Params) (*Method, error) {
	endpoint := fmt.Sprintf(backendMethodResourceEndpoint, backendapiID, hitsID, methodID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &Method{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// ListBackendapiMetrics List existing backend metric
func (c *ThreeScaleClient) ListBackendapiMetrics(backendapiID int64) (*MetricJSONList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	metricList := &MetricJSONList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		tmpList, err := c.ListBackendapiMetricsPerPage(backendapiID, currentPage, BACKEND_METRICS_PER_PAGE)
		if err != nil {
			return nil, err
		}

		metricList.Metrics = append(metricList.Metrics, tmpList.Metrics...)

		allResultsPerPage = len(tmpList.Metrics) == BACKEND_METRICS_PER_PAGE
		currentPage += 1
	}

	return metricList, nil
}

// ListBackendapiMetricsPerPage List existing backend metric for a given page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListBackendapiMetricsPerPage(backendapiID int64, paginationValues ...int) (*MetricJSONList, error) {
	queryValues := url.Values{}

	if len(paginationValues) > 0 {
		queryValues.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		queryValues.Add("per_page", strconv.Itoa(paginationValues[1]))
	}

	endpoint := fmt.Sprintf(backendMetricListResourceEndpoint, backendapiID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := &MetricJSONList{}
	err = handleJsonResp(resp, http.StatusOK, list)
	return list, err
}

// CreateBackendApiMetric Create 3scale Backend metric
func (c *ThreeScaleClient) CreateBackendApiMetric(backendapiID int64, params Params) (*MetricJSON, error) {
	endpoint := fmt.Sprintf(backendMetricListResourceEndpoint, backendapiID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &MetricJSON{}
	err = handleJsonResp(resp, http.StatusCreated, item)
	return item, err
}

// DeleteBackendApiMetric Delete 3scale Backend metric
func (c *ThreeScaleClient) DeleteBackendApiMetric(backendapiID, metricID int64) error {
	endpoint := fmt.Sprintf(backendMetricResourceEndpoint, backendapiID, metricID)

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

// BackendApiMetric Read 3scale Backend metric
func (c *ThreeScaleClient) BackendApiMetric(backendapiID, metricID int64) (*MetricJSON, error) {
	endpoint := fmt.Sprintf(backendMetricResourceEndpoint, backendapiID, metricID)

	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &MetricJSON{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// UpdateBackendApiMetric Update 3scale Backend metric
func (c *ThreeScaleClient) UpdateBackendApiMetric(backendapiID, metricID int64, params Params) (*MetricJSON, error) {
	endpoint := fmt.Sprintf(backendMetricResourceEndpoint, backendapiID, metricID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &MetricJSON{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

func (c *ThreeScaleClient) ListBackendapiMappingRules(backendapiID int64) (*MappingRuleJSONList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	mpList := &MappingRuleJSONList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		tmpList, err := c.ListBackendapiMappingRulesPerPage(backendapiID, currentPage, BACKEND_MAPPINGRULES_PER_PAGE)
		if err != nil {
			return nil, err
		}

		mpList.MappingRules = append(mpList.MappingRules, tmpList.MappingRules...)

		allResultsPerPage = len(tmpList.MappingRules) == BACKEND_MAPPINGRULES_PER_PAGE
		currentPage += 1
	}

	return mpList, nil
}

// ListBackendapiMappingRulesPerPage List existing backend mapping rules for a given page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListBackendapiMappingRulesPerPage(backendapiID int64, paginationValues ...int) (*MappingRuleJSONList, error) {
	queryValues := url.Values{}

	if len(paginationValues) > 0 {
		queryValues.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		queryValues.Add("per_page", strconv.Itoa(paginationValues[1]))
	}

	endpoint := fmt.Sprintf(backendMRListResourceEndpoint, backendapiID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := &MappingRuleJSONList{}
	err = handleJsonResp(resp, http.StatusOK, list)
	return list, err
}

// CreateBackendapiMappingRule Create 3scale Backend mappingrule
func (c *ThreeScaleClient) CreateBackendapiMappingRule(backendapiID int64, params Params) (*MappingRuleJSON, error) {
	endpoint := fmt.Sprintf(backendMRListResourceEndpoint, backendapiID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &MappingRuleJSON{}
	err = handleJsonResp(resp, http.StatusCreated, item)
	return item, err
}

// DeleteBackendapiMappingRule Delete 3scale Backend mapping rule
func (c *ThreeScaleClient) DeleteBackendapiMappingRule(backendapiID, mrID int64) error {
	endpoint := fmt.Sprintf(backendMRResourceEndpoint, backendapiID, mrID)

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

// BackendapiMappingRule Read 3scale Backend mapping rule
func (c *ThreeScaleClient) BackendapiMappingRule(backendapiID, mrID int64) (*MappingRuleJSON, error) {
	endpoint := fmt.Sprintf(backendMRResourceEndpoint, backendapiID, mrID)

	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &MappingRuleJSON{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// UpdateBackendapiMappingRule Update 3scale Backend mapping rule
func (c *ThreeScaleClient) UpdateBackendapiMappingRule(backendapiID, mrID int64, params Params) (*MappingRuleJSON, error) {
	endpoint := fmt.Sprintf(backendMRResourceEndpoint, backendapiID, mrID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &MappingRuleJSON{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// ListBackendapiUsages List existing backend usages for a given product
func (c *ThreeScaleClient) ListBackendapiUsages(productID int64) (BackendAPIUsageList, error) {
	endpoint := fmt.Sprintf(backendUsageListResourceEndpoint, productID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := BackendAPIUsageList{}
	err = handleJsonResp(resp, http.StatusOK, &list)
	return list, err
}

// CreateBackendapiUsage Create 3scale Backend usage
func (c *ThreeScaleClient) CreateBackendapiUsage(productID int64, params Params) (*BackendAPIUsage, error) {
	endpoint := fmt.Sprintf(backendUsageListResourceEndpoint, productID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &BackendAPIUsage{}
	err = handleJsonResp(resp, http.StatusCreated, item)
	return item, err
}

// DeleteBackendapiUsage Delete 3scale Backend usage
func (c *ThreeScaleClient) DeleteBackendapiUsage(productID, backendUsageID int64) error {
	endpoint := fmt.Sprintf(backendUsageResourceEndpoint, productID, backendUsageID)

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

// BackendapiUsage Read 3scale Backend usage
func (c *ThreeScaleClient) BackendapiUsage(productID, backendUsageID int64) (*BackendAPIUsage, error) {
	endpoint := fmt.Sprintf(backendUsageResourceEndpoint, productID, backendUsageID)

	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &BackendAPIUsage{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// UpdateBackendapiUsage Update 3scale Backend usage
func (c *ThreeScaleClient) UpdateBackendapiUsage(productID, backendUsageID int64, params Params) (*BackendAPIUsage, error) {
	endpoint := fmt.Sprintf(backendUsageResourceEndpoint, productID, backendUsageID)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(endpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &BackendAPIUsage{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}
