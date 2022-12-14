package v1

import (
	v1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:skip

// Infrastructure holds cluster-wide information about Infrastructure.  The canonical name is `cluster`
type Infrastructure struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec holds user settable values for configuration
	// +required
	Spec InfrastructureSpec `json:"spec"`
	// status holds observed values from the cluster. They may not be overridden.
	// +optional
	Status InfrastructureStatus `json:"status"`
}

// InfrastructureSpec contains settings that apply to the cluster infrastructure.
type InfrastructureSpec struct {
	// cloudConfig is a reference to a ConfigMap containing the cloud provider configuration file.
	// This configuration file is used to configure the Kubernetes cloud provider integration
	// when using the built-in cloud provider integration or the external cloud controller manager.
	// The namespace for this config map is openshift-config.
	// +optional
	CloudConfig v1.ConfigMapFileReference `json:"cloudConfig"`
}

// InfrastructureStatus describes the infrastructure the cluster is leveraging.
type InfrastructureStatus struct {
	// infrastructureName uniquely identifies a cluster with a human friendly name.
	// Once set it should not be changed. Must be of max length 27 and must have only
	// alphanumeric or hyphen characters.
	InfrastructureName string `json:"infrastructureName"`

	// platform is the underlying infrastructure provider for the cluster.
	//
	// Deprecated: Use platformStatus.type instead.
	Platform PlatformType `json:"platform,omitempty"`

	// platformStatus holds status information specific to the underlying
	// infrastructure provider.
	// +optional
	PlatformStatus *PlatformStatus `json:"platformStatus,omitempty"`

	// etcdDiscoveryDomain is the domain used to fetch the SRV records for discovering
	// etcd servers and clients.
	// For more info: https://github.com/etcd-io/etcd/blob/329be66e8b3f9e2e6af83c123ff89297e49ebd15/Documentation/op-guide/clustering.md#dns-discovery
	EtcdDiscoveryDomain string `json:"etcdDiscoveryDomain"`

	// apiServerURL is a valid URI with scheme(http/https), address and
	// port.  apiServerURL can be used by components like the web console
	// to tell users where to find the Kubernetes API.
	APIServerURL string `json:"apiServerURL"`

	// apiServerInternalURL is a valid URI with scheme(http/https),
	// address and port.  apiServerInternalURL can be used by components
	// like kubelets, to contact the Kubernetes API server using the
	// infrastructure provider rather than Kubernetes networking.
	APIServerInternalURL string `json:"apiServerInternalURI"`
}

// PlatformType is a specific supported infrastructure provider.
type PlatformType string

const (
	// AWSPlatformType represents Amazon Web Services infrastructure.
	AWSPlatformType PlatformType = "AWS"

	// AzurePlatformType represents Microsoft Azure infrastructure.
	AzurePlatformType PlatformType = "Azure"

	// BareMetalPlatformType represents managed bare metal infrastructure.
	BareMetalPlatformType PlatformType = "BareMetal"

	// GCPPlatformType represents Google Cloud Platform infrastructure.
	GCPPlatformType PlatformType = "GCP"

	// LibvirtPlatformType represents libvirt infrastructure.
	LibvirtPlatformType PlatformType = "Libvirt"

	// OpenStackPlatformType represents OpenStack infrastructure.
	OpenStackPlatformType PlatformType = "OpenStack"

	// NonePlatformType means there is no infrastructure provider.
	NonePlatformType PlatformType = "None"

	// VSpherePlatformType represents VMWare vSphere infrastructure.
	VSpherePlatformType PlatformType = "VSphere"
)

// PlatformStatus holds the current status specific to the underlying infrastructure provider
// of the current cluster. Since these are used at status-level for the underlying cluster, it
// is supposed that only one of the status structs is set.
type PlatformStatus struct {
	// type is the underlying infrastructure provider for the cluster. This
	// value controls whether infrastructure automation such as service load
	// balancers, dynamic volume provisioning, machine creation and deletion, and
	// other integrations are enabled. If None, no infrastructure automation is
	// enabled. Allowed values are "AWS", "Azure", "BareMetal", "GCP", "Libvirt",
	// "OpenStack", "VSphere", and "None". Individual components may not support
	// all platforms, and must handle unrecognized platforms as None if they do
	// not support that platform.
	Type PlatformType `json:"type"`

	// AWS contains settings specific to the Amazon Web Services infrastructure provider.
	// +optional
	AWS *AWSPlatformStatus `json:"aws,omitempty"`

	// GCP contains settings specific to the Google Cloud Platform infrastructure provider.
	// +optional
	GCP *GCPPlatformStatus `json:"gcp,omitempty"`
}

// AWSPlatformStatus holds the current status of the Amazon Web Services infrastructure provider.
type AWSPlatformStatus struct {
	// region holds the default AWS region for new AWS resources created by the cluster.
	Region string `json:"region"`

	// resourceTags is a list of additional tags to apply to AWS resources created for the cluster.
	// See https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html for information on tagging AWS resources.
	// AWS supports a maximum of 50 tags per resource. OpenShift reserves 25 tags for its use, leaving 25 tags
	// available for the user.
	// +kubebuilder:validation:MaxItems=25
	// +optional
	ResourceTags []AWSResourceTag `json:"resourceTags,omitempty"`
}

// AWSResourceTag is a tag to apply to AWS resources created for the cluster.
type AWSResourceTag struct {
	// key is the key of the tag
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[0-9A-Za-z_.:/=+-@]+$`
	// +required
	Key string `json:"key"`
	// value is the value of the tag.
	// Some AWS service do not support empty values. Since tags are added to resources in many services, the
	// length of the tag value must meet the requirements of all services.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Pattern=`^[0-9A-Za-z_.:/=+-@]+$`
	// +required
	Value string `json:"value"`
}

// GCPPlatformStatus holds the current status of the Google Cloud Platform infrastructure provider.
type GCPPlatformStatus struct {
	// resourceGroupName is the Project ID for new GCP resources created for the cluster.
	ProjectID string `json:"projectID"`

	// region holds the region for new GCP resources created for the cluster.
	Region string `json:"region"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureList is
type InfrastructureList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ListMeta `json:"metadata"`
	Items           []Infrastructure `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Infrastructure{}, &InfrastructureList{})
}
