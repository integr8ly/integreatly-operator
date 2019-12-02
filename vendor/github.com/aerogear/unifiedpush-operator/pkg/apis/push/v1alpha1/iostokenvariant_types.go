package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IOSTokenVariantSpec defines the desired state of IOSTokenVariant
// +k8s:openapi-gen=true
type IOSTokenVariantSpec struct {

	// Description is a human friendly description for the variant.
	Description string `json:"description,omitempty"`

	// TeamId is used if you are using APNs tokens as opposed
	// to certificates.  Get this value from your APNS console.
	TeamId string `json:"teamId"`

	// KeyId is used if you are using APNs tokens as opposed
	// to certificates.  Get this value from your APNS console.
	KeyId string `json:"keyId"`

	// BundleId is used if you are using APNs tokens as opposed
	// to certificates.  Get this value from your APNS console.
	BundleId string `json:"bundleId"`

	// PrivateKey is used if you are using APNs tokens as opposed
	// to certificates.  Get this value from your APNS console,
	// and ensure it is in p8 format
	PrivateKey string `json:"privateKey"`

	// Production defines if a connection to production APNS
	// server should be used. If false, a connection to Apple's
	// Sandbox/Development APNs server will be established for
	// this iOS variant.
	Production bool `json:"production"`

	// PushApplicationId is the Id of the Application that this
	// Variant corresponds to in the UnifiedPush Server admin UI.
	PushApplicationId string `json:"pushApplicationId"`
}

// IOSTokenVariantStatus defines the observed state of IOSTokenVariant
// +k8s:openapi-gen=true
type IOSTokenVariantStatus struct {
	Ready     bool   `json:"ready"`
	VariantId string `json:"variantId,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IOSTokenVariant is the Schema for the iostokenvariants API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Production",type="boolean",JSONPath=".spec.production"
// +kubebuilder:printcolumn:name="VariantId",type="string",JSONPath=".status.variantId"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready"
// +kubebuilder:subresource:status
type IOSTokenVariant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IOSTokenVariantSpec   `json:"spec,omitempty"`
	Status IOSTokenVariantStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IOSTokenVariantList contains a list of IOSTokenVariant
type IOSTokenVariantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IOSTokenVariant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IOSTokenVariant{}, &IOSTokenVariantList{})
}
