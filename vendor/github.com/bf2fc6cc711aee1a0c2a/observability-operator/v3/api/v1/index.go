package v1

import (
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/core/v1"
)

type RepositoryInfo struct {
	Repository  string
	Channel     string
	Tag         string
	AccessToken string
	Source      *v1.Secret
}

type GrafanaIndex struct {
	Dashboards []string `json:"dashboards"`
}

type DexConfig struct {
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Secret   string `json:"secret"`

	// Kept for backwards compatibility
	// TODO: remove after v3.0.3
	CredentialSecretNamespace string `json:"credentialSecretNamespace"`
	CredentialSecretName      string `json:"credentialSecretName"`
}

type RedhatSsoConfig struct {
	Url           string `json:"redHatSsoAuthServerUrl"`
	Realm         string `json:"redHatSsoRealm"`
	MetricsClient string `json:"metricsClientId"`
	MetricsSecret string `json:"metricsSecret"`
	LogsClient    string `json:"logsClientId"`
	LogsSecret    string `json:"logsSecret"`
}

func (in *RedhatSsoConfig) HasAuthServer() bool {
	return in.Url != "" && in.Realm != ""
}

func (in *RedhatSsoConfig) HasMetrics() bool {
	return in.HasAuthServer() && in.MetricsClient != "" && in.MetricsSecret != ""
}

func (in *RedhatSsoConfig) HasLogs() bool {
	return in.HasAuthServer() && in.LogsClient != "" && in.LogsSecret != ""
}

type ObservatoriumIndex struct {
	Id              string                `json:"id"`
	SecretName      string                `json:"secretName,omitempty"`
	Gateway         string                `json:"gateway"`
	Tenant          string                `json:"tenant"`
	AuthType        ObservabilityAuthType `json:"authType"`
	DexConfig       *DexConfig            `json:"dexConfig,omitempty"`
	RedhatSsoConfig *RedhatSsoConfig      `json:"redhatSsoConfig,omitempty"`
}

func (in *ObservatoriumIndex) IsValid() bool {
	return in.Gateway != "" && in.Tenant != ""
}

type RemoteWriteIndex struct {
	QueueConfig         *v12.QueueConfig    `json:"queueConfig,omitempty"`
	RemoteTimeout       string              `json:"remoteTimeout,omitempty"`
	ProxyUrl            string              `json:"proxyUrl,omitempty"`
	WriteRelabelConfigs []v12.RelabelConfig `json:"writeRelabelConfigs,omitempty"`
}

type AlertmanagerIndex struct {
	PagerDutySecretName           string `json:"pagerDutySecretName"`
	PagerDutySecretNamespace      string `json:"pagerDutySecretNamespace"`
	DeadmansSnitchSecretName      string `json:"deadmansSnitchSecretName"`
	DeadmansSnitchSecretNamespace string `json:"deadmansSnitchSecretNamespace"`
}

type PrometheusIndex struct {
	Rules         []string `json:"rules"`
	PodMonitors   []string `json:"pod_monitors"`
	Federation    string   `json:"federation,omitempty"`
	Observatorium string   `json:"observatorium,omitempty"`
	RemoteWrite   string   `json:"remoteWrite,omitempty"`
}

type PromtailIndex struct {
	Enabled                bool              `json:"enabled,omitempty"`
	NamespaceLabelSelector map[string]string `json:"namespaceLabelSelector,omitempty"`
	Observatorium          string            `json:"observatorium,omitempty"`
}

type RepositoryConfig struct {
	Grafana      *GrafanaIndex        `json:"grafana,omitempty"`
	Prometheus   *PrometheusIndex     `json:"prometheus,omitempty"`
	Alertmanager *AlertmanagerIndex   `json:"alertmanager,omitempty"`
	Promtail     *PromtailIndex       `json:"promtail,omitempty"`
	Observatoria []ObservatoriumIndex `json:"observatoria,omitempty"`
}

type RepositoryIndex struct {
	BaseUrl     string            `json:"-"`
	AccessToken string            `json:"-"`
	Tag         string            `json:"-"`
	Source      *v1.Secret        `json:"-"`
	Id          string            `json:"id"`
	Config      *RepositoryConfig `json:"config"`
}
