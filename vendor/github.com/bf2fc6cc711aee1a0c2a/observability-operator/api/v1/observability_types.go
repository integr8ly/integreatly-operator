/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ObservabilityStageName string

type ObservabilityStageStatus string

type ObservabilityAuthType string

const (
	GrafanaInstallation      ObservabilityStageName = "Grafana"
	GrafanaConfiguration     ObservabilityStageName = "GrafanaConfiguration"
	PrometheusInstallation   ObservabilityStageName = "Prometheus"
	PrometheusConfiguration  ObservabilityStageName = "PrometheusConfiguration"
	PrometheusRules          ObservabilityStageName = "PrometheusRules"
	CsvRemoval               ObservabilityStageName = "CsvRemoval"
	TokenRequest             ObservabilityStageName = "TokenRequest"
	PromtailInstallation     ObservabilityStageName = "PromtailInstallation"
	AlertmanagerInstallation ObservabilityStageName = "AlertmanagerInstallation"
	Configuration            ObservabilityStageName = "configuration"
)

const (
	ResultSuccess    ObservabilityStageStatus = "success"
	ResultFailed     ObservabilityStageStatus = "failed"
	ResultInProgress ObservabilityStageStatus = "in progress"
)

const (
	AuthTypeDex ObservabilityAuthType = "dex"
)

type DexConfig struct {
	Url                       string `json:"url"`
	CredentialSecretNamespace string `json:"credentialSecretNamespace"`
	CredentialSecretName      string `json:"credentialSecretName"`
}

type DashboardSource struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}

type GrafanaConfig struct {
	// How often to refetch the dashboards?
	ResyncPeriod string `json:"resyncPeriod,omitempty"`
}

type ObservatoriumConfig struct {
	// Observatorium Gateway API URL
	Gateway string `json:"gateway"`
	// Observatorium tenant name
	Tenant string `json:"tenant"`

	// Auth type. Currently only dex is supported
	AuthType ObservabilityAuthType `json:"authType,omitempty"`

	// Dex configuration
	AuthDex *DexConfig `json:"dexConfig,omitempty"`
}

type AlertmanagerConfig struct {
	PagerDutySecretName           string `json:"pagerDutySecretName"`
	PagerDutySecretNamespace      string `json:"pagerDutySecretNamespace,omitempty"`
	DeadMansSnitchSecretName      string `json:"deadMansSnitchSecretName"`
	DeadMansSnitchSecretNamespace string `json:"deadMansSnitchSecretNamespace,omitempty"`
}

type Storage struct {
	PrometheusStorageSpec *prometheusv1.StorageSpec `json:"prometheus,omitempty"`
}

// ObservabilitySpec defines the desired state of Observability
type ObservabilitySpec struct {
	// Cluster ID. If not provided, the operator tries to obtain it.
	ClusterID             string                `json:"clusterId,omitempty"`
	ConfigurationSelector *metav1.LabelSelector `json:"configurationSelector,omitempty"`
	ResyncPeriod          string                `json:"resyncPeriod,omitempty"`
	Storage               *Storage              `json:"storage,omitempty"`
	Tolerations           []v1.Toleration       `json:"tolerations,omitempty"`
	Affinity              *v1.Affinity          `json:"affinity,omitempty"`
}

// ObservabilityStatus defines the observed state of Observability
type ObservabilityStatus struct {
	Stage        ObservabilityStageName   `json:"stage"`
	StageStatus  ObservabilityStageStatus `json:"stageStatus"`
	LastMessage  string                   `json:"lastMessage,omitempty"`
	TokenExpires int64                    `json:"tokenExpires,omitempty"`
	ClusterID    string                   `json:"clusterId,omitempty"`
	LastSynced   int64                    `json:"lastSynced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Observability is the Schema for the observabilities API
type Observability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObservabilitySpec   `json:"spec,omitempty"`
	Status ObservabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObservabilityList contains a list of Observability
type ObservabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Observability `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Observability{}, &ObservabilityList{})
}
