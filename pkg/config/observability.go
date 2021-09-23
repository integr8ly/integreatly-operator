package config

import (
	oo "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// Alert manager configuration
	AlertManagerConfigSecretName            = "alertmanager-application-monitoring"
	AlertManagerConfigSecretFileName        = "alertmanager.yaml"
	AlertManagerEmailTemplateSecretFileName = "alertmanager-email-config.tmpl"
	AlertManagerConfigTemplatePath          = "alertmanager/alertmanager-application-monitoring.yaml"
	AlertManagerCustomTemplatePath          = "alertmanager/alertmanager-email-config.tmpl"
)

type Observability struct {
	Config ProductConfig
}

func NewObservability(config ProductConfig) *Observability {
	return &Observability{Config: config}
}

func (m *Observability) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductObservability
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

func (m *Observability) GetLabelSelector() string {
	return "middleware"
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
	return []runtime.Object{
		&oo.Observability{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Observability",
				APIVersion: oo.GroupVersion.String(),
			},
		},
	}
}
