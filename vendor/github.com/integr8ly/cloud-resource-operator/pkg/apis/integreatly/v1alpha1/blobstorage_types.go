package v1alpha1

import (
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BlobStorageSpec defines the desired state of BlobStorage
// +k8s:openapi-gen=true
type BlobStorageSpec types.ResourceTypeSpec

// BlobStorageStatus defines the observed state of BlobStorage
// +k8s:openapi-gen=true
type BlobStorageStatus types.ResourceTypeStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlobStorage is the Schema for the blobstorages API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type BlobStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlobStorageSpec   `json:"spec,omitempty"`
	Status BlobStorageStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlobStorageList contains a list of BlobStorage
type BlobStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlobStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlobStorage{}, &BlobStorageList{})
}
