package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileDeveloperConsoleSpec defines the desired state of MobileDeveloperConsole
// +k8s:openapi-gen=true
type MobileDeveloperConsoleSpec struct {
	// OAuthClientId is the id of the OAuthClient to use when protecting the Mobile Developer Console
	// instance with OpenShift OAuth Proxy.
	OAuthClientId string `json:"oAuthClientId"`

	// OAuthClientSecret is the secret of the OAuthClient to use when protecting the Mobile Developer Console
	// instance with OpenShift OAuth Proxy.
	OAuthClientSecret string `json:"oAuthClientSecret"`
}

// MobileDeveloperConsoleStatus defines the observed state of MobileDeveloperConsole
// +k8s:openapi-gen=true
type MobileDeveloperConsoleStatus struct {
	Phase StatusPhase `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileDeveloperConsole is the Schema for the mobiledeveloperconsoles API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=mobiledeveloperconsoles,shortName=mdc
// +kubebuilder:singular=mobiledeveloperconsole
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:subresource:status
type MobileDeveloperConsole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileDeveloperConsoleSpec   `json:"spec,omitempty"`
	Status MobileDeveloperConsoleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileDeveloperConsoleList contains a list of MobileDeveloperConsole
type MobileDeveloperConsoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileDeveloperConsole `json:"items"`
}

type StatusPhase string

var (
	PhaseEmpty     StatusPhase = ""
	PhaseComplete  StatusPhase = "Complete"
	PhaseProvision StatusPhase = "Provisioning"
)

func init() {
	SchemeBuilder.Register(&MobileDeveloperConsole{}, &MobileDeveloperConsoleList{})
}
