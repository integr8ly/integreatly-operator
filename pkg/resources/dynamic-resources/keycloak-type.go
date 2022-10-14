package dynamic_resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TLSTerminationType string

var (
	DefaultTLSTermintation        TLSTerminationType
	ReencryptTLSTerminationType   TLSTerminationType = "reencrypt"
	PassthroughTLSTerminationType TLSTerminationType = "passthrough"
)

type KeycloakSpec struct {
	Unmanaged bool `json:"unmanaged,omitempty"`
	External KeycloakExternal `json:"external"`
	Extensions []string `json:"extensions,omitempty"`
	Instances int `json:"instances,omitempty"`
	ExternalAccess KeycloakExternalAccess `json:"externalAccess,omitempty"`
	ExternalDatabase KeycloakExternalDatabase `json:"externalDatabase,omitempty"`
	Profile string `json:"profile,omitempty"`
	PodDisruptionBudget PodDisruptionBudgetConfig `json:"podDisruptionBudget,omitempty"`
	KeycloakDeploymentSpec KeycloakDeploymentSpec `json:"keycloakDeploymentSpec,omitempty"`
	PostgresDeploymentSpec PostgresqlDeploymentSpec `json:"postgresDeploymentSpec,omitempty"`
	Migration MigrateConfig `json:"migration,omitempty"`
	StorageClassName *string `json:"storageClassName,omitempty"`
	MultiAvailablityZones MultiAvailablityZonesConfig `json:"multiAvailablityZones,omitempty"`
	DisableMonitoringServices bool `json:"DisableDefaultServiceMonitor,omitempty"`
	DisableReplicasSyncing bool `json:"disableReplicasSyncing,omitempty"`
}

type DeploymentSpec struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

type KeycloakDeploymentSpec struct {
	DeploymentSpec `json:",inline"`
	PodAnnotations map[string]string `json:"podannotations,omitempty"`
	PodLabels map[string]string `json:"podlabels,omitempty"`

	Experimental ExperimentalSpec `json:"experimental,omitempty"`
}

type PostgresqlDeploymentSpec struct {
	DeploymentSpec `json:",inline"`
}

type ExperimentalSpec struct {
	Args []string `json:"args,omitempty"`
	Command []string `json:"command,omitempty"`
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Volumes VolumesSpec `json:"volumes,omitempty"`
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type VolumesSpec struct {
	Items []VolumeSpec `json:"items,omitempty"`
	DefaultMode *int32 `json:"defaultMode,omitempty"`
}

type VolumeSpec struct {
	Name string `json:"name,omitempty"`
	MountPath string `json:"mountPath"`
	ConfigMaps []string `json:"configMaps,omitempty"`
	Secrets []string `json:"secrets,omitempty"`
	Items []corev1.KeyToPath `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

type KeycloakExternal struct {
	Enabled bool `json:"enabled,omitempty"`
	URL string `json:"url,omitempty"`
	ContextRoot string `json:"contextRoot,omitempty"`
}

type KeycloakExternalAccess struct {
	Enabled bool `json:"enabled,omitempty"`
	TLSTermination TLSTerminationType `json:"tlsTermination,omitempty"`
	Host string `json:"host,omitempty"`
}

type KeycloakExternalDatabase struct {
	Enabled bool `json:"enabled,omitempty"`
}

type PodDisruptionBudgetConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type MultiAvailablityZonesConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type MigrateConfig struct {
	MigrationStrategy MigrationStrategy `json:"strategy,omitempty"`
	Backups BackupConfig `json:"backups,omitempty"`
}

type MigrationStrategy string

var (
	NoStrategy       MigrationStrategy
	StrategyRecreate MigrationStrategy = "recreate"
	StrategyRolling  MigrationStrategy = "rolling"
)

type BackupConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type KeycloakStatus struct {
	Phase StatusPhase `json:"phase"`
	Message string `json:"message"`
	Ready bool `json:"ready"`
	SecondaryResources map[string][]string `json:"secondaryResources,omitempty"`
	Version string `json:"version"`
	InternalURL string `json:"internalURL"`
	ExternalURL string `json:"externalURL,omitempty"`
	CredentialSecret string `json:"credentialSecret"`
}

type StatusPhase string

var (
	NoPhase           StatusPhase
	PhaseReconciling  StatusPhase = "reconciling"
	PhaseFailing      StatusPhase = "failing"
	PhaseInitialising StatusPhase = "initialising"
)

type Keycloak struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakSpec   `json:"spec,omitempty"`
	Status KeycloakStatus `json:"status,omitempty"`
}
