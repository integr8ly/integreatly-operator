package config

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type MCG struct {
	config ProductConfig
}

func NewMCG(config ProductConfig) *MCG {
	return &MCG{config: config}
}

func (m *MCG) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{}
}

func (m *MCG) Read() ProductConfig {
	return m.config
}

func (m *MCG) GetNamespace() string {
	return m.config["NAMESPACE"]
}

func (m *MCG) SetNamespace(newNamespace string) {
	m.config["NAMESPACE"] = newNamespace
}

func (m *MCG) GetHost() string {
	return m.config["HOST"]
}

func (m *MCG) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductMCG
}

func (m *MCG) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionMCG
}

func (m *MCG) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionMCG
}

func (m *MCG) GetOperatorNamespace() string {
	return m.config["OPERATOR_NAMESPACE"]
}

func (m *MCG) SetOperatorNamespace(newNamespace string) {
	m.config["OPERATOR_NAMESPACE"] = newNamespace
}
