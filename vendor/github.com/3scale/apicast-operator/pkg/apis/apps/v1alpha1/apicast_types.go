package v1alpha1

import (
	appscommon "github.com/3scale/apicast-operator/pkg/apis/apps"

	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// APIcastSpec defines the desired state of APIcast
// +k8s:openapi-gen=true
type APIcastSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	// +optional
	Replicas *int64 `json:"replicas,omitempty"`
	// +optional
	AdminPortalCredentialsRef *v1.LocalObjectReference `json:"adminPortalCredentialsRef,omitempty"`
	// +optional
	EmbeddedConfigurationSecretRef *v1.LocalObjectReference `json:"embeddedConfigurationSecretRef,omitempty"`
	// +optional
	ServiceAccount *string `json:"serviceAccount,omitempty"`
	// +optional
	Image *string `json:"image,omitempty"`
	// +optional
	ExposedHost *APIcastExposedHost `json:"exposedHost,omitempty"`
	// +optional
	DeploymentEnvironment *DeploymentEnvironmentType `json:"deploymentEnvironment,omitempty"` // THREESCALE_DEPLOYMENT_ENV
	// +optional
	DNSResolverAddress *string `json:"dnsResolverAddress,omitempty"` // RESOLVER
	// +optional
	EnabledServices []string `json:"enabledServices,omitempty"` // APICAST_SERVICES_LIST
	// +optional
	// +kubebuilder:validation:Enum=boot;lazy
	ConfigurationLoadMode *string `json:"configurationLoadMode,omitempty"` // APICAST_CONFIGURATION_LOADER
	// +optional
	// +kubebuilder:validation:Enum=debug;info;notice;warn;error;crit;alert;emerg
	LogLevel *string `json:"logLevel,omitempty"` // APICAST_LOG_LEVEL
	// +optional
	PathRoutingEnabled *bool `json:"pathRoutingEnabled,omitempty"` // APICAST_PATH_ROUTING
	// +optional
	ResponseCodesIncluded *bool `json:"responseCodesIncluded,omitempty"` // APICAST_RESPONSE_CODES
	// +optional
	CacheConfigurationSeconds *int64 `json:"cacheConfigurationSeconds,omitempty"` // APICAST_CONFIGURATION_CACHE
	// +optional
	// +kubebuilder:validation:Enum=disabled;status;policies;debug
	ManagementAPIScope *string `json:"managementAPIScope,omitempty"` // APICAST_MANAGEMENT_API
	// +optional
	OpenSSLPeerVerificationEnabled *bool `json:"openSSLPeerVerificationEnabled,omitempty"` // OPENSSL_VERIFY
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	// +kubebuilder:validation:Minimum=1
	Workers *int32 `json:"workers,omitempty"`
}

type DeploymentEnvironmentType string

const (
	DeploymentEnvironmentProduction = "production"
	DeploymentEnvironmentStaging    = "staging"
)

// APIcastStatus defines the observed state of APIcast
// +k8s:openapi-gen=true
type APIcastStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// Represents the latest available observations of a replica set's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []APIcastCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// The image being used in the APIcast deployment
	// +optional
	Image string `json:"image,omitempty"`
}

type APIcastExposedHost struct {
	Host string `json:"host"`
	// +optional
	TLS []extensions.IngressTLS `json:"tls,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// APIcast is the Schema for the apicasts API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=apicasts,scope=Namespaced
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="APIcast"
type APIcast struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIcastSpec   `json:"spec,omitempty"`
	Status APIcastStatus `json:"status,omitempty"`
}

type APIcastConditionType string

type APIcastCondition struct {
	// Type of replica set condition.
	Type APIcastConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`

	// The Reason, Message, LastHeartbeatTime and LastTransitionTime fields are
	// optional. Unless we really use them they should directly not be used even
	// if they are optional
	// The last time the condition transitioned from one status to another.
	// +optional
	//LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	//Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	//Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// APIcastList contains a list of APIcast
type APIcastList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIcast `json:"items"`
}

func (a *APIcast) GetOwnerRefence() *metav1.OwnerReference {
	trueVar := true
	return &metav1.OwnerReference{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       appscommon.APIcastKind,
		Name:       a.Name,
		UID:        a.UID,
		Controller: &trueVar,
	}
}

func (a *APIcast) Reset() { *a = APIcast{} }

func init() {
	SchemeBuilder.Register(&APIcast{}, &APIcastList{})
}
