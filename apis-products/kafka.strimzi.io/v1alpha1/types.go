// +kubebuilder:object:generate=false
// +kubebuilder:skip
// +kubebuilder:skipversion
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KafkaTopicOperator struct{}
type KafkaUserOperator struct{}

type KafkaListener struct{}

type KafkaSpecEntityOperator struct {
	TopicOperator KafkaTopicOperator `json:"topicOperator"`
	UserOperator  KafkaUserOperator  `json:"userOperator"`
}

type KafkaSpecZookeeper struct {
	Replicas int          `json:"replicas,omitempty"`
	Storage  KafkaStorage `json:"storage,omitempty"`
}

type KafkaSpecKafka struct {
	Version   string                   `json:"version,omitempty"`
	Replicas  int                      `json:"replicas,omitempty"`
	Listeners map[string]KafkaListener `json:"listeners,omitempty"`
	Config    KafkaSpecKafkaConfig     `json:"config,omitempty"`
	Storage   KafkaStorage             `json:"storage,omitempty"`
}

type KafkaStorage struct {
	Type        string `json:"type,omitempty"`
	Size        string `json:"size,omitempty"`
	DeleteClaim bool   `json:"deleteClaim,omitempty"`
}

type KafkaSpecKafkaConfig struct {
	OffsetsTopicReplicationFactor        string `json:"offsets.topic.replication.factor,omitempty"`
	TransactionStateLogReplicationFactor string `json:"transaction.state.log.replication.factor,omitempty"`
	TransactionStateLogMinIsr            string `json:"transaction.state.log.min.isr,omitempty"`
	LogMessageFormatVersion              string `json:"log.message.format.version,omitempty"`
}

// InstallationSpec defines the desired state of Installation
type KafkaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Kafka          KafkaSpecKafka          `json:"kafka,omitempty"`
	Zookeeper      KafkaSpecZookeeper      `json:"zookeeper,omitempty"`
	EntityOperator KafkaSpecEntityOperator `json:"entityOperator,omitempty"`
}

// InstallationStatus defines the observed state of Installation
type KafkaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// Installation is the Schema for the installations API
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec   `json:"spec,omitempty"`
	Status KafkaStatus `json:"status,omitempty"`
}

// InstallationList contains a list of Installation
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

// KafkaTopic defines a Kafka topic
type KafkaTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KafkaTopicSpec `json:"spec"`
}

// KafkaTopicSpec defines the desired state of the Kafka topic
type KafkaTopicSpec struct {
	Partitions int               `json:"partitions"`
	Replicas   int               `json:"replicas"`
	Config     map[string]string `json:"config"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{}, &KafkaTopic{})
}
