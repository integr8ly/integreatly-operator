package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileSecurityServiceAppSpec defines the desired state of MobileSecurityServiceApp
// +k8s:openapi-gen=true
type MobileSecurityServiceAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ClusterHost                   string `json:"clusterHost"`
	HostSufix                     string `json:"hostSufix"`
	Protocol                      string `json:"protocol"`
	AppName                       string `json:"appName"`
	AppId                         string `json:"appId"`

}

// MobileSecurityServiceAppStatus defines the observed state of MobileSecurityServiceApp
// +k8s:openapi-gen=true
type MobileSecurityServiceAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	SDKConfigMapName string `json:"sdkConfigMapName"`
	BindStatus string `json:"bindStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceApp is the Schema for the mobilesecurityserviceapps API
// +k8s:openapi-gen=true
type MobileSecurityServiceApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileSecurityServiceAppSpec   `json:"spec,omitempty"`
	Status MobileSecurityServiceAppStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceAppList contains a list of MobileSecurityServiceApp
type MobileSecurityServiceAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileSecurityServiceApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MobileSecurityServiceApp{}, &MobileSecurityServiceAppList{})
}
