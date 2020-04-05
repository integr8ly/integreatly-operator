package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WebAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WebApp `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WebApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WebAppSpec   `json:"spec"`
	Status            WebAppStatus `json:"status,omitempty"`
}

type WebAppSpec struct {
	AppLabel string         `json:"app_label"`
	Template WebAppTemplate `json:"template"`
}

type WebAppStatus struct {
	Message string `json:"message"`
	Version string `json:"version"`
}

type WebAppTemplate struct {
	Path       string            `json:"path"`
	Parameters map[string]string `json:"parameters"`
}
