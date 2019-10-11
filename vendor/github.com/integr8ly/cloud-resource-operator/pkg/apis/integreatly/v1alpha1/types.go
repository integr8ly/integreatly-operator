package v1alpha1

var (
	PhaseInProgress       StatusPhase   = "in progress"
	PhaseDeleteInProgress StatusPhase   = "deletion in progress"
	PhaseComplete         StatusPhase   = "complete"
	PhaseFailed           StatusPhase   = "failed"
	StatusEmpty           StatusMessage = ""
)

// SecretRef Represents a namespace-scoped Secret
type SecretRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ResourceTypeSpec Represents the basic information required to provision a resource type
type ResourceTypeSpec struct {
	Type      string     `json:"type"`
	Tier      string     `json:"tier"`
	SecretRef *SecretRef `json:"secretRef"`
}

type StatusPhase string
type StatusMessage string

// ResourceTypeStatus Represents the basic status information provided by a resource provider
type ResourceTypeStatus struct {
	Strategy  string        `json:"strategy,omitempty"`
	Provider  string        `json:"provider,omitempty"`
	SecretRef *SecretRef    `json:"secretRef,omitempty"`
	Phase     StatusPhase   `json:"phase,omitempty"`
	Message   StatusMessage `json:"message,omitempty"`
}
