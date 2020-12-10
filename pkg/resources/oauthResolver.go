package resources

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
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
	if os.IsNotExist(err) && os.Getenv(k8sutil.ForceRunModeEnv) == string(k8sutil.LocalRunMode) {
		or.Log.Warning("GetOauthEndPoint() will skip certificate verification - this is acceptable only if operator is running locally")
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		or.client.Transport = tr
		or.client.Timeout = time.Second * 10
	} else {
		if err != nil {
			return nil, fmt.Errorf("failed to read k8s root CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true}
		or.client.Transport = transport
	}

	resp, err := or.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth server config from well known endpoint %s: %w", url, err)
	}
	defer resp.Body.Close()
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
