package resources

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	k8s "github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const oauthServerDetails = "%s/.well-known/oauth-authorization-server"
const defaultHost = "https://openshift.default.svc"
const rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

type OauthResolver struct {
	client *http.Client
	Host   string
	Log    l.Logger
}

func NewOauthResolver(client *http.Client, log l.Logger) *OauthResolver {
	client.Timeout = time.Second * 10
	return &OauthResolver{
		client: client,
		Host:   defaultHost,
		Log:    log,
	}
}

func (or *OauthResolver) GetOauthEndPoint() (*OauthServerConfig, error) {
	url := fmt.Sprintf(oauthServerDetails, or.Host)

	caCert, err := ioutil.ReadFile(rootCAFile)
	// if running locally, CA certificate isn't available in expected path
	if os.IsNotExist(err) && k8s.IsRunLocally() {
		or.Log.Warning("GetOauthEndPoint() will skip certificate verification - this is acceptable only if operator is running locally")
		/* #nosec */
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // gosec G402 skipped as only ran when operator is running locally (used in rhmi only)
		}
		or.client.Transport = tr
		or.client.Timeout = time.Second * 10
	} else {
		if err != nil {
			return nil, fmt.Errorf("failed to read k8s root CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		/* #nosec */
		tlsConfig := &tls.Config{
			RootCAs: caCertPool, // gosec G402 skipped as only used in rhmi
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true}
		or.client.Transport = transport
	}

	resp, err := or.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth server config from well known endpoint %s: %w", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			or.Log.Error("Closing response body error: ", err)
		}
	}(resp.Body)
	dec := json.NewDecoder(resp.Body)
	ret := &OauthServerConfig{}
	if err := dec.Decode(ret); err != nil {
		return nil, fmt.Errorf("failed to decode response from well known endpoint %s: %w", url, err)
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
