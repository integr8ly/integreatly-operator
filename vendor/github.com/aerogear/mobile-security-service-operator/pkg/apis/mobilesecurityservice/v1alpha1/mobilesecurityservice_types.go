package v1alpha1

import (
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileSecurityServiceSpec defines the desired state of MobileSecurityService
// +k8s:openapi-gen=true
type MobileSecurityServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// Value for the Service Environment Variable (PGDATABASE).
	// This value will be shared to create the database managed by the MobileSecurityServiceDB via a configMap.
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabaseName string `json:"databaseName,omitempty"`
	// Value for the Service Environment Variable (PGPASSWORD).
	// This value will be shared to create the database managed by the MobileSecurityServiceDB via a configMap.
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabasePassword string `json:"databasePassword,omitempty"`
	// Value for the Service Environment Variable (PGUSER).
	// This value will be shared to create the database managed by the MobileSecurityServiceDB via a configMap.
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabaseUser string `json:"databaseUser,omitempty"`
	// Value for the Service Environment Variable (PGHOST)
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabaseHost string `json:"databaseHost,omitempty"`
	// Value for the Service Environment Variable (LOG_LEVEL)
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	LogLevel string `json:"logLevel,omitempty"`
	// Value for the Service Environment Variable (LOG_FORMAT)
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	LogFormat string `json:"logFormat,omitempty"`
	// Value for the Service Environment Variable (ACCESS_CONTROL_ALLOW_ORIGIN)
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	AccessControlAllowOrigin string `json:"accessControlAllowOrigin,omitempty"`
	// Value for the Service Environment Variable (ACCESS_CONTROL_ALLOW_CREDENTIALS)
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	AccessControlAllowCredentials string `json:"accessControlAllowCredentials,omitempty"`
	// Value for the Service Environment Variable (PORT)
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	Port int32 `json:"port,omitempty"`

	// Quantity of instances
	Size int32 `json:"size,omitempty"`
	// Service image:tag E.g quay.io/aerogear/mobile-security-service:0.1.0
	Image string `json:"image,omitempty"`
	// Name of the container which will be created for the Service
	ContainerName string `json:"containerName,omitempty"`
	// Used to create the URL to allow public access to the Servic
	// Options [http or https].
	ClusterProtocol string `json:"clusterProtocol,omitempty"`
	// Limit of Memory which will be available for the Service container
	MemoryLimit string `json:"memoryLimit,omitempty"`
	// Limit of Memory Request which will be available for the Service container
	MemoryRequest string `json:"memoryRequest,omitempty"`
	// Limit of CPU which will be available for the Service container
	ResourceCpuLimit string `json:"resourceCpuLimit,omitempty"`
	// CPU resource which will be available for the Service container
	ResourceCpu string `json:"resourceCpu,omitempty"`
	// Limit of Memory which will be available for the OAuth container
	OAuthMemoryLimit string `json:"oAuthMemoryLimit,omitempty"`
	// Limit of Memory Request which will be available for the OAuth container
	OAuthMemoryRequest string `json:"oAuthMemoryRequest,omitempty"`
	// Limit of CPU which will be available for the OAuth container
	OAuthResourceCpuLimit string `json:"oAuthResourceCpuLimit,omitempty"`
	// CPU resource which will be available for the OAuth container
	OAuthResourceCpu string `json:"oAuthResourceCpu,omitempty"`
	// Oauth image:tag E.g docker.io/openshift/oauth-proxy:v1.1.0
	// More info: https://github.com/openshift/oauth-proxy
	OAuthImage string `json:"oAuthImage,omitempty"`
	// Name of the container which will be created for the Service pod as sidecar
	OAuthContainerName string `json:"oAuthContainerName,omitempty"`

	// Name of the configMap which will be created to share the data with MobileSecurityServiceDB.
	// Note that by default it is empty and the name will be : MobileSecurityService CR instance Name + -config
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	ConfigMapName string `json:"configMapName,omitempty"`
	// The name of the route which will vbe created to expose the Service
	RouteName string `json:"routeName,omitempty"`
	// Policy definition to pull the Oauth Image
	// More info: https://kubernetes.io/docs/concepts/containers/images/
	OAuthContainerImagePullPolicy v1.PullPolicy `json:"oAuthContainerImagePullPolicy,omitempty"`
	// Policy definition to pull the Service Image
	// More info: https://kubernetes.io/docs/concepts/containers/images/
	ContainerImagePullPolicy v1.PullPolicy `json:"containerImagePullPolicy,omitempty"`
}

// MobileSecurityServiceStatus defines the observed state of MobileSecurityService
// +k8s:openapi-gen=true
type MobileSecurityServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// Name of the ConfigMap created and managed by it with the values used in the Service and Database Environment Variables
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ConfigMapName string `json:"configMapName"`
	// Name of the Deployment created and managed by it to provided the Service Application
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DeploymentName string `json:"deploymentName"`
	// Status of the Deployment created and managed by it to provided the Service Application
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DeploymentStatus appsv1.DeploymentStatus `json:"deploymentStatus"`
	// Name of the Service created and managed by it to expose the Service Application
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ServiceName string `json:"serviceName"`
	// Status of the Service created and managed by it to expose the Service Application
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ServiceStatus v1.ServiceStatus `json:"serviceStatus"`
	// Name of the Proxy Service created and managed by it to allow its internal communication with the database. Required because of the Oauth configuration.
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ProxyServiceName string `json:"proxyServiceName"`
	// Status of the Proxy Service created and managed by it to allow its internal communication with the database. Required because of the Oauth configuration.
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ProxyServiceStatus v1.ServiceStatus `json:"proxyServiceStatus"`
	// Name of the Route object required to expose public the Service Application
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	RouteName string `json:"routeName"`
	// Status of the Route object required to expose public the Service Application
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-type
	RouteStatus routev1.RouteStatus `json:"routeStatus"`
	// Will be as "OK when all objects are created successfully
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	AppStatus string `json:"appStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityService is the Schema for the mobilesecurityservices API
// +k8s:openapi-gen=true
type MobileSecurityService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileSecurityServiceSpec   `json:"spec,omitempty"`
	Status MobileSecurityServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceList contains a list of MobileSecurityService
type MobileSecurityServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileSecurityService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MobileSecurityService{}, &MobileSecurityServiceList{})
}
