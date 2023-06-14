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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StatusPhase string
type InstallationType string
type ProductName string
type ProductVersion string
type OperatorVersion string
type PreflightStatus string
type StageName string

var (
	PhaseNone                   StatusPhase = ""
	PhaseAwaitingOperator       StatusPhase = "awaiting operator"
	PhaseAwaitingCloudResources StatusPhase = "awaiting cloud resources"
	PhaseCreatingComponents     StatusPhase = "creating components"
	PhaseAwaitingComponents     StatusPhase = "awaiting components"

	PhaseInProgress StatusPhase = "in progress"
	PhaseCompleted  StatusPhase = "completed"
	PhaseFailed     StatusPhase = "failed"

	InstallationTypeManagedApi            InstallationType = "managed-api"
	InstallationTypeMultitenantManagedApi InstallationType = "multitenant-managed-api"

	BootstrapStage               StageName = "bootstrap"
	InstallStage                 StageName = "installation"
	CloudResourcesStage          StageName = "cloud-resources"
	MonitoringStage              StageName = "monitoring"
	ProductsStage                StageName = "products"
	CompleteStage                StageName = "complete"
	UninstallProductsStage       StageName = "uninstall - products"
	UninstallCloudResourcesStage StageName = "uninstall - cloud-resources"
	UninstallBootstrap           StageName = "uninstall - bootstrap"

	ProductRHSSO          ProductName = "rhsso"
	ProductRHSSOUser      ProductName = "rhssouser"
	Product3Scale         ProductName = "3scale"
	ProductMonitoring     ProductName = "middleware-monitoring"
	ProductObservability  ProductName = "observability" // TODO MGDAPI-5833
	ProductCloudResources ProductName = "cloud-resources"
	ProductMonitoringSpec ProductName = "monitoring-spec"
	ProductMarin3r        ProductName = "marin3r"
	ProductGrafana        ProductName = "grafana"
	ProductMCG            ProductName = "mcg"

	// Could not find a way to determine these versions dynamically, so they are hard-coded
	// It is preferable to determine the version of a product dynamically (from a CR, or configmap, etc)
	// Follow up Jira: https://issues.redhat.com/browse/INTLY-5946
	VersionMonitoring     ProductVersion = "1.8.0"
	Version3Scale         ProductVersion = "2.13.0"
	VersionCloudResources ProductVersion = "1.1.0"
	VersionRHSSO          ProductVersion = "7.6"
	VersionRHSSOUser      ProductVersion = "7.6"
	VersionMonitoringSpec ProductVersion = "1.0"
	VersionMarin3r        ProductVersion = "0.11.0"
	VersionMCG            ProductVersion = "4.12.3-rhodf"
	VersionGrafana        ProductVersion = "4.2.0"
	VersionObservability  ProductVersion = "4.2.1"

	PreflightInProgress PreflightStatus = ""
	PreflightSuccess    PreflightStatus = "successful"
	PreflightFail       PreflightStatus = "failed"

	// Operator image tags
	OperatorVersionMonitoring OperatorVersion = "1.8.0"
	OperatorVersionRHSSO      OperatorVersion = "7.6.3-1"
	OperatorVersionRHSSOUser  OperatorVersion = "7.6.3-1"

	OperatorVersionCloudResources OperatorVersion = "1.1.0"
	OperatorVersion3Scale         OperatorVersion = "0.11.5-mas"
	OperatorVersionMonitoringSpec OperatorVersion = "1.0"
	OperatorVersionMarin3r        OperatorVersion = "0.11.0"
	OperatorVersionGrafana        OperatorVersion = "4.2.0"
	OperatorVersionObservability  OperatorVersion = "4.2.1"
	OperatorVersionMCG            OperatorVersion = "4.12.3-rhodf"

	// Event reasons to be used when emitting events
	EventProcessingError       string = "ProcessingError"
	EventInstallationCompleted string = "InstallationCompleted"
	EventPreflightCheckPassed  string = "PreflightCheckPassed"
	EventUpgradeApproved       string = "UpgradeApproved"

	DefaultOriginPullSecretName      = "pull-secret"
	DefaultOriginPullSecretNamespace = "openshift-config" // #nosec G101 -- This is a false positive

	EnvKeyAlertSMTPFrom = "ALERT_SMTP_FROM"
	EnvKeyQuota         = "QUOTA"
)

// RHMISpec defines the desired state of RHMI
type RHMISpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Type                   string                 `json:"type"`
	RoutingSubdomain       string                 `json:"routingSubdomain,omitempty"`
	MasterURL              string                 `json:"masterURL,omitempty"`
	NamespacePrefix        string                 `json:"namespacePrefix"`
	RebalancePods          bool                   `json:"rebalancePods,omitempty"`
	SelfSignedCerts        bool                   `json:"selfSignedCerts,omitempty"`
	PullSecret             PullSecretSpec         `json:"pullSecret,omitempty"`
	UseClusterStorage      string                 `json:"useClusterStorage,omitempty"`
	AlertingEmailAddress   string                 `json:"alertingEmailAddress,omitempty"`
	PriorityClassName      string                 `json:"priorityClassName,omitempty"`
	AlertingEmailAddresses AlertingEmailAddresses `json:"alertingEmailAddresses,omitempty"`
	AlertFromAddress       string                 `json:"alertFromAddress,omitempty"`
	APIServer              string                 `json:"APIServer,omitempty"`

	// OperatorsInProductNamespace is a flag that decides if
	// the product operators should be installed in the product
	// namespace (when set to true) or in standalone namespace
	// (when set to false, default). Standalone namespace will
	// be used only for those operators that support it.
	OperatorsInProductNamespace bool `json:"operatorsInProductNamespace,omitempty"`

	// SMTPSecret is the name of a secret in the installation
	// namespace containing SMTP connection details. The secret
	// must contain the following fields:
	//
	// host
	// port
	// tls
	// username
	// password
	SMTPSecret string `json:"smtpSecret,omitempty"`

	// PagerDutySecret is the name of a secret in the
	// installation namespace containing PagerDuty account
	// details. The secret must contain the following fields:
	//
	// serviceKey
	PagerDutySecret string `json:"pagerDutySecret,omitempty"`

	// DeadMansSnitchSecret is the name of a secret in the
	// installation namespace containing connection details
	// for Dead Mans Snitch. The secret must contain the
	// following fields:
	//
	// url
	DeadMansSnitchSecret string `json:"deadMansSnitchSecret,omitempty"`
}

type PullSecretSpec struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type AlertingEmailAddresses struct {
	BusinessUnit string `json:"businessUnit"`
	CSSRE        string `json:"cssre"`
}

type CustomSmtpStatus struct {
	Enabled bool   `json:"enabled"`
	Error   string `json:"error,omitempty"`
}

type CustomDomainStatus struct {
	Enabled bool   `json:"enabled"`
	Error   string `json:"error,omitempty"`
}

// RHMIStatus defines the observed state of RHMI
type RHMIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Stages             map[StageName]RHMIStageStatus `json:"stages"`
	Stage              StageName                     `json:"stage"`
	PreflightStatus    PreflightStatus               `json:"preflightStatus,omitempty"`
	PreflightMessage   string                        `json:"preflightMessage,omitempty"`
	LastError          string                        `json:"lastError"`
	GitHubOAuthEnabled bool                          `json:"gitHubOAuthEnabled,omitempty"`
	SMTPEnabled        bool                          `json:"smtpEnabled,omitempty"`
	Version            string                        `json:"version,omitempty"`
	ToVersion          string                        `json:"toVersion,omitempty"`
	Quota              string                        `json:"quota,omitempty"`
	ToQuota            string                        `json:"toQuota,omitempty"`
	CustomSmtp         *CustomSmtpStatus             `json:"customSmtp,omitempty"`
	CustomDomain       *CustomDomainStatus           `json:"customDomain,omitempty"`
}

type RHMIStageStatus struct {
	Name     StageName                         `json:"name"`
	Phase    StatusPhase                       `json:"phase"`
	Products map[ProductName]RHMIProductStatus `json:"products,omitempty"`
}

type RHMIProductStatus struct {
	Name            ProductName     `json:"name"`
	OperatorVersion OperatorVersion `json:"operator,omitempty"`
	Version         ProductVersion  `json:"version"`
	Host            string          `json:"host"`
	Type            string          `json:"type,omitempty"`
	Mobile          bool            `json:"mobile,omitempty"`
	Phase           StatusPhase     `json:"status"`
	Uninstall       bool            `json:"uninstall,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RHMI is the Schema for the rhmis API
type RHMI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RHMISpec   `json:"spec,omitempty"`
	Status RHMIStatus `json:"status,omitempty"`
}

func (i *RHMI) GetProductStatusObject(product ProductName) *RHMIProductStatus {
	for _, stage := range i.Status.Stages {
		if product, ok := stage.Products[product]; ok {
			return &product
		}
	}
	return &RHMIProductStatus{
		Name: product,
	}
}

func (i *RHMI) GetPullSecretSpec() *PullSecretSpec {
	if i.Spec.PullSecret.Name != "" && i.Spec.PullSecret.Namespace != "" {
		return &(i.Spec.PullSecret)
	} else {
		return &PullSecretSpec{Name: DefaultOriginPullSecretName, Namespace: DefaultOriginPullSecretNamespace}
	}
}

// +kubebuilder:object:root=true

// RHMIList contains a list of RHMI
type RHMIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHMI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RHMI{}, &RHMIList{})
}
