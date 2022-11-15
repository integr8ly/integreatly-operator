package threescale

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/sirupsen/logrus"
)

//go:generate moq -out three_scale_moq.go . ThreeScaleInterface
type ThreeScaleInterface interface {
	SetNamespace(ns string)
	AddAuthenticationProvider(data map[string]string, accessToken string) (*http.Response, error)
	GetAuthenticationProviders(accessToken string) (*AuthProviders, error)
	GetAuthenticationProviderByName(name string, accessToken string) (*AuthProvider, error)
	GetUser(username, accessToken string) (*User, error)
	GetUsers(accessToken string) (*Users, error)
	AddUser(username string, email string, password string, accessToken string) (*http.Response, error)
	DeleteUser(userID int, accessToken string) (*http.Response, error)
	SetUserAsAdmin(userID int, accessToken string) (*http.Response, error)
	SetUserAsMember(userID int, accessToken string) (*http.Response, error)
	SetFromEmailAddress(emailAddress string, accessToken string) (*http.Response, error)
	UpdateUser(userID int, username string, email string, accessToken string) (*http.Response, error)

	CreateAccount(accessToken, orgName, username string) (string, error)
	CreateBackend(accessToken, name, privateEndpoint string) (int, error)
	CreateMetric(accessToken string, backendID int, friendlyName, unit string) (int, error)
	CreateBackendMappingRule(accessToken string, backendID, metricID int, httpMethod, pattern string, delta int) error
	CreateService(accessToken, name, systemName string) (string, error)
	CreateBackendUsage(accessToken, serviceID string, backendID int, path string) error
	CreateApplicationPlan(accessToken, serviceID, name string) (string, error)
	CreateApplication(accessToken, accountID, planID, name, description string) (string, error)
	DeployProxy(accessToken, serviceID string) error
	PromoteProxy(accessToken, serviceID, env, to string) (string, error)

	DeleteService(accessToken, serviceID string) error
	DeleteBackend(accessToken string, backendID int) error
	DeleteAccount(accessToken, accountID string) error

	CreateTenant(accessToken string, account AccountDetail, password string, email string) (*SignUpAccount, error)
	ListTenantAccounts(accessToken string, page int) ([]AccountDetail, error)
	GetTenantAccount(accessToken string, id int) (*SignUpAccount, error)
	DeleteTenant(accessToken string, id int) error
	DeleteTenants(accessToken string, accounts []AccountDetail) error

	ActivateUser(accessToken string, accountId, userId int) error
	AddAuthProviderToAccount(accessToken string, account AccountDetail, authProviderDetail AuthProviderDetails) error
	IsAuthProviderAdded(accessToken string, authProviderName string, account AccountDetail) (bool, error)
}

const (
	adminRole  = "admin"
	memberRole = "member"
)

type threeScaleClient struct {
	httpc          *http.Client
	wildCardDomain string
	ns             string
}

var _ ThreeScaleInterface = &threeScaleClient{}

func NewThreeScaleClient(httpc *http.Client, wildCardDomain string) *threeScaleClient {

	return &threeScaleClient{
		httpc:          httpc,
		wildCardDomain: wildCardDomain,
	}
}

func (tsc *threeScaleClient) SetNamespace(ns string) {
	tsc.ns = ns
}

func (tsc *threeScaleClient) AddAuthenticationProvider(data map[string]string, accessToken string) (*http.Response, error) {
	data["access_token"] = accessToken
	reqData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	tsc.httpc.Timeout = time.Second * 10
	res, err := tsc.httpc.Post(
		fmt.Sprintf("https://3scale-admin.%s/admin/api/account/authentication_providers.json", tsc.wildCardDomain),
		"application/json",
		bytes.NewBuffer(reqData),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (tsc *threeScaleClient) GetAuthenticationProviders(accessToken string) (*AuthProviders, error) {
	res, err := tsc.httpc.Get(
		fmt.Sprintf("https://3scale-admin.%s/admin/api/account/authentication_providers.json?access_token=%s", tsc.wildCardDomain, accessToken),
	)
	if err != nil {
		return nil, err
	}

	authProviders := &AuthProviders{}
	err = json.NewDecoder(res.Body).Decode(authProviders)
	if err != nil {
		return nil, err
	}

	return authProviders, nil
}

func (tsc *threeScaleClient) GetAuthenticationProviderByName(name string, accessToken string) (*AuthProvider, error) {
	authProviders, err := tsc.GetAuthenticationProviders(accessToken)
	if err != nil {
		return nil, err
	}

	for _, ap := range authProviders.AuthProviders {
		if ap.ProviderDetails.Name == name {
			return ap, nil
		}
	}

	return nil, &tsError{message: "Authprovider not found", StatusCode: http.StatusNotFound}
}

func (tsc *threeScaleClient) GetUser(username, accessToken string) (*User, error) {
	users, err := tsc.GetUsers(accessToken)
	if err != nil {
		return nil, err
	}

	for _, u := range users.Users {
		if u.UserDetails.Username == username {
			return u, nil
		}
	}

	return nil, &tsError{message: "User not found", StatusCode: http.StatusNotFound}
}

func (tsc *threeScaleClient) GetUsers(accessToken string) (*Users, error) {
	res, err := tsc.httpc.Get(
		fmt.Sprintf("https://3scale-admin.%s/admin/api/users.json?access_token=%s", tsc.wildCardDomain, accessToken),
	)
	if err != nil {
		return nil, err
	}

	users := &Users{}
	err = json.NewDecoder(res.Body).Decode(users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (tsc *threeScaleClient) SetFromEmailAddress(emailAddress string, accessToken string) (*http.Response, error) {

	//curl -v --header "Content-Type: application/json" -X PUT "https://3scale-admin.apps.bg.o7rx.s1.devshift.org/admin/api/provider.xml"
	// --data '{"access_token":"05807975eb3cbec201d16fc54a327546960fc61bb278169e28eafdb99913bbbe","from_email":"test@test.com"}'

	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
		"from_email":   emailAddress,
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/provider.xml", tsc.wildCardDomain)
	req, err := http.NewRequest(
		"PUT",
		url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	tsc.httpc.Timeout = time.Second * 10

	res, err := tsc.httpc.Do(req)

	if err == nil && res.StatusCode != 200 {

		err = errors.New(fmt.Sprintf("StatusCode %v calling SetFromEmailAddress", res.StatusCode))
	}

	return res, err
}

func (tsc *threeScaleClient) AddUser(username string, email string, password string, accessToken string) (*http.Response, error) {
	data := make(map[string]string)
	data["access_token"] = accessToken
	data["username"] = username
	data["email"] = email
	data["password"] = password
	reqData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	res, err := tsc.httpc.Post(
		fmt.Sprintf("https://3scale-admin.%s/admin/api/users.json", tsc.wildCardDomain),
		"application/json",
		bytes.NewBuffer(reqData),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (tsc *threeScaleClient) DeleteUser(userID int, accessToken string) (*http.Response, error) {
	data := make(map[string]string)
	data["access_token"] = accessToken
	reqData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d.json", tsc.wildCardDomain, userID),
		bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-type", "application/json")
	tsc.httpc.Timeout = time.Second * 10
	res, err := tsc.httpc.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (tsc *threeScaleClient) SetUserAsAdmin(userID int, accessToken string) (*http.Response, error) {
	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d/admin.json", tsc.wildCardDomain, userID)
	req, err := http.NewRequest(
		"PUT",
		url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	tsc.httpc.Timeout = time.Second * 10

	res, err := tsc.httpc.Do(req)

	return res, err
}

func (tsc *threeScaleClient) SetUserAsMember(userID int, accessToken string) (*http.Response, error) {
	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d/member.json", tsc.wildCardDomain, userID)
	req, err := http.NewRequest(
		"PUT",
		url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	tsc.httpc.Timeout = time.Second * 10
	res, err := tsc.httpc.Do(req)

	return res, err
}

func (tsc *threeScaleClient) UpdateUser(userID int, username string, email string, accessToken string) (*http.Response, error) {
	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
		"username":     username,
		"email":        email,
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d.json", tsc.wildCardDomain, userID)
	req, err := http.NewRequest(
		"PUT",
		url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	tsc.httpc.Timeout = time.Second * 10
	res, err := tsc.httpc.Do(req)

	return res, err
}

func (tsc *threeScaleClient) CreateAccount(accessToken, orgName, username string) (string, error) {
	data := map[string]interface{}{
		"org_name": orgName,
		"username": username,
	}

	res, err := tsc.makeRequest("POST", "signup.xml", withAccessToken(accessToken, data))
	if err != nil {
		return "", err
	}

	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return "", err
	}

	return xmlFromResponse(res, "//account/id/text()")
}

func (tsc *threeScaleClient) CreateBackend(accessToken, name, privateEndpoint string) (int, error) {
	data := map[string]interface{}{
		"name":             name,
		"private_endpoint": privateEndpoint,
	}

	res, err := tsc.makeRequest("POST", "backend_apis.json", withAccessToken(accessToken, data))
	if err != nil {
		return 0, err
	}

	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return 0, err
	}

	responseBody := &struct {
		BackendAPI struct {
			ID int `json:"id"`
		} `json:"backend_api"`
	}{}
	if err := jsonFromResponse(res, responseBody); err != nil {
		return 0, err
	}

	return responseBody.BackendAPI.ID, nil
}

func (tsc *threeScaleClient) CreateMetric(accessToken string, backendID int, friendlyName, unit string) (int, error) {
	res, err := tsc.makeRequest(
		"POST",
		fmt.Sprintf("backend_apis/%d/metrics.json", backendID),
		withAccessToken(accessToken, map[string]interface{}{
			"friendly_name": friendlyName,
			"unit":          unit,
		}),
	)
	if err != nil {
		return 0, err
	}
	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return 0, err
	}

	responseBody := &struct {
		Metric struct {
			ID int `json:"id"`
		} `json:"metric"`
	}{}
	if err := jsonFromResponse(res, responseBody); err != nil {
		return 0, err
	}

	return responseBody.Metric.ID, nil
}

func (tsc *threeScaleClient) CreateBackendMappingRule(accessToken string, backendID, metricID int, httpMethod, pattern string, delta int) error {
	res, err := tsc.makeRequest(
		"POST",
		fmt.Sprintf("backend_apis/%d/mapping_rules.json", backendID),
		withAccessToken(accessToken, map[string]interface{}{
			"http_method": httpMethod,
			"pattern":     pattern,
			"delta":       delta,
			"metric_id":   metricID,
		}),
	)
	if err != nil {
		return err
	}

	return assertStatusCode(http.StatusCreated, res)
}

func (tsc *threeScaleClient) CreateService(accessToken, name, systemName string) (string, error) {
	res, err := tsc.makeRequest(
		"POST",
		"services.xml",
		withAccessToken(accessToken, map[string]interface{}{
			"name":        name,
			"system_name": name,
		}),
	)
	if err != nil {
		return "", err
	}
	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return "", err
	}

	return xmlFromResponse(res, "//service/id/text()")
}

func (tsc *threeScaleClient) CreateBackendUsage(accessToken, serviceID string, backendID int, path string) error {
	res, err := tsc.makeRequest(
		"POST",
		fmt.Sprintf("services/%s/backend_usages.json", serviceID),
		withAccessToken(accessToken, map[string]interface{}{
			"backend_api_id": backendID,
			"path":           path,
		}),
	)

	if err != nil {
		return err
	}
	return assertStatusCode(http.StatusCreated, res)
}

func (tsc *threeScaleClient) CreateApplicationPlan(accessToken, serviceID, name string) (string, error) {
	res, err := tsc.makeRequest(
		"POST",
		fmt.Sprintf("services/%s/application_plans.xml", serviceID),
		withAccessToken(accessToken, map[string]interface{}{
			"name": name,
		}),
	)
	if err != nil {
		return "", err
	}

	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return "", err
	}

	return xmlFromResponse(res, "//plan/id/text()")
}

func (tsc *threeScaleClient) CreateApplication(accessToken, accountID, planID, name, description string) (string, error) {
	res, err := tsc.makeRequest(
		"POST",
		fmt.Sprintf("accounts/%s/applications.xml", accountID),
		withAccessToken(accessToken, map[string]interface{}{
			"plan_id":     planID,
			"name":        name,
			"description": description,
		}),
	)
	if err != nil {
		return "", err
	}

	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return "", err
	}

	return xmlFromResponse(res, "//application/user_key/text()")
}

func (tsc *threeScaleClient) DeployProxy(accessToken, serviceID string) error {
	res, err := tsc.makeRequest(
		"POST",
		fmt.Sprintf("services/%s/proxy/deploy.xml", serviceID),
		onlyAccessToken(accessToken),
	)

	if err != nil {
		return err
	}

	return assertStatusCode(http.StatusCreated, res)
}

func (tsc *threeScaleClient) PromoteProxy(accessToken, serviceID, env, to string) (string, error) {
	res, err := tsc.httpc.Get(fmt.Sprintf("https://3scale-admin.%s/admin/api/services/%s/proxy/configs/%s/latest.json?access_token=%s", tsc.wildCardDomain, serviceID, env, accessToken))
	if err != nil {
		return "", err
	}
	if err := assertStatusCode(http.StatusOK, res); err != nil {
		return "", err
	}

	proxyConfigResponse := &struct {
		ProxyConfig struct {
			Content struct {
				Proxy struct {
					Endpoint string `json:"endpoint"`
				} `json:"proxy"`
			} `json:"content"`
			Version int `json:"version"`
		} `json:"proxy_config"`
	}{}

	if err := jsonFromResponse(res, proxyConfigResponse); err != nil {
		return "", err
	}

	res, err = tsc.makeRequest(
		"POST",
		fmt.Sprintf("services/%s/proxy/configs/%s/%d/promote.json", serviceID, env, proxyConfigResponse.ProxyConfig.Version),
		withAccessToken(accessToken, map[string]interface{}{
			"to": to,
		}),
	)
	if err != nil {
		return "", err
	}
	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return "", err
	}

	if err := jsonFromResponse(res, proxyConfigResponse); err != nil {
		return "", err
	}

	return proxyConfigResponse.ProxyConfig.Content.Proxy.Endpoint, nil
}

func (tsc *threeScaleClient) DeleteService(accessToken, serviceID string) error {
	res, err := tsc.makeRequest(
		"DELETE",
		fmt.Sprintf("services/%s.xml", serviceID),
		onlyAccessToken(accessToken),
	)
	if err != nil {
		return err
	}

	return assertStatusCode(http.StatusOK, res)
}

func (tsc *threeScaleClient) DeleteBackend(accessToken string, backendID int) error {
	res, err := tsc.makeRequest(
		"DELETE",
		fmt.Sprintf("backend_apis/%d.json", backendID),
		onlyAccessToken(accessToken),
	)
	if err != nil {
		return err
	}

	return assertStatusCode(http.StatusOK, res)
}

func (tsc *threeScaleClient) DeleteAccount(accessToken, accountID string) error {
	res, err := tsc.makeRequest(
		"DELETE",
		fmt.Sprintf("accounts/%s.xml", accountID),
		onlyAccessToken(accessToken),
	)
	if err != nil {
		return err
	}

	return assertStatusCode(http.StatusOK, res)
}

func (tsc *threeScaleClient) ListTenantAccounts(accessToken string, page int) ([]AccountDetail, error) {
	// curl -v  -X GET "https://master.apps.jmonteir.edy6.s1.devshift.org/admin/api/accounts.json?access_token=AIjluIOs"
	res, err := tsc.makeRequestToMaster(
		"GET",
		"admin/api/accounts.xml",
		withAccessToken(accessToken, map[string]interface{}{
			"page": page,
		}),
	)
	if err != nil {
		return nil, err
	}

	if err := assertStatusCode(http.StatusOK, res); err != nil {
		return nil, err
	}

	accountList := XMLAccountList{}
	if err := responseFromXML(res, &accountList); err != nil {
		return nil, err
	}

	accounts := []AccountDetail{}
	// removes pre created 3scale accounts
	for _, account := range accountList.Accounts {
		if account.Id != 1 && account.Id != 2 {
			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

func (tsc *threeScaleClient) CreateTenant(accessToken string, account AccountDetail, password string, email string) (*SignUpAccount, error) {
	res, err := tsc.makeRequestToMaster(
		"POST",
		"master/api/providers.xml",
		withAccessToken(accessToken, map[string]interface{}{
			"org_name": account.OrgName,
			"username": account.Name,
			"password": password,
			"email":    email,
		}),
	)
	if err != nil {
		return nil, err
	}

	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return nil, err
	}

	signupAccount := &SignUpAccount{}
	if err := responseFromXML(res, signupAccount); err != nil {
		return nil, err
	}

	return signupAccount, nil
}

func (tsc *threeScaleClient) GetTenantAccount(accessToken string, id int) (*SignUpAccount, error) {
	res, err := tsc.makeRequestToMaster(
		"GET",
		fmt.Sprintf("master/api/providers/{%v}.xml", id),
		onlyAccessToken(accessToken),
	)
	if err != nil {
		return nil, err
	}

	if err := assertStatusCode(http.StatusOK, res); err != nil {
		return nil, err
	}

	signupAccount := SignUpAccount{}
	if err := responseFromXML(res, &signupAccount); err != nil {
		return nil, err
	}

	return &signupAccount, nil
}

func (tsc *threeScaleClient) ActivateUser(accessToken string, accountId, userId int) error {
	res, err := tsc.makeRequestToMaster(
		"PUT",
		fmt.Sprintf("admin/api/accounts/%d/users/%d/activate.xml", accountId, userId),
		onlyAccessToken(accessToken),
	)
	if err != nil {
		return err
	}

	if err := assertStatusCode(http.StatusOK, res); err != nil {
		return err
	}

	return nil
}

func (tsc *threeScaleClient) AddAuthProviderToAccount(accessToken string, account AccountDetail, authProviderDetail AuthProviderDetails) error {

	url := fmt.Sprintf("%s/%s", account.AdminBaseURL, "admin/api/account/authentication_providers.json")
	res, err := makeRequest(url,
		"POST",
		withAccessToken(accessToken, map[string]interface{}{
			"kind":                              authProviderDetail.Kind,
			"name":                              authProviderDetail.Name,
			"client_id":                         authProviderDetail.ClientId,
			"client_secret":                     authProviderDetail.ClientSecret,
			"site":                              authProviderDetail.Site,
			"skip_ssl_certificate_verification": authProviderDetail.SkipSSLCertificateVerification,
			"published":                         authProviderDetail.Published,
			"system_name":                       authProviderDetail.SystemName,
		}),
		tsc,
	)
	if err != nil {
		return fmt.Errorf("Error creating new authentication provider for %s tenant account: , %w", account.OrgName, err)
	}

	if err := assertStatusCode(http.StatusCreated, res); err != nil {
		return err
	}

	return nil
}

func (tsc *threeScaleClient) IsAuthProviderAdded(accessToken string, authProviderName string, account AccountDetail) (bool, error) {
	isAuthProviderAdded := false
	url := fmt.Sprintf("%s/%s", account.AdminBaseURL, "admin/api/account/authentication_providers.json")
	res, err := makeRequest(url,
		"GET",
		onlyAccessToken(accessToken),
		tsc,
	)
	if err != nil {
		return isAuthProviderAdded, err
	}

	if err := assertStatusCode(http.StatusOK, res); err != nil {
		return isAuthProviderAdded, err
	}

	authProviders := &AuthProviders{}
	if err := jsonFromResponse(res, &authProviders); err != nil {
		return isAuthProviderAdded, err
	}

	for _, authProvider := range authProviders.AuthProviders {
		if authProvider.ProviderDetails.Name == authProviderName {
			isAuthProviderAdded = true
			break
		}
	}

	return isAuthProviderAdded, nil
}

func (tsc *threeScaleClient) DeleteTenants(accessToken string, accounts []AccountDetail) error {
	for _, account := range accounts {
		err := tsc.DeleteTenant(accessToken, account.Id)
		if err == nil {
			return fmt.Errorf("Error deleting tenant: %s", account.Name)
		}
	}
	return nil
}

func (tsc *threeScaleClient) DeleteTenant(accessToken string, accountId int) error {
	res, err := tsc.makeRequestToMaster(
		"DELETE",
		fmt.Sprintf("master/api/providers/%d.xml", accountId),
		onlyAccessToken(accessToken),
	)
	if err != nil {
		return err
	}

	if err := assertStatusCode(http.StatusOK, res); err != nil {
		return err
	}

	return nil
}

func makeRequest(url, method string, parameters map[string]interface{}, tsc *threeScaleClient) (*http.Response, error) {
	dataJSON, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		method,
		url,
		bytes.NewBuffer(dataJSON),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return tsc.httpc.Do(req)
}

func (tsc *threeScaleClient) makeRequest(method, path string, parameters map[string]interface{}) (*http.Response, error) {
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/%s", tsc.wildCardDomain, path)
	return makeRequest(url, method, parameters, tsc)
}

func (tsc *threeScaleClient) makeRequestToMaster(method, path string, parameters map[string]interface{}) (*http.Response, error) {
	url := fmt.Sprintf("https://master.%s/%s", tsc.wildCardDomain, path)
	return makeRequest(url, method, parameters, tsc)
}

func xmlFromResponse(res *http.Response, xpath string) (string, error) {
	doc, err := xmlquery.Parse(res.Body)
	if err != nil {
		return "", err
	}

	node := xmlquery.FindOne(doc, xpath)
	if node == nil {
		return "", fmt.Errorf("query not found in doc")
	}

	return node.Data, nil
}

func jsonFromResponse(res *http.Response, target interface{}) error {
	logrus.Infof("body %v", res.Body)
	return json.NewDecoder(res.Body).Decode(target)
}

func responseFromXML(res *http.Response, target interface{}) error {
	return xml.NewDecoder(res.Body).Decode(target)
}

func onlyAccessToken(accessToken string) map[string]interface{} {
	return map[string]interface{}{
		"access_token": accessToken,
	}
}

func withAccessToken(accessToken string, data map[string]interface{}) map[string]interface{} {
	data["access_token"] = accessToken
	return data
}

func assertStatusCode(expected int, res *http.Response) error {
	if res.StatusCode == expected {
		return nil
	}

	body, _ := ioutil.ReadAll(res.Body)
	return fmt.Errorf("unexpected status code: %d. Body: %s", res.StatusCode, string(body))
}
