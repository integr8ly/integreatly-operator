package v1alpha1

import (
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SMTPCredentialsSpec defines the desired state of SMTPCredentials
// +k8s:openapi-gen=true
type SMTPCredentialSetSpec types.ResourceTypeSpec

// SMTPCredentialsStatus defines the observed state of SMTPCredentials
// +k8s:openapi-gen=true
type SMTPCredentialSetStatus types.ResourceTypeStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SMTPCredentials is the Schema for the smtpcredentialset API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SMTPCredentialSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SMTPCredentialSetSpec   `json:"spec,omitempty"`
	Status SMTPCredentialSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SMTPCredentialsList contains a list of SMTPCredentials
type SMTPCredentialSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SMTPCredentialSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SMTPCredentialSet{}, &SMTPCredentialSetList{})
}
