package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AddressPlanSpec defines the desired state of AddressPlan
type AddressPlanSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	DisplayName      string               `json:"displayName"`
	DisplayOrder     int                  `json:"displayOrder"`
	ShortDescription string               `json:"shortDescription"`
	LongDescription  string               `json:"longDescription"`
	AddressType      string               `json:"addressType"`
	Resources        AddressPlanResources `json:"resources"`
	Partitions       int                  `json:"partitions"`
}

// AddressPlanStatus defines the observed state of AddressPlan
type AddressPlanStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

type AddressPlanResources struct {
	Router string `json:"router"`
	Broker string `json:"broker"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddressPlan is the Schema for the addressplans API
// +k8s:openapi-gen=true
type AddressPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddressPlanSpec   `json:"spec,omitempty"`
	Status AddressPlanStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddressPlanList contains a list of AddressPlan
type AddressPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddressPlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AddressPlan{}, &AddressPlanList{})
}
