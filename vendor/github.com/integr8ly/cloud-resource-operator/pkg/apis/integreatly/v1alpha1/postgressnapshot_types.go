package v1alpha1

import (
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgresSnapshotSpec defines the desired state of PostgresSnapshot
// +k8s:openapi-gen=true
type PostgresSnapshotSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ResourceName string `json:"resourceName"`
}

// PostgresSnapshotStatus defines the observed state of PostgresSnapshot
// +k8s:openapi-gen=true
type PostgresSnapshotStatus types.ResourceTypeSnapshotStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgresSnapshot is the Schema for the postgressnapshots API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PostgresSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresSnapshotSpec   `json:"spec,omitempty"`
	Status PostgresSnapshotStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgresSnapshotList contains a list of PostgresSnapshot
type PostgresSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresSnapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresSnapshot{}, &PostgresSnapshotList{})
}
