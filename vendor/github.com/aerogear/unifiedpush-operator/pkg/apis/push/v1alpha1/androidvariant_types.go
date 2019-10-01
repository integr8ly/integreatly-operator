package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AndroidVariantSpec defines the desired state of AndroidVariant
// +k8s:openapi-gen=true
type AndroidVariantSpec struct {

	// Description is a human friendly description for the
	// variant.
	Description string `json:"description,omitempty"`

	// ServerKey is the key from the Firebase Console of a project
	// which has been enabled for FCM.
	ServerKey string `json:"serverKey"`

	// SenderId is the "Google Project Number" from the API
	// Console. It is *not* needed for sending push messages, but
	// it is a convenience to "see" it on the UnifiedPush Server
	// Admin UI as well, since the Android applications require it
	// (called Sender ID there). That way all information is
	// stored on the same object.
	SenderId string `json:"senderId"`

	// PushApplicationId is the Id of the Application that this
	// Variant corresponds to in the UnifiedPush Server admin UI.
	PushApplicationId string `json:"pushApplicationId"`
}

// AndroidVariantStatus defines the observed state of AndroidVariant
// +k8s:openapi-gen=true
type AndroidVariantStatus struct {
	Ready     bool   `json:"ready"`
	VariantId string `json:"variantId,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AndroidVariant is the Schema for the androidvariants API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="VariantId",type="string",JSONPath=".status.variantId"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready"
// +kubebuilder:subresource:status
type AndroidVariant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AndroidVariantSpec   `json:"spec,omitempty"`
	Status AndroidVariantStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AndroidVariantList contains a list of AndroidVariant
type AndroidVariantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AndroidVariant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AndroidVariant{}, &AndroidVariantList{})
}
