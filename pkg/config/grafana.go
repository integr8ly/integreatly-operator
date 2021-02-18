package config

import (
	"errors"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

type Grafana struct {
	config ProductConfig
}

func NewGrafana(config ProductConfig) *Grafana {
	return &Grafana{config: config}
}
func (s *Grafana) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&grafanav1alpha1.Grafana{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Grafana",
				APIVersion: grafanav1alpha1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (s *Grafana) GetNamespace() string {
	return s.config["NAMESPACE"]
}

func (s *Grafana) SetNamespace(newNamespace string) {
	s.config["NAMESPACE"] = newNamespace
}

func (s *Grafana) GetOperatorNamespace() string {
	return s.config["OPERATOR_NAMESPACE"]
}

func (s *Grafana) SetOperatorNamespace(newNamespace string) {
	s.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (s *Grafana) Read() ProductConfig {
	return s.config
}

func (s *Grafana) GetHost() string {
	return s.config["HOST"]
}

func (s *Grafana) GetLabelSelector() string {
	return "middleware"
}

func (s *Grafana) SetHost(newHost string) {
	s.config["HOST"] = newHost
}

func (s *Grafana) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductGrafana
}

func (s *Grafana) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(s.config["VERSION"])
}

func (s *Grafana) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionGrafana
}

func (s *Grafana) SetProductVersion(newVersion string) {
	s.config["VERSION"] = newVersion
}

func (s *Grafana) Validate() error {
	if s.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if s.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	return nil
}
