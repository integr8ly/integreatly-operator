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
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisSnapshotSpec defines the desired state of RedisSnapshot
type RedisSnapshotSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of RedisSnapshot. Edit RedisSnapshot_types.go to remove/update
	ResourceName string `json:"resourceName"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=redissnapshots,scope=Namespaced

// RedisSnapshot is the Schema for the redissnapshots API
type RedisSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSnapshotSpec                `json:"spec,omitempty"`
	Status types.ResourceTypeSnapshotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisSnapshotList contains a list of RedisSnapshot
type RedisSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisSnapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisSnapshot{}, &RedisSnapshotList{})
}
