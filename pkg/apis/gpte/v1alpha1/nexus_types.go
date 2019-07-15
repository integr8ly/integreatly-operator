package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NexusSpec defines the desired state of Nexus
// +k8s:openapi-gen=true
type NexusSpec struct {
	NexusVolumeSize    string `json:"nexusVolumeSize,omitempty"`
	NexusSSL           bool   `json:"nexusSsl,omitempty"`
	NexusImageTag      string `json:"nexusImageTag,omitempty"`
	NexusCPURequest    int    `json:"nexusCpuRequest,omitempty"`
	NexusCPULimit      int    `json:"nexusCpuLimit,omitempty"`
	NexusMemoryRequest string `json:"nexusMemoryRequest,omitempty"`
	NexusMemoryLimit   string `json:"nexusMemoryLimit,omitempty"`
}

// NexusStatus defines the observed state of Nexus
// +k8s:openapi-gen=true
type NexusStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Nexus is the Schema for the nexus API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Nexus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusSpec   `json:"spec,omitempty"`
	Status NexusStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NexusList contains a list of Nexus
type NexusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nexus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nexus{}, &NexusList{})
}
