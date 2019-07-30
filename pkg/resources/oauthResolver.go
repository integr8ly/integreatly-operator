package resources

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

const oauthServerDetails = "%s/.well-known/oauth-authorization-server"
const defaultHost = "https://openshift.default.svc"

type OauthResolver struct {
	client   *http.Client
	Host     string
	InSecure bool
}

func NewOauthResolver(client *http.Client) *OauthResolver {
	return &OauthResolver{
		client: client,
		Host:   defaultHost,
	}
}

func (or *OauthResolver) GetOauthEndPoint() (*OauthServerConfig, error) {
	url := fmt.Sprintf(oauthServerDetails, or.Host)
	if or.InSecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		or.client.Transport = tr
	}
	resp, err := or.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get oauth server config from well known endpoint "+url)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	ret := &OauthServerConfig{}
	if err := dec.Decode(ret); err != nil {
		return nil, errors.Wrap(err, "failed to decode response from well known end point "+url)
	}

	return ret, nil
}

type OauthServerConfig struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	ScopesSupported               []string `json:"scopes_supported"`
	ResponseTypesSupported        []string `json:"response_types_supported"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}
