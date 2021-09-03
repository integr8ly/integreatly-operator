package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApicuritoSpec defines the desired state of Apicurito
// +k8s:openapi-gen=true
type ApicuritoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Size int32 `json:"size"`

	// +kubebuilder:validation:Pattern=.+:.+
	Image string `json:"image"`
}

// ApicuritoStatus defines the observed state of Apicurito
// +k8s:openapi-gen=true
type ApicuritoStatus struct {
	// Nodes are the names of the apicurito pods
	// +listType=set
	Nodes []string `json:"nodes"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Apicurito is the Schema for the apicuritos API
// +k8s:openapi-gen=true
type Apicurito struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApicuritoSpec   `json:"spec,omitempty"`
	Status ApicuritoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApicuritoList contains a list of Apicurito
type ApicuritoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Apicurito `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Apicurito{}, &ApicuritoList{})
}
