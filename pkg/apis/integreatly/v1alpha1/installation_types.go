package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StatusPhase string
type InstallationType string
type ProductName string
type ProductVersion string

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

	ProductAMQStreams          ProductName = "amqstreams"
	ProductAMQOnline           ProductName = "amqonline"
	ProductSolutionExplorer    ProductName = "solution-explorer"
	ProductRHSSO               ProductName = "rhsso"
	ProductCodeReadyWorkspaces ProductName = "codeready-workspaces"
	ProductFuse                ProductName = "fuse"
	Product3Scale              ProductName = "3scale"
	ProductNexus               ProductName = "nexus"

	VersionAMQStreams          ProductVersion = "1.1"
	VersionAMQOnline           ProductVersion = "1.1"
	VersionSolutionExplorer    ProductVersion = "2.10"
	VersionRHSSO               ProductVersion = "7.3"
	VersionCodeReadyWorkspaces ProductVersion = "1.2"
	VersionFuse                ProductVersion = "1.7"
	Version3Scale              ProductVersion = "2.6"
	VersionNexus               ProductVersion = "3.16"
)

// InstallationSpec defines the desired state of Installation
// +k8s:openapi-gen=true
type InstallationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Type             string `json:"type"`
	RoutingSubdomain string `json:"routingSubdomain"`
	MasterURL        string `json:"masterUrl"`
	NamespacePrefix  string `json:"namespacePrefix"`
	SelfSignedCerts  bool   `json:"selfSignedCerts"`
}

// InstallationStatus defines the observed state of Installation
// +k8s:openapi-gen=true
type InstallationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Stages map[string]*InstallationStageStatus `json:"stages"`
}

type InstallationStageStatus struct {
	Name     string                                     `json:"name"`
	Phase    StatusPhase                                `json:"phase"`
	Products map[ProductName]*InstallationProductStatus `json:"products"`
}

type InstallationProductStatus struct {
	Name    ProductName    `json:"name"`
	Version ProductVersion `json:"version"`
	Host    string         `json:"host"`
	Type    string         `json:"type,omitempty"`
	Mobile  bool           `json:"mobile,omitempty"`
	Status  StatusPhase    `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Installation is the Schema for the installations API
// +k8s:openapi-gen=true
type Installation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallationSpec   `json:"spec,omitempty"`
	Status InstallationStatus `json:"status,omitempty"`
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
