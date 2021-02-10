package config

import (
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type AMQStreams struct {
	config ProductConfig
}

func NewAMQStreams(config ProductConfig) *AMQStreams {
	return &AMQStreams{config: config}
}
func (a *AMQStreams) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&kafkav1alpha1.Kafka{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kafkav1alpha1.SchemeGroupVersion.String(),
				Kind:       kafkav1alpha1.KafkaKind,
			},
		},
	}
}

func (a *AMQStreams) GetHost() string {
	return a.config["HOST"]
}

func (a *AMQStreams) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *AMQStreams) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *AMQStreams) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *AMQStreams) GetOperatorNamespace() string {
	return a.config["OPERATOR_NAMESPACE"]
}

func (a *AMQStreams) SetOperatorNamespace(newNamespace string) {
	a.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (a *AMQStreams) Read() ProductConfig {
	return a.config
}

func (a *AMQStreams) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductAMQStreams
}

func (a *AMQStreams) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionAMQStreams
}

func (a *AMQStreams) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionAMQStreams
}
