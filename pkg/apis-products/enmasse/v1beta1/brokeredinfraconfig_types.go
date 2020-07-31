package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BrokeredInfraConfigSpec defines the desired state of BrokeredInfraConfig
type BrokeredInfraConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Admin  InfraConfigAdmin  `json:"admin"`
	Broker InfraConfigBroker `json:"broker"`
}

type InfraConfigAdmin struct {
	Resources InfraConfigResources `json:"resources"`
}

type InfraConfigBroker struct {
	Resources         InfraConfigResources `json:"resources"`
	AddressFullPolicy string               `json:"addressFullPolicy"`
	MaxUnavailable    int                  `json:"maxUnavailable,omitempty"`
}

type InfraConfigRouter struct {
	MinReplicas    int                  `json:"minReplicas"`
	Resources      InfraConfigResources `json:"resources"`
	LinkCapacity   int                  `json:"linkCapacity"`
	MaxUnavailable int                  `json:"maxUnavailable,omitempty"`
}

type InfraConfigResources struct {
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
}

// BrokeredInfraConfigStatus defines the observed state of BrokeredInfraConfig
type BrokeredInfraConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BrokeredInfraConfig is the Schema for the brokeredinfraconfigs API
// +k8s:openapi-gen=true
type BrokeredInfraConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BrokeredInfraConfigSpec   `json:"spec,omitempty"`
	Status BrokeredInfraConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BrokeredInfraConfigList contains a list of BrokeredInfraConfig
type BrokeredInfraConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BrokeredInfraConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BrokeredInfraConfig{}, &BrokeredInfraConfigList{})
}
