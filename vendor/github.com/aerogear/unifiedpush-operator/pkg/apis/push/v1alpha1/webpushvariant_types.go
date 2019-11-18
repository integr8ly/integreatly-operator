package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WebPushVariantSpec defines the desired state of WebPushVariant
// +k8s:openapi-gen=true
type WebPushVariantSpec struct {

	// Description is a human friendly description for the
	// variant.
	Description string `json:"description,omitempty"`

	// PrivateKey is the private part of a VAPID keypair. It is used
	// to sign messages and identify them as being from this server by the
	// client.
	PrivateKey string `json:"privateKey"`

	// PublicKey is the public part of a VAPID keypair. This will
	// be transmitted to the user in the mobile-services.json file and the client
	// uses it to verify messages were send by this server.
	PublicKey string `json:"publicKey"`

	// Alias is sent to a web push push service and must be either
	// a url or mailto link.  The purpose is to provide a way for the push service
	// to identify the sender of the message.
	Alias string `json:"alias"`

	// PushApplicationId is the Id of the Application that this
	// Variant corresponds to in the UnifiedPush Server admin UI.
	PushApplicationId string `json:"pushApplicationId"`
}

// WebPushVariantStatus defines the observed state of WebPushVariant
// +k8s:openapi-gen=true
type WebPushVariantStatus struct {
	Ready     bool   `json:"ready"`
	VariantId string `json:"variantId,omitempty"`
	Secret    string `json:"secret,omitempty"`
	Message   string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebPushVariant is the Schema for the webpushvariants API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type WebPushVariant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebPushVariantSpec   `json:"spec,omitempty"`
	Status WebPushVariantStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebPushVariantList contains a list of WebPushVariant
type WebPushVariantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebPushVariant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebPushVariant{}, &WebPushVariantList{})
}
