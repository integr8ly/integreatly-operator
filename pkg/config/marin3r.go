package config

import (
	marin3rv1alpha "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Marin3r struct {
	config ProductConfig
}

func NewMarin3r(config ProductConfig) *Marin3r {
	return &Marin3r{config: config}
}

func (m *Marin3r) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductMarin3r
}

func (m *Marin3r) GetOperatorNamespace() string {
	return m.config["OPERATOR_NAMESPACE"]
}

func (m *Marin3r) SetOperatorNamespace(newNamespace string) {
	m.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (m *Marin3r) GetNamespace() string {
	return m.config["NAMESPACE"]
}

func (m *Marin3r) Read() ProductConfig {
	return m.config
}

func (m *Marin3r) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionMarin3r
}

func (m *Marin3r) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionMarin3r
}

func (m *Marin3r) GetHost() string {
	return m.config["HOST"]
}

func (m *Marin3r) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&marin3rv1alpha.EnvoyConfig{
			TypeMeta: metav1.TypeMeta{
				Kind:       "EnvoyConfig",
				APIVersion: marin3rv1alpha.SchemeGroupVersion.String(),
			},
		},
	}
}

func (m *Marin3r) SetNamespace(newNamespace string) {
	m.config["NAMESPACE"] = newNamespace
}
