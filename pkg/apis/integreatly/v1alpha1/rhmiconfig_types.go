/*
Copyright YEAR Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RHMIConfigSpec defines the desired state of RHMIConfig
type RHMIConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Upgrade     Upgrade     `json:"upgrade,omitempty"`
	Maintenance Maintenance `json:"maintenance,omitempty"`
	Backup      Backup      `json:"backup,omitempty"`
}

// RHMIConfigStatus defines the observed state of RHMIConfig
type RHMIConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

type Upgrade struct {
	Contacts              string `json:"contacts,omitempty"`
	AlwaysImmediately     bool   `json:"alwaysImmediately"`
	DuringNextMaintenance bool   `json:"duringNextMaintenance"`
	ApplyOn               string `json:"applyOn,omitempty"`
}

type Maintenance struct {
	ApplyFrom string `json:"applyFrom,omitempty"`
}

type Backup struct {
	ApplyOn string `json:"applyOn,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHMIConfig is the Schema for the rhmiconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rhmiconfigs,scope=Namespaced
type RHMIConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RHMIConfigSpec   `json:"spec,omitempty"`
	Status RHMIConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHMIConfigList contains a list of RHMIConfig
type RHMIConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHMIConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RHMIConfig{}, &RHMIConfigList{})
}
