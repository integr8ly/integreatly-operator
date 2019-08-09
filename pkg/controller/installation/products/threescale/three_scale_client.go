package threescale

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

//go:generate moq -out three_scale_moq.go . ThreeScaleInterface
type ThreeScaleInterface interface {
	SetNamespace(ns string)
	AddSSOIntegration(data map[string]string, accessToken string) (*http.Response, error)
	GetUser(username, accessToken string) (*User, error)
	GetUsers(accessToken string) (*Users, error)
	AddUser(username string, email string, password string, accessToken string) (*http.Response, error)
	DeleteUser(userId int, accessToken string) (*http.Response, error)
	SetUserAsAdmin(userId int, accessToken string) (*http.Response, error)
	SetUserAsMember(userId int, accessToken string) (*http.Response, error)
	UpdateUser(userId int, username string, email string, accessToken string) (*http.Response, error)
}

type threeScaleClient struct {
	httpc          *http.Client
	wildCardDomain string
	ns             string
}

func NewThreeScaleClient(httpc *http.Client, wildCardDomain string) *threeScaleClient {
	return &threeScaleClient{
		httpc:          httpc,
		wildCardDomain: wildCardDomain,
	}
}

func (tsc *threeScaleClient) SetNamespace(ns string) {
	tsc.ns = ns
}

func (tsc *threeScaleClient) AddSSOIntegration(data map[string]string, accessToken string) (*http.Response, error) {
	data["access_token"] = accessToken
	reqData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

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

	return nil, errors.New("3Scale admin user not found")
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

func (tsc *threeScaleClient) AddUser(username string, email string, password string, accessToken string) (*http.Response, error) {
	data := make(map[string]string)
	data["access_token"] = accessToken
	data["username"] = username
	data["email"] = email
	data["password"] = password
	reqData, err := json.Marshal(data)
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

func (tsc *threeScaleClient) DeleteUser(userId int, accessToken string) (*http.Response, error) {
	data := make(map[string]string)
	data["access_token"] = accessToken
	reqData, err := json.Marshal(data)

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d.json", tsc.wildCardDomain, userId),
		bytes.NewBuffer(reqData))
	req.Header.Add("Content-type", "application/json")
	res, err := tsc.httpc.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (tsc *threeScaleClient) SetUserAsAdmin(userId int, accessToken string) (*http.Response, error) {
	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
	})
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d/admin.json", tsc.wildCardDomain, userId)
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

func (tsc *threeScaleClient) SetUserAsMember(userId int, accessToken string) (*http.Response, error) {
	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
	})
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d/member.json", tsc.wildCardDomain, userId)
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

func (tsc *threeScaleClient) UpdateUser(userId int, username string, email string, accessToken string) (*http.Response, error) {
	data, err := json.Marshal(map[string]string{
		"access_token": accessToken,
		"username":     username,
		"email":        email,
	})
	url := fmt.Sprintf("https://3scale-admin.%s/admin/api/users/%d.json", tsc.wildCardDomain, userId)
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
