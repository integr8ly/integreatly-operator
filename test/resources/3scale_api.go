package resources

import (
	"errors"
	"fmt"
	"net/http"
	url2 "net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ThreeScalePingUrl          = "%v/admin/api/applications.xml"
	ThreeScaleProductCreateUrl = "%v/apiconfig/services"
	ThreeScaleProductDeleteUrl = "%v/apiconfig/services/%v"
	ThreeScaleClient           = "3scale"
	ThreeScaleUserInviteUrl    = "%v/p/admin/account/invitations/"
)

type ThreeScaleAPIClient interface {
	Ping() error
	LoginOpenshift(masterUrl, username, password, namespacePrefix string) error
	Login3Scale(clientSecret string) error
	CreateProduct(name string) (string, error)
	DeleteProduct(href string) error
	SendUserInvitation(name string) (string, error)
	SetUserAsAdmin(username string, email string, userID string) error
	GetUserId(username string) (string, error)
	VerifyUserIsAdmin(userID string) (bool, error)
}

type ThreeScaleAPIClientImpl struct {
	host         string
	client       *http.Client
	kubeClient   k8sclient.Client
	token        string
	keycloakHost string
	redirectUrl  string
	logger       SimpleLogger
}

func NewThreeScaleAPIClient(host, keycloakHost, redirectUrl string, client *http.Client, kubeClient k8sclient.Client, logger SimpleLogger) ThreeScaleAPIClient {
	return &ThreeScaleAPIClientImpl{
		host:         host,
		client:       client,
		kubeClient:   kubeClient,
		keycloakHost: keycloakHost,
		redirectUrl:  redirectUrl,
		logger:       logger,
	}
}

// Request a csrf token to be used in a form post request by sending
// a get request to the form url first
func requestCRSFToken(c *http.Client, formUrl string) (string, error) {
	resp, err := c.Get(formUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := ParseHtmlResponse(resp)
	if err != nil {
		return "", err
	}

	selector := doc.Find("meta[name='csrf-token']")
	if selector.Length() == 0 {
		return "", errors.New(fmt.Sprintf("no csrf token found in: %v", doc.Text()))
	}

	// Get the csrf token from the form to use it in the post
	csrf, _ := selector.Attr("content")
	return csrf, nil
}

func (r *ThreeScaleAPIClientImpl) Login3Scale(clientSecret string) error {
	token, err := Auth3Scale(r.client, r.redirectUrl, r.keycloakHost, ThreeScaleClient, clientSecret)
	if err != nil {
		return err
	}

	r.token = token
	return nil
}

func (r *ThreeScaleAPIClientImpl) LoginOpenshift(masterUrl, username, password, namespacePrefix string) error {
	authUrl := fmt.Sprintf("%s/auth/login", masterUrl)
	if err := DoAuthOpenshiftUser(authUrl, username, password, r.client, "testing-idp", r.logger); err != nil {
		return err
	}

	openshiftClient := NewOpenshiftClient(r.client, masterUrl)
	if err := OpenshiftUserReconcileCheck(openshiftClient, r.kubeClient, namespacePrefix, username); err != nil {
		return err
	}
	return nil
}

// Call the applications endpoint of 3Scale to make sure it is available
// and the user has permissions to retrieve the list of applications
func (r *ThreeScaleAPIClientImpl) Ping() error {
	url := fmt.Sprintf(ThreeScalePingUrl, r.host)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", r.token))
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("expected 200 but got %v", resp.StatusCode))
	}

	return nil
}

// Delete a 3Scale product by id
func (r *ThreeScaleAPIClientImpl) DeleteProduct(id string) error {
	url := fmt.Sprintf(ThreeScaleProductDeleteUrl, r.host, id)
	formUrl := fmt.Sprintf("%v/apiconfig/services/%v/edit", r.host, id)

	csrf, err := requestCRSFToken(r.client, formUrl)
	if err != nil {
		return err
	}

	formValues := url2.Values{
		"authenticity_token": []string{csrf},
		"_method":            []string{"delete"},
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(formValues.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", r.token))

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("expected 200 but got %v", resp.StatusCode))
	}

	return nil
}

// Create a 3Scale product
func (r *ThreeScaleAPIClientImpl) CreateProduct(name string) (string, error) {
	url := fmt.Sprintf(ThreeScaleProductCreateUrl, r.host)
	formUrl := fmt.Sprintf("%v/apiconfig/services/new", r.host)

	// First try to get the CRSF token by requesting the form
	csrf, err := requestCRSFToken(r.client, formUrl)
	if err != nil {
		return "", err
	}

	formValues := url2.Values{
		"service[name]":        []string{name},
		"service[system_name]": []string{fmt.Sprintf("%v-system", name)},
		"service[description]": []string{fmt.Sprintf("%v dummy service", name)},
		"commit":               []string{"Create+Product"},
		"authenticity_token":   []string{csrf},
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(formValues.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", r.token))

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("expected 200 but got %v", resp.StatusCode))
	}

	// Parse the html to get a link back to the created service
	doc, err := ParseHtmlResponse(resp)
	selector := doc.Find("a.pf-c-nav__link")
	if selector.Length() == 0 {
		return "", errors.New("unable to retrieve service id")
	}

	href, _ := selector.Attr("href")
	if strings.Contains(href, "/") == false {

	}

	id := strings.Split(href, "/")
	return id[len(id)-1], nil
}

// Create a 3Scale user invite
func (r *ThreeScaleAPIClientImpl) SendUserInvitation(name string) (string, error) {

	url := fmt.Sprintf(ThreeScaleUserInviteUrl, r.host)
	formUrl := fmt.Sprintf("%v/p/admin/account/invitations/new", r.host)

	// First try to get the CRSF token by requesting the form
	csrf, err := requestCRSFToken(r.client, formUrl)
	if err != nil {
		return "", err
	}

	formValues := url2.Values{
		"invitation[email]": []string{name},
		//"service[system_name]": []string{fmt.Sprintf("%v-system", name)},
		//"service[description]": []string{fmt.Sprintf("%v dummy service", name)},
		"commit":             []string{"Send"},
		"authenticity_token": []string{csrf},
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(formValues.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", r.token))

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("expected 200 but got %v", resp.StatusCode))
	}

	return "Completed", nil
}

func (r *ThreeScaleAPIClientImpl) SetUserAsAdmin(username string, email string, userID string) error {

	url := fmt.Sprintf("%v/p/admin/account/users/%v", r.host, userID)
	formUrl := fmt.Sprintf("%v/p/admin/account/users/%v/edit", r.host, userID)

	// First try to get the CRSF token by requesting the form
	csrf, err := requestCRSFToken(r.client, formUrl)
	if err != nil {
		return err
	}

	formValues := url2.Values{
		"user[username]":              []string{username},
		"user[email]":                 []string{email},
		"user[role]":                  []string{"admin"},
		"user[password]":              []string{},
		"user[password_confirmation]": []string{},
		"commit":                      []string{"Update+User"},
		"authenticity_token":          []string{csrf},
		"_method":                     []string{"patch"},
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(formValues.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", r.token))

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("expected 200 but got %v", resp.StatusCode))
	}

	return nil
}

func (r *ThreeScaleAPIClientImpl) GetUserId(username string) (string, error) {
	url := fmt.Sprintf("%v/p/admin/account/users", r.host)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", r.token))
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("expected 200 but got %v", resp.StatusCode))
	}

	doc, err := ParseHtmlResponse(resp)
	if err != nil {
		return "", err
	}

	links := doc.Find("a[title='Edit']")

	link := ""
	links.Each(func(index int, s *goquery.Selection) {
		if s.Contents().Text() == username {
			link, _ = s.Attr("href")
			return
		}
	})

	if link == "" {
		return "", fmt.Errorf("Failed to retrieve link to edit user")
	}

	userId := ""
	s := strings.Split(link, "/")
	for _, val := range s {
		if _, err := strconv.Atoi(val); err == nil {
			userId = val
			break
		}
	}

	return userId, nil
}

func (r *ThreeScaleAPIClientImpl) VerifyUserIsAdmin(userID string) (bool, error) {

	formUrl := fmt.Sprintf("%v/p/admin/account/users/%v/edit", r.host, userID)
	resp, err := r.client.Get(formUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	doc, err := ParseHtmlResponse(resp)
	if err != nil {
		return false, err
	}

	selector := doc.Find("input[id='user_role_admin']")
	_, exists := selector.Attr("checked")
	if exists {
		return true, nil
	}

	return false, nil
}
