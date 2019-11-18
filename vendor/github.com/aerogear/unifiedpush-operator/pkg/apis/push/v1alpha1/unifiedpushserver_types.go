package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UnifiedPushServerSpec defines the desired state of UnifiedPushServer
// +k8s:openapi-gen=true
type UnifiedPushServerSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Backups is an array of configs that will be used to create CronJob resource instances
	Backups []UnifiedPushServerBackup `json:"backups,omitempty"`

	// UseMessageBroker can be set to true to use managed queues, if you are using enmasse. Defaults to false.
	UseMessageBroker bool `json:"useMessageBroker,omitempty"`

	UnifiedPushResourceRequirements corev1.ResourceRequirements `json:"unifiedPushResourceRequirements,omitempty"`
	OAuthResourceRequirements       corev1.ResourceRequirements `json:"oAuthResourceRequirements,omitempty"`
	PostgresResourceRequirements    corev1.ResourceRequirements `json:"postgresResourceRequirements,omitempty"`

	// PVC size for Postgres service
	PostgresPVCSize string `json:"postgresPVCSize,omitempty"`
}

// UnifiedPushServerStatus defines the observed state of UnifiedPushServer
// +k8s:openapi-gen=true
type UnifiedPushServerStatus struct {
	Phase StatusPhase `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UnifiedPushServer is the Schema for the unifiedpushservers API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=unifiedpushservers,shortName=ups
// +kubebuilder:singular=unifiedpushserver
// +kubebuilder:subresource:status
type UnifiedPushServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UnifiedPushServerSpec   `json:"spec,omitempty"`
	Status UnifiedPushServerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UnifiedPushServerList contains a list of UnifiedPushServer
type UnifiedPushServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UnifiedPushServer `json:"items"`
}

// Backup contains the info needed to configure a CronJob for backups
type UnifiedPushServerBackup struct {
	// Name is the name that will be given to the resulting
	// CronJob
	Name string `json:"name"`

	// Schedule is the schedule that the job will be run at, in
	// cron format
	Schedule string `json:"schedule"`

	// EncryptionKeySecretName is the name of a secret containing
	// PGP/GPG details, including "GPG_PUBLIC_KEY",
	// "GPG_TRUST_MODEL", and "GPG_RECIPIENT"
	EncryptionKeySecretName string `json:"encryptionKeySecretName,omitempty"`

	// EncryptionKeySecretNamespace is the name of the namespace
	// that the secret referenced in EncryptionKeySecretName
	// resides in
	EncryptionKeySecretNamespace string `json:"encryptionKeySecretNamespace,omitempty"`

	// BackendSecretName is the name of a secret containing
	// storage backend details, such as "AWS_S3_BUCKET_NAME",
	// "AWS_ACCESS_KEY_ID", and "AWS_SECRET_ACCESS_KEY"
	BackendSecretName string `json:"backendSecretName"`

	// BackendSecretNamespace is the name of the namespace that
	// the secret referenced in BackendSecretName resides in
	BackendSecretNamespace string `json:"backendSecretNamespace,omitempty"`
}

type StatusPhase string

var (
	PhaseEmpty     StatusPhase = ""
	PhaseComplete  StatusPhase = "Complete"
	PhaseProvision StatusPhase = "Provisioning"
)

func init() {
	SchemeBuilder.Register(&UnifiedPushServer{}, &UnifiedPushServerList{})
}
