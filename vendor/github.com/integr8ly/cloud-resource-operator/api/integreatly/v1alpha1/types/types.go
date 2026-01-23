package types

import (
	"fmt"
)

var (
	PhaseInProgress                               StatusPhase   = "in progress"
	PhaseDeleteInProgress                         StatusPhase   = "deletion in progress"
	PhaseComplete                                 StatusPhase   = "complete"
	PhasePaused                                   StatusPhase   = "paused"
	PhaseFailed                                   StatusPhase   = "failed"
	StatusEmpty                                   StatusMessage = ""
	StatusUnsupportedType                         StatusMessage = "unsupported deployment type"
	StatusDeploymentConfigNotFound                StatusMessage = "deployment configuration not found"
	StatusSkipCreate                              StatusMessage = "skipping create or update for maintenance"
	StatusNetworkCreateError                      StatusMessage = "failed to create network service"
	StatusNetworkIPRangePendingCreation           StatusMessage = "ip address range is pending creation"
	StatusNetworkIPRangeNotExistOrPendingCreation StatusMessage = "ip address range does not exist or is pending creation"
	StatusNetworkServiceConnectionPendingCreation StatusMessage = "service connection is pending creation"
)

type SecretRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// Duration is a custom time duration that can support any combination of the following units, in descending order: d, h, m.
// Examples: `1d`, `12h`, `12h30m`, `7d12h15m`
// +kubebuilder:validation:Pattern:="^(0|(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?)$"
type Duration string

// +kubebuilder:object:generate=true
type ResourceTypeSpec struct {
	Type       string `json:"type"`
	Tier       string `json:"tier"`
	SkipCreate bool   `json:"skipCreate,omitempty"`
	// ApplyImmediately is only available to Postgres cr, for blobstorage and redis cr's currently does nothing
	ApplyImmediately  bool       `json:"applyImmediately,omitempty"`
	MaintenanceWindow bool       `json:"maintenanceWindow,omitempty"`
	SecretRef         *SecretRef `json:"secretRef"`
	// Size allows defining the node size. It is only available to Redis CR. Blobstorage and Postgres CR's currently does nothing
	Size string `json:"size,omitempty"`
	// SnapshotFrequency is how frequent a new snapshot is to be taken.
	// Does not apply to BlobStorage
	SnapshotFrequency Duration `json:"snapshotFrequency,omitempty"`
	// SnapshotRetention is the number of days each snapshot is to be retained.
	// Does not apply to BlobStorage
	SnapshotRetention Duration `json:"snapshotRetention,omitempty"`
}

type StatusPhase string

type StatusMessage string

func (sm StatusMessage) WrapError(err error) StatusMessage {
	if err == nil {
		return sm
	}
	return StatusMessage(fmt.Sprintf("%s: %s", sm, err.Error()))
}

// +kubebuilder:object:generate=true
type ResourceTypeStatus struct {
	Strategy  string        `json:"strategy,omitempty"`
	Provider  string        `json:"provider,omitempty"`
	Version   string        `json:"version,omitempty"`
	SecretRef *SecretRef    `json:"secretRef,omitempty"`
	Phase     StatusPhase   `json:"phase,omitempty"`
	Message   StatusMessage `json:"message,omitempty"`
}

type ResourceTypeSnapshotStatus struct {
	SnapshotID string        `json:"snapshotID,omitempty"`
	Phase      StatusPhase   `json:"phase,omitempty"`
	Message    StatusMessage `json:"message,omitempty"`
	Strategy   string        `json:"strategy,omitempty"`
}
