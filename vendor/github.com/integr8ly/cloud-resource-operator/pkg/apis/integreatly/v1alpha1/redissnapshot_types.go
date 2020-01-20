package v1alpha1

import (
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisSnapshotSpec defines the desired state of RedisSnapshot
// +k8s:openapi-gen=true
type RedisSnapshotSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ResourceName string `json:"resourceName"`
}

// RedisSnapshotStatus defines the observed state of RedisSnapshot
// +k8s:openapi-gen=true
type RedisSnapshotStatus types.ResourceTypeSnapshotStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RedisSnapshot is the Schema for the redissnapshots API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type RedisSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSnapshotSpec   `json:"spec,omitempty"`
	Status RedisSnapshotStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RedisSnapshotList contains a list of RedisSnapshot
type RedisSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisSnapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisSnapshot{}, &RedisSnapshotList{})
}
