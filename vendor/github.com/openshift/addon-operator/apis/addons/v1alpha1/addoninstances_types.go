package v1alpha1

import (
	"path"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddonInstanceSpec defines the configuration to consider while taking AddonInstance-related decisions such as HeartbeatTimeouts
type AddonInstanceSpec struct {
	// This field indicates whether the addon is marked for deletion.
	// +optional
	MarkedForDeletion bool `json:"markedForDeletion"`
	// The periodic rate at which heartbeats are expected to be received by the AddonInstance object
	// +kubebuilder:default="10s"
	HeartbeatUpdatePeriod metav1.Duration `json:"heartbeatUpdatePeriod,omitempty"`
}

// AddonInstanceStatus defines the observed state of Addon
type AddonInstanceStatus struct {
	// The most recent generation observed by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions is a list of status conditions ths object is in.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Timestamp of the last reported status check
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
}

// AddonInstance is a managed service facing interface to get configuration and report status back.
//
// **Example**
// ```yaml
// apiVersion: addons.managed.openshift.io/v1alpha1
// kind: AddonInstance
// metadata:
//
//	name: addon-instance
//	namespace: my-addon-namespace
//
// spec:
//
//	heartbeatUpdatePeriod: 30s
//
// status:
//
//	lastHeartbeatTime: 2021-10-11T08:14:50Z
//	conditions:
//	- type: addons.managed.openshift.io/Healthy
//	  status: "True"
//
// ```
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Last Heartbeat",type="date",JSONPath=".status.lastHeartbeatTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type AddonInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonInstanceSpec   `json:"spec,omitempty"`
	Status AddonInstanceStatus `json:"status,omitempty"`
}

// AddonInstanceList contains a list of AddonInstances
// +kubebuilder:object:root=true
type AddonInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddonInstance `json:"items"`
}

const (
	DefaultAddonInstanceName                  = "addon-instance"
	DefaultAddonInstanceHeartbeatUpdatePeriod = 10 * time.Second
)

// AddonInstanceCondition is a condition Type used by AddonInstance
// status conditions.
type AddonInstanceCondition string

func (c AddonInstanceCondition) String() string {
	return path.Join(group, string(c))
}

// AddonInstance Conditions
const (
	// AddonInstanceHealthy tracks the general health of an Addon.
	//
	// If false the service is degraded to a point that manual intervention is likely.
	// Higher level controllers are advised to stop actions that might further worsen the state of the service.
	// For example: delaying upgrades until the status is cleared.
	AddonInstanceConditionHealthy AddonInstanceCondition = "Healthy"
	// AddonInstanceDegraded reports partial lose of functionallity which otherwise
	// does not affect the availability of an addon.
	AddonInstanceConditionDegraded AddonInstanceCondition = "Degraded"
	// AddonInstanceConditionInstalled reports installation status as either
	// 'True' or 'False' with additional detail provided thorugh messages
	// while condition is still 'False'.
	AddonInstanceConditionInstalled AddonInstanceCondition = "Installed"

	// ReadyToBeDeleted condition indicates whether the addon is ready to be deleted or not.
	AddonInstanceConditionReadyToBeDeleted AddonInstanceCondition = "ReadyToBeDeleted"
)

// AddonInstanceHealthyReason is a condition reason used by
// AddonInstance status conditions when condition type is
// AddonInstanceConditionHealthy.
type AddonInstanceHealthyReason string

func (r AddonInstanceHealthyReason) String() string {
	return string(r)
}

const (
	// AddonInstanceHealthyReasonReceivingHeartbeats is a status condition
	// reason used when heartbeats are received within the configured
	// threshold.
	AddonInstanceHealthyReasonReceivingHeartbeats AddonInstanceHealthyReason = "ReceivingHeartbeats"
	// AddonInstanceHealthyReasonPendingFirstHeartbeat is a status condition
	// reason used before the first heartbeat has been received from the addon.
	AddonInstanceHealthyReasonPendingFirstHeartbeat AddonInstanceHealthyReason = "PendingFirstHeartbeat"
	// AddonInstanceHealthyReasonHeatbeatTimeout is a status condition
	// reason when the last received timeout was not received before
	// the configured threshold.
	AddonInstanceHealthyReasonHeartbeatTimeout AddonInstanceHealthyReason = "HeartbeatTimeout"
)

// AddonInstanceInstalledReason is a condition reason used by
// AddonInstance status conditions when condition type is
// AddonInstanceConditionInstalled.
type AddonInstanceInstalledReason string

func (r AddonInstanceInstalledReason) String() string {
	return string(r)
}

const (
	// AddonInstanceInstalledReasonSetupComplete is a status condition
	// reason used when addon installation setup work is complete.
	AddonInstanceInstalledReasonSetupComplete AddonInstanceInstalledReason = "SetupComplete"
	// AddonInstanceInstalledReasonTeardownComplete is a status condition
	// reason used when addon installation teardown work is complete.
	AddonInstanceInstalledReasonTeardownComplete AddonInstanceInstalledReason = "TeardownComplete"
	// AddonInstanceInstalledReasonBlocked is a status condition
	// reason used when addon installation is blocked.
	AddonInstanceInstalledReasonBlocked AddonInstanceInstalledReason = "Blocked"
)

type AddonInstanceReadyToBeDeleted string

func (r AddonInstanceReadyToBeDeleted) String() string {
	return string(r)
}

const (
	// Addon is ready to be deleted.
	AddonInstanceReasonReadyToBeDeleted AddonInstanceReadyToBeDeleted = "AddonReadyToBeDeleted"

	// Addon is not yet ready to deleted.
	AddonInstanceReasonNotReadyToBeDeleted AddonInstanceReadyToBeDeleted = "AddonNotReadyToBeDeleted"
)

func init() {
	register(&AddonInstance{}, &AddonInstanceList{})
}
