package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UnifiedPushServerSpec defines the desired state of UnifiedPushServer
// +k8s:openapi-gen=true
type UnifiedPushServerSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	//ExternalDB can be set to true to use details from Database and connect to external db
	ExternalDB bool `json:"externalDB,omitempty"`

	// Database allows specifying the external PostgreSQL details directly in the CR. Only one
	// of Database or DatabaseSecret should be specified, and ExternalDB must be true, otherwise
	// a new PostgreSQL instance will be created (and deleted) on the cluster automatically.
	Database UnifiedPushServerDatabase `json:"database,omitempty"`

	// DatabaseSecret allows reading the external PostgreSQL details from a pre-existing Secret
	// (ExternalDB must be true for it to be used). Only one of Database or DatabaseSecret
	// should be specified, and ExternalDB must be true, otherwise a new PostgreSQL instance
	// will be created (and deleted) on the cluster automatically.
	//
	// Here's an example of all of the fields that the secret must contain:
	//
	// POSTGRES_DATABASE: sampledb
	// POSTGRES_HOST: 172.30.139.148
	// POSTGRES_PORT: "5432"
	// POSTGRES_USERNAME: userMSM
	// POSTGRES_PASSWORD: RmwWKKIM7or7oJig
	// POSTGRES_SUPERUSER: "false"
	// POSTGRES_VERSION: "10"
	//
	DatabaseSecret string `json:"databaseSecret,omitempty"`

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
	// Phase indicates whether the CR is reconciling(good), failing(bad), or initializing.
	Phase StatusPhase `json:"phase"`

	// Message is a more human-readable message indicating details about current phase or error.
	Message string `json:"message,omitempty"`

	// Ready is True if all resources are in a ready state and all work is done (phase should be
	// "reconciling"). The type in the Go code here is deliberately a pointer so that we can
	// distinguish between false and "not set", since it's an optional field.
	Ready *bool `json:"ready,omitempty"`

	// SecondaryResources is a map of all the secondary resources types and names created for
	// this CR.  e.g "Deployment": [ "DeploymentName1", "DeploymentName2" ]
	SecondaryResources map[string][]string `json:"secondaryResources,omitempty"`
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

// UnifiedPushServerDatabase contains the data needed to connect to external database
type UnifiedPushServerDatabase struct {
	//Name for external database support
	Name string `json:"name,omitempty"`
	//Password for external database support
	Password string `json:"password,omitempty"`
	//User for external database support
	User string `json:"user,omitempty"`
	//Host for external database support
	Host string `json:"host,omitempty"`
	//Port for external database support
	Port intstr.IntOrString `json:"port,omitempty"`
}

type StatusPhase string

var (
	PhaseEmpty        StatusPhase
	PhaseFailing      StatusPhase = "Failing"
	PhaseReconciling  StatusPhase = "Reconciling"
	PhaseInitializing StatusPhase = "Initializing"
)

func init() {
	SchemeBuilder.Register(&UnifiedPushServer{}, &UnifiedPushServerList{})
}
