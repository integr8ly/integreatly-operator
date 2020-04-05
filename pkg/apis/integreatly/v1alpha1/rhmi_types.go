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
	PhaseAccepted               StatusPhase = "accepted"
	PhaseCreatingSubscription   StatusPhase = "creating subscription"
	PhaseAwaitingOperator       StatusPhase = "awaiting operator"
	PhaseAwaitingCloudResources StatusPhase = "awaiting cloud resources"
	PhaseCreatingComponents     StatusPhase = "creating components"
	PhaseAwaitingComponents     StatusPhase = "awaiting components"

	PhaseInProgress StatusPhase = "in progress"
	PhaseCompleted  StatusPhase = "completed"
	PhaseFailed     StatusPhase = "failed"

	InstallationTypeWorkshop InstallationType = "workshop"
	InstallationTypeManaged  InstallationType = "managed"

	BootstrapStage        StageName = "bootstrap"
	CloudResourcesStage   StageName = "cloud-resources"
	MonitoringStage       StageName = "monitoring"
	AuthenticationStage   StageName = "authentication"
	ProductsStage         StageName = "products"
	SolutionExplorerStage StageName = "solution-explorer"

	ProductAMQStreams          ProductName = "amqstreams"
	ProductAMQOnline           ProductName = "amqonline"
	ProductSolutionExplorer    ProductName = "solution-explorer"
	ProductRHSSO               ProductName = "rhsso"
	ProductRHSSOUser           ProductName = "rhssouser"
	ProductCodeReadyWorkspaces ProductName = "codeready-workspaces"
	ProductFuse                ProductName = "fuse"
	ProductFuseOnOpenshift     ProductName = "fuse-on-openshift"
	Product3Scale              ProductName = "3scale"
	ProductUps                 ProductName = "ups"
	ProductApicurito           ProductName = "apicurito"
	ProductMonitoring          ProductName = "middleware-monitoring"
	ProductCloudResources      ProductName = "cloud-resources"
	ProductDataSync            ProductName = "datasync"

	// Could not find a way to determine these versions dynamically, so they are hard-coded
	// It is preferable to determine the version of a product dynamically (from a CR, or configmap, etc)
	// Follow up Jira: https://issues.redhat.com/browse/INTLY-5946
	VersionAMQOnline           ProductVersion = "1.4"
	VersionApicurito           ProductVersion = "7.5"
	VersionAMQStreams          ProductVersion = "1.1.0"
	VersionCodeReadyWorkspaces ProductVersion = "2.0.0"
	VersionFuseOnOpenshift     ProductVersion = "master"
	VersionMonitoring          ProductVersion = "1.1.4"
	Version3Scale              ProductVersion = "2.7"
	VersionUps                 ProductVersion = "2.3.2"
	VersionCloudResources      ProductVersion = "0.13.1"
	VersionFuseOnline          ProductVersion = "7.5"
	VersionDataSync            ProductVersion = "0.9.4"
	VersionRHSSO               ProductVersion = "7.3"
	VersionRHSSOUser           ProductVersion = "7.3"

	PreflightInProgress PreflightStatus = ""
	PreflightSuccess    PreflightStatus = "successful"
	PreflightFail       PreflightStatus = "failed"

	// Operator image tags
	OperatorVersionAMQStreams          OperatorVersion = "1.1.0"
	OperatorVersionAMQOnline           OperatorVersion = "1.4"
	OperatorVersionMonitoring          OperatorVersion = "1.1.4"
	OperatorVersionSolutionExplorer    OperatorVersion = "0.0.46"
	OperatorVersionRHSSO               OperatorVersion = "8.0.1"
	OperatorVersionRHSSOUser           OperatorVersion = "8.0.1"
	OperatorVersionCodeReadyWorkspaces OperatorVersion = "2.0"
	OperatorVersionFuse                OperatorVersion = "1.5.0"
	OperatorVersion3Scale              OperatorVersion = "0.4.0"
	OperatorVersionCloudResources      OperatorVersion = "0.13.1"
	OperatorVersionUPS                 OperatorVersion = "0.5.0"
	OperatorVersionApicurito           OperatorVersion = "1.5.0"

	// Event reasons to be used when emitting events
	EventProcessingError       string = "ProcessingError"
	EventInstallationCompleted string = "InstallationCompleted"
	EventPreflightCheckPassed  string = "PreflightCheckPassed"
)

// RHMISpec defines the desired state of Installation
// +k8s:openapi-gen=true
type RHMISpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Type              string         `json:"type"`
	RoutingSubdomain  string         `json:"routingSubdomain,omitempty"`
	MasterURL         string         `json:"masterURL,omitempty"`
	NamespacePrefix   string         `json:"namespacePrefix"`
	SelfSignedCerts   bool           `json:"selfSignedCerts,omitempty"`
	PullSecret        PullSecretSpec `json:"pullSecret,omitempty"`
	UseClusterStorage string         `json:"useClusterStorage,omitempty"`

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

// RHMIStatus defines the observed state of Installation
// +k8s:openapi-gen=true
type RHMIStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Stages             map[StageName]RHMIStageStatus `json:"stages"`
	Stage              StageName                     `json:"stage"`
	PreflightStatus    PreflightStatus               `json:"preflightStatus,omitempty"`
	PreflightMessage   string                        `json:"preflightMessage,omitempty"`
	LastError          string                        `json:"lastError"`
	GitHubOAuthEnabled bool                          `json:"gitHubOAuthEnabled,omitempty"`
	SMTPEnabled        bool                          `json:"smtpEnabled,omitempty"`
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
	Status          StatusPhase     `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHMI is the Schema for the RHMI API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rhmis,scope=Namespaced
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHMIList contains a list of Installation
type RHMIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHMI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RHMI{}, &RHMIList{})
}
