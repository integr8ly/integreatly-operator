package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LauncherSpec defines the desired state of Launcher
type LauncherSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	GitHub    GitHubConfig    `json:"github"`
	OpenShift OpenShiftConfig `json:"openshift,omitempty"`
	OAuth     OAuthConfig     `json:"oauth,omitempty"`
}

// OAuthConfig defines the OAuth configuration
type OAuthConfig struct {
	Enabled bool `json:"enabled"`
}

// OpenShiftConfig defines the OpenShift configuration
type OpenShiftConfig struct {
	ConsoleURL string `json:"consoleUrl,omitempty"`
}

// GitHubConfig defines the Git configuration
type GitHubConfig struct {
	Token SensitiveValue `json:"token,omitempty"`
}

// SensitiveValue defines a sensitive value
type SensitiveValue struct {
	ValueFrom ValueFrom `json:"valueFrom"`
}

// ValueFrom defines where the value is read from
type ValueFrom struct {
	SecretKeyRef SecretKeyRef `json:"secretKeyRef,omitempty"`
	Text         string       `json:"text,omitempty"`
}

// SecretKeyRef defines how the retrieve the secret value
type SecretKeyRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// LauncherStatus defines the observed state of Launcher
type LauncherStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Launcher is the Schema for the launchers API
// +k8s:openapi-gen=true
type Launcher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LauncherSpec   `json:"spec,omitempty"`
	Status LauncherStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LauncherList contains a list of Launcher
type LauncherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Launcher `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Launcher{}, &LauncherList{})
}
