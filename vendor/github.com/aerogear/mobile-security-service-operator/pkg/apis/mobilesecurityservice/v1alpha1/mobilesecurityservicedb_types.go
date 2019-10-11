package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/extensions/v1beta1"

)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileSecurityServiceDBSpec defines the desired state of MobileSecurityServiceDB
// +k8s:openapi-gen=true
type MobileSecurityServiceDBSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Size                   int32  `json:"size"`
	Image                  string `json:"image"`
	ContainerName          string `json:"containerName"`
	DatabaseName           string `json:"databaseName,omitempty"`
	DatabasePassword       string `json:"databasePassword,omitempty"`
	DatabaseUser           string `json:"databaseUser,omitempty"`
	DatabaseNameParam      string `json:"databaseNameParam"`
	DatabasePasswordParam  string `json:"databasePasswordParam"`
	DatabaseUserParam      string `json:"databaseUserParam"`
	DatabasePort           int32  `json:"databasePort"`
	DatabaseMemoryLimit    string `json:"databaseMemoryLimit"`
	DatabaseMemoryRequest  string `json:"databaseMemoryRequest"`
	DatabaseStorageRequest string `json:"databaseStorageRequest"`
	ConfigMapName          string `json:"configMapName,omitempty"`
}

// MobileSecurityServiceDBStatus defines the observed state of MobileSecurityServiceDB
// +k8s:openapi-gen=true
type MobileSecurityServiceDBStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	PersistentVolumeClaimName string `json:"persistentVolumeClaimName"`
	DeploymentName string `json:"deploymentName"`
	DeploymentStatus v1beta1.DeploymentStatus `json:"deploymentStatus"`
	ServiceName string `json:"serviceName"`
	ServiceStatus v1.ServiceStatus `json:"serviceStatus"`
	DatabaseStatus string `json:databaseStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceDB is the Schema for the mobilesecurityservicedbs API
// +k8s:openapi-gen=true
type MobileSecurityServiceDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileSecurityServiceDBSpec   `json:"spec,omitempty"`
	Status MobileSecurityServiceDBStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceDBList contains a list of MobileSecurityServiceDB
type MobileSecurityServiceDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileSecurityServiceDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MobileSecurityServiceDB{}, &MobileSecurityServiceDBList{})
}
