package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileSecurityServiceUnbindSpec defines the desired state of MobileSecurityServiceUnbind
// +k8s:openapi-gen=true
type MobileSecurityServiceUnbindSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ClusterHost                   string `json:"clusterHost"`
	HostSufix                     string `json:"hostSufix"`
	Protocol                      string `json:"protocol"`
	AppId                         string `json:"appId"`
}

// MobileSecurityServiceUnbindStatus defines the observed state of MobileSecurityServiceUnbind
// +k8s:openapi-gen=true
type MobileSecurityServiceUnbindStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	UnbindStatus string `json:"unbindStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceUnbind is the Schema for the mobilesecurityserviceunbinds API
// +k8s:openapi-gen=true
type MobileSecurityServiceUnbind struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileSecurityServiceUnbindSpec   `json:"spec,omitempty"`
	Status MobileSecurityServiceUnbindStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceUnbindList contains a list of MobileSecurityServiceUnbind
type MobileSecurityServiceUnbindList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileSecurityServiceUnbind `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MobileSecurityServiceUnbind{}, &MobileSecurityServiceUnbindList{})
}
