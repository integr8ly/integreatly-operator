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
	PhaseNone                 StatusPhase = ""
	PhaseAccepted             StatusPhase = "accepted"
	PhaseCreatingSubscription StatusPhase = "creating subscription"
	PhaseAwaitingOperator     StatusPhase = "awaiting operator"
	PhaseCreatingComponents   StatusPhase = "creating components"
	PhaseAwaitingComponents   StatusPhase = "awaiting components"
	PhaseInProgress           StatusPhase = "in progress"
	PhaseCompleted            StatusPhase = "completed"
	PhaseFailed               StatusPhase = "failed"

	InstallationTypeWorkshop InstallationType = "workshop"
	InstallationTypeManaged  InstallationType = "managed"

	BootstrapStage        StageName = "bootstrap"
	CloudResourcesStage   StageName = "cloud-resources"
	MonitoringStage       StageName = "monitoring"
	AuthenticationStage   StageName = "authentication"
	ProductsStage         StageName = "products"
	SolutionExplorerStage StageName = "solution-explorer"

	ProductAMQStreams             ProductName = "amqstreams"
	ProductAMQOnline              ProductName = "amqonline"
	ProductSolutionExplorer       ProductName = "solution-explorer"
	ProductRHSSO                  ProductName = "rhsso"
	ProductRHSSOUser              ProductName = "rhssouser"
	ProductCodeReadyWorkspaces    ProductName = "codeready-workspaces"
	ProductFuse                   ProductName = "fuse"
	ProductFuseOnOpenshift        ProductName = "fuse-on-openshift"
	Product3Scale                 ProductName = "3scale"
	ProductNexus                  ProductName = "nexus"
	ProductLauncher               ProductName = "launcher"
	ProductUps                    ProductName = "ups"
	ProductMonitoring             ProductName = "monitoring"
	ProductMobileSecurityService  ProductName = "mobilesecurityservice"
	ProductMobileDeveloperConsole ProductName = "mdc"
	ProductCloudResources         ProductName = "cloud-resources"

	// PBrookes 08/08/2019:
	// Could not find a way to determine these versions dynamically, so they are hard-coded
	// It is preferable to determine the version of a product dynamically (from a CR, or configmap, etc)
	VersionAMQOnline             ProductVersion = "1.2.2"
	VersionAMQStreams            ProductVersion = "1.1.0"
	VersionCodeReadyWorkspaces   ProductVersion = "1.2.0.GA"
	VersionFuseOnOpenshift       ProductVersion = "master"
	VersionMonitoring            ProductVersion = "0.0.28"
	VersionNexus                 ProductVersion = "3.16"
	VersionLauncher              ProductVersion = "0.1.2"
	VersionUps                   ProductVersion = "2.3.2"
	VersionMobileSecurityService ProductVersion = "0.2.2"
	VersionCloudResources        ProductVersion = "0.1.0"

	PreflightInProgress PreflightStatus = ""
	PreflightSuccess    PreflightStatus = "successful"
	PreflightFail       PreflightStatus = "failed"

	OperatorVersionAMQStreams            OperatorVersion = "1.1.0"
	OperatorVersionAMQOnline             OperatorVersion = "1.2.2"
	OperatorVersionMonitoring            OperatorVersion = "0.0.28"
	OperatorVersionSolutionExplorer      OperatorVersion = "0.0.34"
	OperatorVersionRHSSO                 OperatorVersion = "1.9.5"
	OperatorVersionRHSSOUser             OperatorVersion = "1.9.5"
	OperatorVersionCodeReadyWorkspaces   OperatorVersion = "1.2.2"
	OperatorVersionFuse                  OperatorVersion = "1.4.0"
	OperatorVersion3Scale                OperatorVersion = "1.9.8"
	OperatorVersionNexus                 OperatorVersion = "0.9.0"
	OperatorVersionLauncher              OperatorVersion = "0.1.2"
	OperatorVersionUPS                   OperatorVersion = "0.3.0"
	OperatorVersionMobileSecurityService OperatorVersion = "0.4.1"
	OperatorVersionMDC                   OperatorVersion = "0.3.0"
	OperatorVersionCloudResources        OperatorVersion = "0.1.0"
)

// InstallationSpec defines the desired state of Installation
// +k8s:openapi-gen=true
type InstallationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Type             string         `json:"type"`
	RoutingSubdomain string         `json:"routingSubdomain,omitempty"`
	MasterURL        string         `json:"masterURL,omitempty"`
	NamespacePrefix  string         `json:"namespacePrefix"`
	SelfSignedCerts  bool           `json:"selfSignedCerts"`
	PullSecret       PullSecretSpec `json:"pullSecret,omitempty"`
}

type PullSecretSpec struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// InstallationStatus defines the observed state of Installation
// +k8s:openapi-gen=true
type InstallationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Stages           map[StageName]*InstallationStageStatus `json:"stages"`
	PreflightStatus  PreflightStatus                        `json:"preflightStatus,omitempty"`
	PreflightMessage string                                 `json:"preflightMessage,omitempty"`
}

type InstallationStageStatus struct {
	Name     StageName                                  `json:"name"`
	Phase    StatusPhase                                `json:"phase"`
	Products map[ProductName]*InstallationProductStatus `json:"products,omitempty"`
}

type InstallationProductStatus struct {
	Name            ProductName     `json:"name"`
	OperatorVersion OperatorVersion `json:"operator,omitempty"`
	Version         ProductVersion  `json:"version"`
	Host            string          `json:"host"`
	Type            string          `json:"type,omitempty"`
	Mobile          bool            `json:"mobile,omitempty"`
	Status          StatusPhase     `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Installation is the Schema for the installations API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=installations,scope=Namespaced
type Installation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallationSpec   `json:"spec,omitempty"`
	Status InstallationStatus `json:"status,omitempty"`
}

func (i *Installation) GetProductStatusObject(product ProductName) *InstallationProductStatus {
	for _, stage := range i.Status.Stages {
		if product, ok := stage.Products[product]; ok {
			return product
		}
	}
	return &InstallationProductStatus{
		Name: product,
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallationList contains a list of Installation
type InstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Installation{}, &InstallationList{})
}
