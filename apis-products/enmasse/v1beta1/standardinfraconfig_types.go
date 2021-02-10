// +kubebuilder:object:generate=false
// +kubebuilder:skip
// +kubebuilder:skipversion
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StandardInfraConfigSpec defines the desired state of StandardInfraConfig
type StandardInfraConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Admin  InfraConfigAdmin  `json:"admin"`
	Broker InfraConfigBroker `json:"broker"`
	Router InfraConfigRouter `json:"router"`
}

// StandardInfraConfigStatus defines the observed state of StandardInfraConfig
type StandardInfraConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// StandardInfraConfig is the Schema for the standardinfraconfigs API
// +k8s:openapi-gen=true
type StandardInfraConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandardInfraConfigSpec   `json:"spec,omitempty"`
	Status StandardInfraConfigStatus `json:"status,omitempty"`
}

// StandardInfraConfigList contains a list of StandardInfraConfig
type StandardInfraConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StandardInfraConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StandardInfraConfig{}, &StandardInfraConfigList{})
}
