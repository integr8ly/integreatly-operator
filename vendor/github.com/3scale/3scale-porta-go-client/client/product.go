package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	productListResourceEndpoint                = "/admin/api/services.json"
	productResourceEndpoint                    = "/admin/api/services/%d.json"
	productMethodListResourceEndpoint          = "/admin/api/services/%d/metrics/%d/methods.json"
	productMethodResourceEndpoint              = "/admin/api/services/%d/metrics/%d/methods/%d.json"
	productMetricListResourceEndpoint          = "/admin/api/services/%d/metrics.json"
	productMetricResourceEndpoint              = "/admin/api/services/%d/metrics/%d.json"
	productMappingRuleListResourceEndpoint     = "/admin/api/services/%d/proxy/mapping_rules.json"
	productMappingRuleResourceEndpoint         = "/admin/api/services/%d/proxy/mapping_rules/%d.json"
	productProxyResourceEndpoint               = "/admin/api/services/%d/proxy.json"
	productProxyDeployResourceEndpoint         = "/admin/api/services/%d/proxy/deploy.json"
	PRODUCTS_PER_PAGE                      int = 500
)

// BackendApi Read 3scale Backend
func (c *ThreeScaleClient) Product(id int64) (*Product, error) {
	endpoint := fmt.Sprintf(productResourceEndpoint, id)

	req, err := c.buildGetJSONReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	product := &Product{}
	err = handleJsonResp(resp, http.StatusOK, product)
	return product, err
}

// CreateProduct Create 3scale Product
func (c *ThreeScaleClient) CreateProduct(name string, params Params) (*Product, error) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	values.Add("name", name)

	body := strings.NewReader(values.Encode())
	req, err := c.buildPostReq(productListResourceEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	product := &Product{}
	err = handleJsonResp(resp, http.StatusCreated, product)
	return product, err
}

// UpdateProduct Update existing product
func (c *ThreeScaleClient) UpdateProduct(id int64, params Params) (*Product, error) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}

	putProductEndpoint := fmt.Sprintf(productResourceEndpoint, id)

	body := strings.NewReader(values.Encode())
	req, err := c.buildUpdateReq(putProductEndpoint, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	product := &Product{}
	err = handleJsonResp(resp, http.StatusOK, product)
	return product, err
}

// DeleteProduct Delete existing product
func (c *ThreeScaleClient) DeleteProduct(id int64) error {
	productEndpoint := fmt.Sprintf(productResourceEndpoint, id)

	req, err := c.buildDeleteReq(productEndpoint, nil)
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

func (c *ThreeScaleClient) ListProducts() (*ProductList, error) {
	// Keep asking until the results length is lower than "per_page" param
	currentPage := 1
	productList := &ProductList{}

	allResultsPerPage := false
	for next := true; next; next = allResultsPerPage {
		tmpProductList, err := c.ListProductsPerPage(currentPage, PRODUCTS_PER_PAGE)
		if err != nil {
			return nil, err
		}

		productList.Products = append(productList.Products, tmpProductList.Products...)

		allResultsPerPage = len(tmpProductList.Products) == PRODUCTS_PER_PAGE
		currentPage += 1
	}

	return productList, nil

}

// ListProductsPerPage List existing products in a single page
// paginationValues[0] = Page in the paginated list. Defaults to 1 for the API, as the client will not send the page param.
// paginationValues[1] = Number of results per page. Default and max is 500 for the aPI, as the client will not send the per_page param.
func (c *ThreeScaleClient) ListProductsPerPage(paginationValues ...int) (*ProductList, error) {
	queryValues := url.Values{}

	if len(paginationValues) > 0 {
		queryValues.Add("page", strconv.Itoa(paginationValues[0]))
	}

	if len(paginationValues) > 1 {
		queryValues.Add("per_page", strconv.Itoa(paginationValues[1]))
	}

	req, err := c.buildGetReq(productListResourceEndpoint)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	productList := &ProductList{}
	err = handleJsonResp(resp, http.StatusOK, productList)
	return productList, err
}

// ListProductMethods List existing product methods
func (c *ThreeScaleClient) ListProductMethods(productID, hitsID int64) (*MethodList, error) {
	endpoint := fmt.Sprintf(productMethodListResourceEndpoint, productID, hitsID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := &MethodList{}
	err = handleJsonResp(resp, http.StatusOK, list)
	return list, err
}

// CreateProductMethod Create 3scale product method
func (c *ThreeScaleClient) CreateProductMethod(productID, hitsID int64, params Params) (*Method, error) {
	endpoint := fmt.Sprintf(productMethodListResourceEndpoint, productID, hitsID)

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

// DeleteProductMethod Delete 3scale product method
func (c *ThreeScaleClient) DeleteProductMethod(productID, hitsID, methodID int64) error {
	endpoint := fmt.Sprintf(productMethodResourceEndpoint, productID, hitsID, methodID)

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

// ProductMethod Read 3scale product method
func (c *ThreeScaleClient) ProductMethod(productID, hitsID, methodID int64) (*Method, error) {
	endpoint := fmt.Sprintf(productMethodResourceEndpoint, productID, hitsID, methodID)

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

// UpdateProductMethod Update 3scale product method
func (c *ThreeScaleClient) UpdateProductMethod(productID, hitsID, methodID int64, params Params) (*Method, error) {
	endpoint := fmt.Sprintf(productMethodResourceEndpoint, productID, hitsID, methodID)

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

// ListProductMetrics List existing product metrics
func (c *ThreeScaleClient) ListProductMetrics(productID int64) (*MetricJSONList, error) {
	endpoint := fmt.Sprintf(productMetricListResourceEndpoint, productID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := &MetricJSONList{}
	err = handleJsonResp(resp, http.StatusOK, list)
	return list, err
}

// CreateProductMetric Create 3scale product metric
func (c *ThreeScaleClient) CreateProductMetric(productID int64, params Params) (*MetricJSON, error) {
	endpoint := fmt.Sprintf(productMetricListResourceEndpoint, productID)

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

// DeleteProductMetric Delete 3scale product metric
func (c *ThreeScaleClient) DeleteProductMetric(productID, metricID int64) error {
	endpoint := fmt.Sprintf(productMetricResourceEndpoint, productID, metricID)

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

// ProductMetric Read 3scale product metric
func (c *ThreeScaleClient) ProductMetric(productID, metricID int64) (*MetricJSON, error) {
	endpoint := fmt.Sprintf(productMetricResourceEndpoint, productID, metricID)

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

// UpdateProductMetric Update 3scale product metric
func (c *ThreeScaleClient) UpdateProductMetric(productID, metricID int64, params Params) (*MetricJSON, error) {
	endpoint := fmt.Sprintf(productMetricResourceEndpoint, productID, metricID)

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

// ListProductMappingRules List existing product mappingrules
func (c *ThreeScaleClient) ListProductMappingRules(productID int64) (*MappingRuleJSONList, error) {
	endpoint := fmt.Sprintf(productMappingRuleListResourceEndpoint, productID)
	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := &MappingRuleJSONList{}
	err = handleJsonResp(resp, http.StatusOK, list)
	return list, err
}

// CreateProductMappingRule Create 3scale product mappingrule
func (c *ThreeScaleClient) CreateProductMappingRule(productID int64, params Params) (*MappingRuleJSON, error) {
	endpoint := fmt.Sprintf(productMappingRuleListResourceEndpoint, productID)

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

// DeleteProductMappingRule Delete 3scale product mappingrule
func (c *ThreeScaleClient) DeleteProductMappingRule(productID, itemID int64) error {
	endpoint := fmt.Sprintf(productMappingRuleResourceEndpoint, productID, itemID)

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

// ProductMappingRule Read 3scale product mappingrule
func (c *ThreeScaleClient) ProductMappingRule(productID, itemID int64) (*MappingRuleJSON, error) {
	endpoint := fmt.Sprintf(productMappingRuleResourceEndpoint, productID, itemID)

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

// UpdateProductMappingRule Update 3scale product mappingrule
func (c *ThreeScaleClient) UpdateProductMappingRule(productID, itemID int64, params Params) (*MappingRuleJSON, error) {
	endpoint := fmt.Sprintf(productMappingRuleResourceEndpoint, productID, itemID)

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

// ProductProxy Read 3scale product proxy
func (c *ThreeScaleClient) ProductProxy(productID int64) (*ProxyJSON, error) {
	endpoint := fmt.Sprintf(productProxyResourceEndpoint, productID)

	req, err := c.buildGetReq(endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &ProxyJSON{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// UpdateProductProxy Update 3scale product mappingrule
func (c *ThreeScaleClient) UpdateProductProxy(productID int64, params Params) (*ProxyJSON, error) {
	endpoint := fmt.Sprintf(productProxyResourceEndpoint, productID)

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

	item := &ProxyJSON{}
	err = handleJsonResp(resp, http.StatusOK, item)
	return item, err
}

// ProductProxyDeploy Promotes proxy configuration to staging
func (c *ThreeScaleClient) DeployProductProxy(productID int64) (*ProxyJSON, error) {
	endpoint := fmt.Sprintf(productProxyDeployResourceEndpoint, productID)

	req, err := c.buildPostReq(endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	item := &ProxyJSON{}
	err = handleJsonResp(resp, http.StatusCreated, item)
	return item, err
}
