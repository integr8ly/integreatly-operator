/*


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

type ProvisioningStatus string

var (
	UserAnnotated              ProvisioningStatus = "user annotated"
	ThreeScaleAccountReady     ProvisioningStatus = "3scale account ready"
	ThreeScaleAccountRequested ProvisioningStatus = "3scale account requested"
)

// RhoamTenantSpec defines the desired state of RhoamTenant
type RhoamTenantSpec struct {
}

// RhoamTenantStatus defines the observed state of RhoamTenant
type RhoamTenantStatus struct {
	LastError          string             `json:"lastError"`
	ProvisioningStatus ProvisioningStatus `json:"provisioningStatus"`
	TenantUrl          string             `json:"tenantUrl,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// RhoamTenant is the Schema for the RhoamTenants API
type RhoamTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RhoamTenantSpec   `json:"spec,omitempty"`
	Status RhoamTenantStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RhoamTenantList contains a list of RhoamTenant
type RhoamTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RhoamTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RhoamTenant{}, &RhoamTenantList{})
}
