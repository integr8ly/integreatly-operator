package config

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Observability struct {
	Config ProductConfig
}

func NewObservability(config ProductConfig) *Observability {
	return &Observability{Config: config}
}

func (m *Observability) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductMarin3r
}

func (m *Observability) GetOperatorNamespace() string {
	return m.Config["OPERATOR_NAMESPACE"]
}

func (m *Observability) SetOperatorNamespace(newNamespace string) {
	m.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (m *Observability) GetNamespace() string {
	return m.Config["NAMESPACE"]
}

func (m *Observability) SetNamespace(newNamespace string) {
	m.Config["NAMESPACE"] = newNamespace
}


func (m *Observability) Read() ProductConfig {
	return m.Config
}

func (m *Observability) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionObservability
}

func (m *Observability) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionObservability
}

func (m *Observability) SetProductVersion(newVersion string) {
	m.Config["VERSION"] = newVersion
}

func (m *Observability) GetHost() string {
	return m.Config["HOST"]
}

func (m *Observability) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{}
}