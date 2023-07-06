package v1alpha1

import (
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddonSpec defines the desired state of Addon.
type AddonSpec struct {
	// Human readable name for this addon.
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`

	// Version of the Addon to deploy.
	// Used for reporting via status and metrics.
	// +optional
	Version string `json:"version,omitempty"`

	// Pause reconciliation of Addon when set to True
	// +optional
	Paused bool `json:"pause"`

	// Defines a list of Kubernetes Namespaces that belong to this Addon.
	// Namespaces listed here will be created prior to installation of the Addon and
	// will be removed from the cluster when the Addon is deleted.
	// Collisions with existing Namespaces will result in the existing Namespaces
	// being adopted.
	Namespaces []AddonNamespace `json:"namespaces,omitempty"`

	// Labels to be applied to all resources.
	CommonLabels map[string]string `json:"commonLabels,omitempty"`

	// Annotations to be applied to all resources.
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty"`

	// Correlation ID for co-relating current AddonCR revision and reported status.
	// +optional
	CorrelationID string `json:"correlationID,omitempty"`

	// Defines how an Addon is installed.
	// This field is immutable.
	Install AddonInstallSpec `json:"install"`

	// Defines whether the addon needs acknowledgment from the underlying
	// addon's operator before deletion.
	// +optional
	DeleteAckRequired bool `json:"deleteAckRequired"`

	// UpgradePolicy enables status reporting via upgrade policies.
	UpgradePolicy *AddonUpgradePolicy `json:"upgradePolicy,omitempty"`

	// Defines how an addon is monitored.
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`

	// Settings for propagating secrets from the Addon Operator install namespace into Addon namespaces.
	SecretPropagation *AddonSecretPropagation `json:"secretPropagation,omitempty"`
	// defines the PackageOperator image as part of the addon Spec
	AddonPackageOperator *AddonPackageOperator `json:"packageOperator,omitempty"`
}

type AddonPackageOperator struct {
	Image string `json:"image"`
}

type AddonSecretPropagation struct {
	Secrets []AddonSecretPropagationReference `json:"secrets"`
}

type AddonSecretPropagationReference struct {
	// Source secret name in the Addon Operator install namespace.
	SourceSecret corev1.LocalObjectReference `json:"sourceSecret"`
	// Destination secret name in every Addon namespace.
	DestinationSecret corev1.LocalObjectReference `json:"destinationSecret"`
}

type AddonUpgradePolicy struct {
	// Upgrade policy id.
	ID string `json:"id"`
}

type AddonUpgradePolicyValue string

const (
	AddonUpgradePolicyValueStarted   AddonUpgradePolicyValue = "started"
	AddonUpgradePolicyValueCompleted AddonUpgradePolicyValue = "completed"
)

// Tracks the last state last reported to the Upgrade Policy endpoint.
type AddonUpgradePolicyStatus struct {
	// Upgrade policy id.
	ID string `json:"id"`
	// Upgrade policy value.
	Value AddonUpgradePolicyValue `json:"value"`
	// Upgrade Policy Version.
	// +optional
	Version string `json:"version,omitempty"`
	// The most recent generation a status update was based on.
	ObservedGeneration int64 `json:"observedGeneration"`
}

type MonitoringSpec struct {
	// Configuration parameters to be injected in the ServiceMonitor used for federation.
	// The target prometheus server found by matchLabels needs to serve service-ca signed TLS traffic
	// (https://docs.openshift.com/container-platform/4.6/security/certificate_types_descriptions/service-ca-certificates.html),
	// and it needs to be runing inside the namespace specified by `.monitoring.federation.namespace`
	// with the service name 'prometheus'.
	Federation *MonitoringFederationSpec `json:"federation,omitempty"`

	// Settings For Monitoring Stack
	// +optional
	MonitoringStack *MonitoringStackSpec `json:"monitoringStack,omitempty"`
}

type MonitoringStackSpec struct {
	// Settings for RHOBS Remote Write
	// +optional
	RHOBSRemoteWriteConfig *RHOBSRemoteWriteConfigSpec `json:"rhobsRemoteWriteConfig,omitempty"`
}

type RHOBSRemoteWriteConfigSpec struct {
	// RHOBS endpoints where your data is sent to
	// It varies by environment:
	// - Staging: https://observatorium-mst.stage.api.openshift.com/api/metrics/v1/<tenant id>/api/v1/receive
	// - Production: https://observatorium-mst.api.openshift.com/api/metrics/v1/<tenant id>/api/v1/receive
	URL string `json:"url"`

	// OAuth2 config for the remote write URL
	// +optional
	OAuth2 *monv1.OAuth2 `json:"oauth2,omitempty"`

	// List of metrics to push to RHOBS.
	// Any metric not listed here is dropped.
	Allowlist []string `json:"allowlist,omitempty"`
}

type MonitoringFederationSpec struct {
	// Namespace where the prometheus server is running.
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// The name of the service port fronting the prometheus server.
	// +kubebuilder:validation:MinLength=1
	PortName string `json:"portName"`

	// List of series names to federate from the prometheus server.
	// +listType:set
	MatchNames []string `json:"matchNames"`

	// List of labels used to discover the prometheus server(s) to be federated.
	// +kubebuilder:validation:MinProperties=1
	MatchLabels map[string]string `json:"matchLabels"`
}

// AddonInstallSpec defines the desired Addon installation type.
type AddonInstallSpec struct {
	// Type of installation.
	// +kubebuilder:validation:Enum={"OLMOwnNamespace","OLMAllNamespaces"}
	Type AddonInstallType `json:"type"`
	// OLMAllNamespaces config parameters. Present only if Type = OLMAllNamespaces.
	OLMAllNamespaces *AddonInstallOLMAllNamespaces `json:"olmAllNamespaces,omitempty"`
	// OLMOwnNamespace config parameters. Present only if Type = OLMOwnNamespace.
	OLMOwnNamespace *AddonInstallOLMOwnNamespace `json:"olmOwnNamespace,omitempty"`
}

// Common Addon installation parameters.
type AddonInstallOLMCommon struct {
	// Namespace to install the Addon into.
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Defines the CatalogSource image.
	// +kubebuilder:validation:MinLength=1
	CatalogSourceImage string `json:"catalogSourceImage"`

	// Channel for the Subscription object.
	// +kubebuilder:validation:MinLength=1
	Channel string `json:"channel"`

	// Name of the package to install via OLM.
	// OLM will resove this package name to install the matching bundle.
	// +kubebuilder:validation:MinLength=1
	PackageName string `json:"packageName"`

	// Reference to a secret of type kubernetes.io/dockercfg or kubernetes.io/dockerconfigjson
	// in the addon operators installation namespace.
	// The secret referenced here, will be made available to the addon in the addon installation namespace,
	// as addon-pullsecret prior to installing the addon itself.
	PullSecretName string `json:"pullSecretName,omitempty"`

	// Configs to be passed to subscription OLM object
	// +optional
	Config *SubscriptionConfig `json:"config,omitempty"`

	// Additional catalog source objects to be created in the cluster
	// +optional
	AdditionalCatalogSources []AdditionalCatalogSource `json:"additionalCatalogSources,omitempty"`
}

type SubscriptionConfig struct {
	// Array of env variables to be passed to the subscription object.
	EnvironmentVariables []EnvObject `json:"env"`
}

type AdditionalCatalogSource struct {
	// Name of the additional catalog source
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Image url of the additional catalog source
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`
}

type EnvObject struct {
	// Name of the environment variable
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Value of the environment variable
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value"`
}

// AllNamespaces specific Addon installation parameters.
type AddonInstallOLMAllNamespaces struct {
	AddonInstallOLMCommon `json:",inline"`
}

// OwnNamespace specific Addon installation parameters.
type AddonInstallOLMOwnNamespace struct {
	AddonInstallOLMCommon `json:",inline"`
}

type AddonInstallType string

const (
	// All namespaces on the cluster (default)
	// installs the Operator in the default openshift-operators namespace to
	// watch and be made available to all namespaces in the cluster.
	// Maps directly to the OLM default install mode "all namespaces".
	OLMAllNamespaces AddonInstallType = "OLMAllNamespaces"
	// Installs the operator into a specific namespace.
	// The Operator will only watch and be made available for use in this single namespace.
	// Maps directly to the OLM install mode "specific namespace"
	OLMOwnNamespace AddonInstallType = "OLMOwnNamespace"
)

// Annotation keys for delete signal from OCM.
const (
	DeleteAnnotationFlag  = "addons.managed.openshift.io/delete"
	DeleteTimeoutDuration = "addons.managed.openshift.io/deletetimeout"
)

// Addon condition reasons

const (
	// Addon as fully reconciled
	AddonReasonFullyReconciled = "FullyReconciled"

	// Addon is terminating
	AddonReasonTerminating = "Terminating"

	// Addon has a configurtion error
	AddonReasonConfigError = "ConfigurationError"

	// Addon has paused reconciliation
	AddonReasonPaused = "AddonPaused"

	// Addon has an unready Catalog source
	AddonReasonUnreadyCatalogSource = "UnreadyCatalogSource"

	// Addon has an unready additional Catalog source
	AddonReasonUnreadyAdditionalCatalogSource = "UnreadyAdditionalCatalogSource"

	// Addon has unready namespaces
	AddonReasonUnreadyNamespaces = "UnreadyNamespaces"

	// Addon has unready metrics federation
	AddonReasonUnreadyMonitoringFederation = "UnreadyMonitoringFederation"

	// Addon has unready monitoring stack
	AddonReasonUnreadyMonitoringStack = "UnreadyMonitoringStack"

	// Addon has unready ClusterPackageTemplate
	AddonReasonUnreadyClusterPackageTemplate = "UnreadyClusterPackageTemplate"

	// Addon has unready CSV
	AddonReasonUnreadyCSV = "UnreadyCSV"

	// CSV for the addon is missing
	AddonReasonMissingCSV = "MissingCSV"

	// Addon cannot find a referenced secret to propagate
	AddonReasonMissingSecretForPropagation = "MissingSecretForPropagation"

	// Addon upgrade has started.
	AddonReasonUpgradeStarted = "AddonUpgradeStarted"

	// Addon upgrade has succeeded.
	AddonReasonUpgradeSucceeded = "AddonUpgradeSucceeded"

	// Addon has successfully been uninstalled.
	AddonReasonInstalled = "AddonInstalled"

	// Addon has successfully been uninstalled.
	AddonReasonNotInstalled = "AddonNotInstalled"

	// Addon is ready to be deleted.
	AddonReasonReadyToBeDeleted = "AddonReadyToBeDeleted"

	// Addon is not yet ready to deleted.
	AddonReasonNotReadyToBeDeleted = "AddonNotReadyToBeDeleted"

	// Addon has timed out waiting for acknowledgement from the underlying addon.
	AddonReasonDeletionTimedOut = "AddonReasonDeletionTimedOut"
)

type AddonNamespace struct {
	// Name of the KubernetesNamespace.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Labels to be added to the namespace
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to be added to the namespace
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

const (
	// Available condition indicates that all resources for the Addon are reconciled and healthy
	Available = "Available"

	// Paused condition indicates that the reconciliation of resources for the Addon(s) has paused
	Paused = "Paused"

	// UpgradeStarted condition indicates that the addon upgrade has started.
	UpgradeStarted = "UpgradeStarted"

	// UpgradeSucceeded condition indicates that the addon upgrade has succeeded.
	UpgradeSucceeded = "UpgradeSucceeded"

	// Installed condition indicates that the addon has been installed successfully
	// and was available atleast once.
	Installed = "Installed"

	// ReadyToBeDeleted condition indicates whether the addon is ready to be deleted or not.
	ReadyToBeDeleted = "ReadyToBeDeleted"

	// DeleteTimeout condition indicates whether an addon has timed out waiting for an delete acknowledgement
	// from underlying addon.
	DeleteTimeout = "DeleteTimeout"
)

// AddonStatus defines the observed state of Addon
type AddonStatus struct {
	// The most recent generation observed by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions is a list of status conditions ths object is in.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// DEPRECATED: This field is not part of any API contract
	// it will go away as soon as kubectl can print conditions!
	// Human readable status - please use .Conditions from code
	Phase AddonPhase `json:"phase,omitempty"`
	// Tracks last reported upgrade policy status.
	// +optional
	UpgradePolicy *AddonUpgradePolicyStatus `json:"upgradePolicy,omitempty"`
	// Tracks the last addon status reported to OCM.
	// +optional
	OCMReportedStatusHash *OCMAddOnStatusHash `json:"ocmReportedStatusHash,omitempty"`
	// Observed version of the Addon on the cluster, only present when .spec.version is populated.
	// +optional
	ObservedVersion string `json:"observedVersion,omitempty"`
	// Namespaced name of the csv(available) that was last observed.
	// +optional
	LastObservedAvailableCSV string `json:"lastObservedAvailableCSV,omitempty"`
}

type AddOnStatusCondition struct {
	StatusType  string                 `json:"status_type"`
	StatusValue metav1.ConditionStatus `json:"status_value"`
	Reason      string                 `json:"reason"`
}

type OCMAddOnStatusHash struct {
	// Hash of the last reported status.
	StatusHash string `json:"statusHash"`
	// The most recent generation a status update was based on.
	ObservedGeneration int64 `json:"observedGeneration"`
}

// Struct used to hash the reported addon status (along with correlationID).
type OCMAddOnStatus struct {
	// ID of the addon.
	AddonID string `json:"addonID"`
	// Correlation ID for co-relating current AddonCR revision and reported status.
	CorrelationID string `json:"correlationID"`
	// Reported addon status conditions
	StatusConditions []AddOnStatusCondition `json:"statusConditions"`
	// The most recent generation a status update was based on.
	ObservedGeneration int64 `json:"observedGeneration"`
}

type AddonPhase string

// Well-known Addon Phases for printing a Status in kubectl,
// see deprecation notice in AddonStatus for details.
const (
	PhasePending     AddonPhase = "Pending"
	PhaseReady       AddonPhase = "Ready"
	PhaseTerminating AddonPhase = "Terminating"
	PhaseError       AddonPhase = "Error"
)

// Addon is the Schema for the Addons API
//
// **Example**
// ```yaml
// apiVersion: addons.managed.openshift.io/v1alpha1
// kind: Addon
// metadata:
//
//	name: reference-addon
//
// spec:
//
//	displayName: An amazing example addon!
//	namespaces:
//	- name: reference-addon
//	install:
//	  type: OLMOwnNamespace
//	  olmOwnNamespace:
//	    namespace: reference-addon
//	    packageName: reference-addon
//	    channel: alpha
//	    catalogSourceImage: quay.io/osd-addons/reference-addon-index@sha256:58cb1c4478a150dc44e6c179d709726516d84db46e4e130a5227d8b76456b5bd
//
// ```
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Addon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AddonSpec `json:"spec,omitempty"`
	// +kubebuilder:default={phase:Pending}
	Status AddonStatus `json:"status,omitempty"`
}

func (a *Addon) IsAvailable() bool {
	return meta.IsStatusConditionTrue(a.Status.Conditions, Available)
}

func (a *Addon) SetUpgradePolicyStatus(val AddonUpgradePolicyValue) {
	a.Status.UpgradePolicy = &AddonUpgradePolicyStatus{
		ID:                 a.Spec.UpgradePolicy.ID,
		Value:              val,
		Version:            a.Spec.Version,
		ObservedGeneration: a.Generation,
	}
}

func (a *Addon) UpgradeCompleteForCurrentVersion() bool {
	return a.Status.UpgradePolicy != nil &&
		a.Status.UpgradePolicy.Version == a.Spec.Version &&
		a.Status.UpgradePolicy.Value == AddonUpgradePolicyValueCompleted
}

// AddonList contains a list of Addon
// +kubebuilder:object:root=true
type AddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Addon `json:"items"`
}

func init() {
	register(&Addon{}, &AddonList{})
}
