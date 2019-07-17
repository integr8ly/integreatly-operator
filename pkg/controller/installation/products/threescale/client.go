package threescale

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type threeScaleClient struct {
	httpc          *http.Client
	wildCardDomain string
	ns             string
}

func NewThreeScaleClient(httpc *http.Client, wildCardDomain string, ns string) *threeScaleClient {
	return &threeScaleClient{
		httpc:          httpc,
		wildCardDomain: wildCardDomain,
		ns:             ns,
	}
}

func (tsc *threeScaleClient) AddSSOIntegration(data map[string]string, accessToken string) (*http.Response, error) {
	data["access_token"] = accessToken
	reqData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(
		fmt.Sprintf("https://3scale-admin.%s/admin/api/account/authentication_providers.json", tsc.wildCardDomain),
		"application/json",
		bytes.NewBuffer(reqData),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (tsc *threeScaleClient) GetAdminUser(accessToken string) (*User, error) {
	users, err := tsc.GetUsers(accessToken)
	if err != nil {
		return nil, err
	}

	for _, u := range users.Users {
		if u.UserDetails.Role == "admin" {
			return u, nil
		}
	}

	return nil, errors.New("3Scale admin user not found")
}

func (tsc *threeScaleClient) GetUsers(accessToken string) (*Users, error) {
	res, err := http.Get(
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

func (tsc *threeScaleClient) AddUser(username string, email string, password string, accessToken string) (*http.Response, error) {
	data := make(map[string]string)
	data["access_token"] = accessToken
	data["username"] = username
	data["email"] = email
	data["password"] = password
	reqData, err := json.Marshal(data)
	res, err := http.Post(
		fmt.Sprintf("https://3scale-admin.%s/admin/api/users.json", tsc.wildCardDomain),
		"application/json",
		bytes.NewBuffer(reqData),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (tsc *threeScaleClient) UpdateAdminPortalUserDetails(username string, email string, accessToken string) (*http.Response, error) {
	tsAdmin, err := tsc.GetAdminUser(accessToken)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
		"username":     username,
		"email":        email,
	})
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d.json", tsc.wildCardDomain, tsAdmin.UserDetails.Id)
	req, err := http.NewRequest(
		"PUT",
		url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := tsc.httpc.Do(req)

	return res, err
}
