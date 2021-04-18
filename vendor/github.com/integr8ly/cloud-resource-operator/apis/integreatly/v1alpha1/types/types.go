package types

import (
	"fmt"
)

var (
	PhaseInProgress                StatusPhase   = "in progress"
	PhaseDeleteInProgress          StatusPhase   = "deletion in progress"
	PhaseComplete                  StatusPhase   = "complete"
	PhasePaused                    StatusPhase   = "paused"
	PhaseFailed                    StatusPhase   = "failed"
	StatusEmpty                    StatusMessage = ""
	StatusUnsupportedType          StatusMessage = "unsupported deployment type"
	StatusDeploymentConfigNotFound StatusMessage = "deployment configuration not found"
	StatusSkipCreate               StatusMessage = "skipping create or update for maintenance"
)

type SecretRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// +kubebuilder:object:generate=true
type ResourceTypeSpec struct {
	Type       string     `json:"type"`
	Tier       string     `json:"tier"`
	SkipCreate bool       `json:"skipCreate,omitempty"`
	SecretRef  *SecretRef `json:"secretRef"`
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
	SecretRef *SecretRef    `json:"secretRef,omitempty"`
	Phase     StatusPhase   `json:"phase,omitempty"`
	Message   StatusMessage `json:"message,omitempty"`
}

type ResourceTypeSnapshotStatus struct {
	SnapshotID string        `json:"snapshotID,omitempty"`
	Phase      StatusPhase   `json:"phase,omitempty"`
	Message    StatusMessage `json:"message,omitempty"`
}
