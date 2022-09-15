// TODO(MGDAPI-4641) Remove this file,
// After removal run make code/compile.
// Check that auto generated file config/crd/bases/integreatly.org_rhmiconfigs.yaml has being removed
// ZZ_generated.deepcopy.go should be auto updated

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RHMIConfig is the Schema for the rhmiconfigs API
type RHMIConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// +kubebuilder:object:root=true

// RHMIConfigList contains a list of RHMIConfig
type RHMIConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHMIConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RHMIConfig{}, &RHMIConfigList{})
}
