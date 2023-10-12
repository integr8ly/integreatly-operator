package config

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Grafana struct {
	config ProductConfig
}

func NewGrafana(config ProductConfig) *Grafana {
	return &Grafana{config: config}
}
func (s *Grafana) GetWatchableCRDs() []runtime.Object {
	return nil
}

func (s *Grafana) GetNamespace() string {
	return s.config["NAMESPACE"]
}

func (s *Grafana) SetNamespace(newNamespace string) {
	s.config["NAMESPACE"] = newNamespace
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

func (s *Grafana) SetProductVersion(newVersion string) {
	s.config["VERSION"] = newVersion
}

func (s *Grafana) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	// it's a stub, not in use. Present just to avoid error in reconciler:
	//Cannot use 'productConfig' (type *Grafana) as the type ConfigReadableType does not implement 'ConfigReadable'
	//as some methods are missing:GetOperatorVersion() integreatlyv1alpha1.OperatorVersion
	return "123stub"
}
