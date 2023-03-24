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
	Csv                      ObservabilityStageName = "Csv"
	TokenRequest             ObservabilityStageName = "TokenRequest"
	PromtailInstallation     ObservabilityStageName = "PromtailInstallation"
	AlertmanagerInstallation ObservabilityStageName = "AlertmanagerInstallation"
	Configuration            ObservabilityStageName = "configuration"
	LoggingInstallation      ObservabilityStageName = "Logging"
	Migration                ObservabilityStageName = "resource name migration"
)

const (
	ResultSuccess    ObservabilityStageStatus = "success"
	ResultFailed     ObservabilityStageStatus = "failed"
	ResultInProgress ObservabilityStageStatus = "in progress"
)

const (
	AuthTypeDex    ObservabilityAuthType = "dex"
	AuthTypeRedhat ObservabilityAuthType = "redhat"
)

type Storage struct {
	PrometheusStorageSpec   *prometheusv1.StorageSpec `json:"prometheus,omitempty"`
	AlertManagerStorageSpec *prometheusv1.StorageSpec `json:"alertmanager,omitempty"`
}

type SelfContained struct {
	DisableRepoSync                       *bool                    `json:"disableRepoSync,omitempty"`
	DisableObservatorium                  *bool                    `json:"disableObservatorium,omitempty"`
	DisablePagerDuty                      *bool                    `json:"disablePagerDuty,omitempty"`
	DisableDeadmansSnitch                 *bool                    `json:"disableDeadmansSnitch,omitempty"`
	DisableSmtp                           *bool                    `json:"disableSmtp,omitempty"`
	DisableBlackboxExporter               *bool                    `json:"disableBlackboxExporter,omitempty"`
	SelfSignedCerts                       *bool                    `json:"selfSignedCerts,omitempty"`
	OverrideSelectors                     *bool                    `json:"overrideSelectors,omitempty"`
	FederatedMetrics                      []string                 `json:"federatedMetrics,omitempty"`
	PodMonitorLabelSelector               *metav1.LabelSelector    `json:"podMonitorLabelSelector,omitempty"`
	PodMonitorNamespaceSelector           *metav1.LabelSelector    `json:"podMonitorNamespaceSelector,omitempty"`
	ServiceMonitorLabelSelector           *metav1.LabelSelector    `json:"serviceMonitorLabelSelector,omitempty"`
	ServiceMonitorNamespaceSelector       *metav1.LabelSelector    `json:"serviceMonitorNamespaceSelector,omitempty"`
	RuleLabelSelector                     *metav1.LabelSelector    `json:"ruleLabelSelector,omitempty"`
	RuleNamespaceSelector                 *metav1.LabelSelector    `json:"ruleNamespaceSelector,omitempty"`
	ProbeLabelSelector                    *metav1.LabelSelector    `json:"probeSelector,omitempty"`
	ProbeNamespaceSelector                *metav1.LabelSelector    `json:"probeNamespaceSelector,omitempty"`
	GrafanaDashboardLabelSelector         *metav1.LabelSelector    `json:"grafanaDashboardLabelSelector,omitempty"`
	AlertManagerConfigSecret              string                   `json:"alertManagerConfigSecret,omitempty"`
	AlertManagerVersion                   string                   `json:"alertManagerVersion,omitempty"`
	BlackboxBearerTokenSecret             string                   `json:"blackboxBearerTokenSecret,omitempty"`
	PrometheusVersion                     string                   `json:"prometheusVersion,omitempty"`
	AlertManagerResourceRequirement       *v1.ResourceRequirements `json:"alertManagerResourceRequirement,omitempty"`
	PrometheusResourceRequirement         *v1.ResourceRequirements `json:"prometheusResourceRequirement,omitempty"`
	PrometheusOperatorResourceRequirement *v1.ResourceRequirements `json:"prometheusOperatorResourceRequirement,omitempty"`
	GrafanaResourceRequirement            *v1.ResourceRequirements `json:"grafanaResourceRequirement,omitempty"`
	GrafanaOperatorResourceRequirement    *v1.ResourceRequirements `json:"grafanaOperatorResourceRequirement,omitempty"`
	GrafanaVersion                        string                   `json:"grafanaVersion,omitempty"`
	GrafanaInitImage                      string                   `json:"grafanaInitImage,omitempty"`
	DisableLogging                        *bool                    `json:"disableLogging,omitempty"`
	OriginOauthProxyImage                 string                   `json:"originOauthProxyImage,omitempty"`
}

// ObservabilitySpec defines the desired state of Observability
type ObservabilitySpec struct {
	// Cluster ID. If not provided, the operator tries to obtain it.
	ClusterID               string                `json:"clusterId,omitempty"`
	ConfigurationSelector   *metav1.LabelSelector `json:"configurationSelector,omitempty"`
	ResyncPeriod            string                `json:"resyncPeriod,omitempty"`
	Storage                 *Storage              `json:"storage,omitempty"`
	Tolerations             []v1.Toleration       `json:"tolerations,omitempty"`
	Affinity                *v1.Affinity          `json:"affinity,omitempty"`
	SelfContained           *SelfContained        `json:"selfContained,omitempty"`
	DescopedMode            *DescopedMode         `json:"descopedMode,omitempty"`
	Retention               string                `json:"retention,omitempty"`
	AlertManagerDefaultName string                `json:"alertManagerDefaultName,omitempty"`
	PrometheusDefaultName   string                `json:"prometheusDefaultName,omitempty"`
	GrafanaDefaultName      string                `json:"grafanaDefaultName,omitempty"`
}

type DescopedMode struct {
	Enabled                     *bool  `json:"enabled,omitempty"`
	PrometheusOperatorNamespace string `json:"prometheusOperatorNamespace,omitempty"`
}

// ObservabilityStatus defines the observed state of Observability
type ObservabilityStatus struct {
	Stage          ObservabilityStageName   `json:"stage"`
	StageStatus    ObservabilityStageStatus `json:"stageStatus"`
	LastMessage    string                   `json:"lastMessage,omitempty"`
	TokenExpires   int64                    `json:"tokenExpires,omitempty"`
	ClusterID      string                   `json:"clusterId,omitempty"`
	LastSynced     int64                    `json:"lastSynced,omitempty"`
	Migrated       bool                     `json:"migrated,omitempty"`
	ResourcesRoute string                   `json:"resourcesRoute,omitempty"`
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

func (in *Observability) ExternalSyncDisabled() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.DisableRepoSync != nil && *in.Spec.SelfContained.DisableRepoSync
}

func (in *Observability) OverrideSelectors() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.OverrideSelectors != nil && *in.Spec.SelfContained.OverrideSelectors
}

func (in *Observability) ObservatoriumDisabled() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.DisableObservatorium != nil && *in.Spec.SelfContained.DisableObservatorium
}

func (in *Observability) PagerDutyDisabled() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.DisablePagerDuty != nil && *in.Spec.SelfContained.DisablePagerDuty
}

func (in *Observability) DeadMansSnitchDisabled() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.DisableDeadmansSnitch != nil && *in.Spec.SelfContained.DisableDeadmansSnitch
}

func (in *Observability) SmtpDisabled() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.DisableSmtp != nil && *in.Spec.SelfContained.DisableSmtp
}

func (in *Observability) BlackboxExporterDisabled() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.DisableBlackboxExporter != nil && *in.Spec.SelfContained.DisableBlackboxExporter
}

func (in *Observability) SelfSignedCerts() bool {
	return in.Spec.SelfContained != nil && in.Spec.SelfContained.SelfSignedCerts != nil && *in.Spec.SelfContained.SelfSignedCerts
}

func (in *Observability) HasAlertmanagerConfigSecret() (bool, string) {
	if in.Spec.SelfContained != nil && in.Spec.SelfContained.AlertManagerConfigSecret != "" {
		return true, in.Spec.SelfContained.AlertManagerConfigSecret
	}

	return false, ""
}

func (in *Observability) HasBlackboxBearerTokenSecret() (bool, string) {
	if in.Spec.SelfContained != nil && in.Spec.SelfContained.BlackboxBearerTokenSecret != "" {
		return true, in.Spec.SelfContained.BlackboxBearerTokenSecret
	}

	return false, ""
}

func (in *Observability) DescopedModeEnabled() bool {
	if in.Spec.DescopedMode != nil && in.Spec.DescopedMode.Enabled != nil {
		return *in.Spec.DescopedMode.Enabled
	}

	return false
}

func (in *Observability) GetPrometheusOperatorNamespace() string {
	if in.DescopedModeEnabled() && in.Spec.DescopedMode.PrometheusOperatorNamespace != "" {
		return in.Spec.DescopedMode.PrometheusOperatorNamespace
	}

	return in.Namespace
}

func init() {
	SchemeBuilder.Register(&Observability{}, &ObservabilityList{})
}
