package threescale

import "net/http"

type Users struct {
	Users []*User `json:"users"`
}

type User struct {
	UserDetails UserDetails `json:"user"`
}

type UserDetails struct {
	Id       int    `json:"id"`
	State    string `json:"state"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type AuthProviders struct {
	AuthProviders []*AuthProvider `json:"authentication_providers"`
}

type AuthProvider struct {
	ProviderDetails AuthProviderDetails `json:"authentication_provider"`
}

type AuthProviderDetails struct {
	Id                             int    `json:"id"`
	Kind                           string `json:"kind"`
	AccountType                    string `json:"account_type"`
	Name                           string `json:"name"`
	SystemName                     string `json:"system_name"`
	ClientId                       string `json:"client_id"`
	ClientSecret                   string `json:"client_secret"`
	Site                           string `json:"site"`
	AuthorizeURL                   string `json:"authorize_url"`
	SkipSSLCertificateVerification bool   `json:"skip_ssl_certificate_verification"`
	AutomaticallyApproveAccounts   bool   `json:"automatically_approve_accounts"`
	AccountId                      int    `json:"account_id"`
	UsernameKey                    string `json:"username_key"`
	IdentifierKey                  string `json:"identifier_key"`
	TrustEmail                     bool   `json:"trust_email"`
	Published                      bool   `json:"published"`
	CreatedAt                      string `json:"created_at"`
	UpdatedAt                      string `json:"updated_at"`
	CallbackUrl                    string `json:"callback_url"`
}

type tsError struct {
	message    string
	StatusCode int
}

type SignUpAccount struct {
	AccountDetail      AccountDetail      `xml:"account"`
	AccountAccessToken AccountAccessToken `xml:"access_token"`
}

type Account struct {
	Id      int    `xml:"id"`
	Name    string `xml:"name"`
	OrgName string `xml:"org_name"`
}

type AccountDetail struct {
	Id           int      `xml:"id"`
	Name         string   `xml:"name"`
	OrgName      string   `xml:"org_name"`
	AdminBaseURL string   `xml:"admin_base_url"`
	State        string   `xml:"state"`
	Users        XMLUsers `xml:"users"`
}

type AccountAccessToken struct {
	Id    int    `xml:"id"`
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type XMLAccountList struct {
	Accounts []AccountDetail `xml:"account"`
}

type XMLUserDetails struct {
	Id       int    `xml:"id"`
	State    string `xml:"state"`
	Role     string `xml:"role"`
	Username string `xml:"username"`
	Email    string `xml:"email"`
}

type XMLUsers struct {
	User []XMLUserDetails `xml:"user"`
}

func (tse *tsError) Error() string {
	return tse.message
}

func tsIsNotFoundError(e error) bool {
	switch e := e.(type) {
	case *tsError:
		return e.StatusCode == http.StatusNotFound
	}

	return false
}
