package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileSecurityServiceDBSpec defines the desired state of MobileSecurityServiceDB
// +k8s:openapi-gen=true
type MobileSecurityServiceDBSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// Value for the Database Environment Variable (Spec.DatabaseNameParam).
	// This value will be used when the ConfigMap created by the MobileSecurityService is not found
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabaseName string `json:"databaseName,omitempty"`

	// Value for the Database Environment Variable (Spec.DatabasePasswordParam).
	// This value will be used when the ConfigMap created by the MobileSecurityService is not found
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabasePassword string `json:"databasePassword,omitempty"`

	// Value for the Database Environment Variable (Spec.DatabaseUser).
	// This value will be used when the ConfigMap created by the MobileSecurityService is not found
	// More info: https://github.com/aerogear/mobile-security-service-operator#changing-the-environment-variables-values
	DatabaseUser string `json:"databaseUser,omitempty"`

	// Key Value for the Database Environment Variable in order to inform the database mame
	// Note that each database version/image can expected a different value for it.
	DatabaseNameParam string `json:"databaseNameParam,omitempty"`

	// Key Value for the Database Environment Variable in order to inform the database password
	// Note that each database version/image can expected a different value for it.
	DatabasePasswordParam string `json:"databasePasswordParam,omitempty"`

	// Key Value for the Database Environment Variable in order to inform the database user
	// Note that each database version/image can expected a different value for it.
	DatabaseUserParam string `json:"databaseUserParam,omitempty"`

	// Value for the Database Environment Variable in order to define the port which it should use. It will be used in its container as well
	DatabasePort int32 `json:"databasePort,omitempty"`

	// Quantity of instances
	Size int32 `json:"size,omitempty"`
	// Database image:tag E.g "centos/postgresql-96-centos7"
	Image string `json:"image,omitempty"`
	// Name to create the Database container
	ContainerName string `json:"containerName,omitempty"`
	// Limit of Memory which will be available for the database container
	DatabaseMemoryLimit string `json:"databaseMemoryLimit,omitempty"`
	// Limit of Memory Request which will be available for the database container
	DatabaseMemoryRequest string `json:"databaseMemoryRequest,omitempty"`
	// Limit of Storage Request which will be available for the database container
	DatabaseStorageRequest string `json:"databaseStorageRequest,omitempty"`

	// Policy definition to pull the Database Image
	// More info: https://kubernetes.io/docs/concepts/containers/images/
	ContainerImagePullPolicy v1.PullPolicy `json:"containerImagePullPolicy,omitempty"`
}

// MobileSecurityServiceDBStatus defines the observed state of MobileSecurityServiceDB
// +k8s:openapi-gen=true
type MobileSecurityServiceDBStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// Name of the PersistentVolumeClaim created and managed by it
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	PersistentVolumeClaimName string `json:"persistentVolumeClaimName"`
	// Name of the Database Deployment created and managed by it
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DeploymentName string `json:"deploymentName"`
	// Status of the Database Deployment created and managed by it
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DeploymentStatus appsv1.DeploymentStatus `json:"deploymentStatus"`
	// Name of the Database Service created and managed by it
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ServiceName string `json:"serviceName"`
	// Status of the Database Service created and managed by it
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	ServiceStatus v1.ServiceStatus `json:"serviceStatus"`
	// Will be as "OK when all objects are created successfully
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DatabaseStatus string `json:"databaseStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceDB is the Schema for the mobilesecurityservicedbs API
// +k8s:openapi-gen=true
type MobileSecurityServiceDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileSecurityServiceDBSpec   `json:"spec,omitempty"`
	Status MobileSecurityServiceDBStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceDBList contains a list of MobileSecurityServiceDB
type MobileSecurityServiceDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileSecurityServiceDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MobileSecurityServiceDB{}, &MobileSecurityServiceDBList{})
}
