package threescale

import "net/http"

type Users struct {
	Users []*User `json:"users"`
}

type User struct {
	UserDetails UserDetails `json:"user"`
}

type UserDetails struct {
	ID       int    `json:"id"`
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
	ID                             int    `json:"id"`
	Kind                           string `json:"kind"`
	AccountType                    string `json:"account_type"`
	Name                           string `json:"name"`
	SystemName                     string `json:"system_name"`
	ClientID                       string `json:"client_id"`
	ClientSecret                   string `json:"client_secret"`
	Site                           string `json:"site"`
	AuthorizeURL                   string `json:"authorize_url"`
	SkipSSLCertificateVerification bool   `json:"skip_ssl_certificate_verification"`
	AutomaticallyApproveAccounts   bool   `json:"automatically_approve_accounts"`
	AccountID                      int    `json:"account_id"`
	UsernameKey                    string `json:"username_key"`
	IdentifierKey                  string `json:"identifier_key"`
	TrustEmail                     bool   `json:"trust_email"`
	Published                      bool   `json:"published"`
	CreatedAt                      string `json:"created_at"`
	UpdatedAt                      string `json:"updated_at"`
	CallbackURL                    string `json:"callback_url"`
}

type tsError struct {
	message    string
	StatusCode int
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
