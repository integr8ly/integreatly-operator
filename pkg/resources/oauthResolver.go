package resources

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const oauthServerDetails = "%s/.well-known/oauth-authorization-server"
const defaultHost = "https://openshift.default.svc"
const rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

type OauthResolver struct {
	client *http.Client
	Host   string
}

func NewOauthResolver(client *http.Client) *OauthResolver {
	return &OauthResolver{
		client: client,
		Host:   defaultHost,
	}
}

func (or *OauthResolver) GetOauthEndPoint() (*OauthServerConfig, error) {
	url := fmt.Sprintf(oauthServerDetails, or.Host)

	caCert, err := ioutil.ReadFile(rootCAFile)
	// if running locally, CA certificate isn't available in expected path
	if os.IsNotExist(err) || os.Getenv("INTEGREATLY_OPERATOR_DISABLE_ELECTION") != "" {
		logrus.Warn("GetOauthEndPoint() will skip certificate verification - this is acceptable only if operator is running locally")
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		or.client.Transport = tr
	} else {
		if err != nil {
			return nil, errors.Wrap(err, "failed to read k8s root CA file")
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		or.client.Transport = transport
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
