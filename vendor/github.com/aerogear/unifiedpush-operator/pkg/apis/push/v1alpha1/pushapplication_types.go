package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PushApplicationSpec defines the desired state of PushApplication
// +k8s:openapi-gen=true
type PushApplicationSpec struct {

	// Description is a description of the app to be displayed in
	// the UnifiedPush Server admin UI
	Description string `json:"description"`
}

// PushApplicationStatus defines the observed state of PushApplication
// +k8s:openapi-gen=true
type PushApplicationStatus struct {
	// PushApplicationId is an identifer used to register Variants
	// with this PushApplication
	PushApplicationId string `json:"pushApplicationId"`

	// MasterSecret is a master password, used for sending message
	// to this PushApplication, or it's Variant(s)
	MasterSecret string `json:"masterSecret"`

	// Variants is a slice of variant (AndroidVariant or
	// IOSVariant, in this package) names associated with this
	// Application
	Variants []string `json:"variants,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PushApplication is the Schema for the pushapplications API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PushApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PushApplicationSpec   `json:"spec,omitempty"`
	Status PushApplicationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PushApplicationList contains a list of PushApplication
type PushApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PushApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PushApplication{}, &PushApplicationList{})
}
