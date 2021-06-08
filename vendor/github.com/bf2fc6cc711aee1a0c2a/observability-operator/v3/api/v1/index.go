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

type DexIndex struct {
	Url                       string `json:"url"`
	CredentialSecretName      string `json:"credentialSecretName"`
	CredentialSecretNamespace string `json:"credentialSecretNamespace"`
}

type ObservatoriumIndex struct {
	Id        string                `json:"id"`
	Gateway   string                `json:"gateway"`
	Tenant    string                `json:"tenant"`
	AuthType  ObservabilityAuthType `json:"authType"`
	DexConfig *DexConfig            `json:"dexConfig,omitempty"`
}

type RemoteWriteIndex struct {
	QueueConfig         *v12.QueueConfig    `json:"queueConfig,omitempty"`
	RemoteTimeout       string              `json:"remoteTimeout,omitempty"`
	ProxyUrl            string              `json:"proxyUrl,omitempty"`
	WriteRelabelConfigs []v12.RelabelConfig `json:"writeRelabelConfigs,omitempty"`

	// for v2.0.0 backwards compatibility
	Patterns []string `json:"patterns,omitempty"`
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
