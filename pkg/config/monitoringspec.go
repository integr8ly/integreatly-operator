package config

import (
	"errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type MonitoringSpec struct {
	Config ProductConfig
}

func NewMonitoringSpec(config ProductConfig) *MonitoringSpec {
	return &MonitoringSpec{Config: config}
}

func (m *MonitoringSpec) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{}
}

func (m *MonitoringSpec) GetNamespace() string {
	return m.Config["NAMESPACE"]
}

func (m *MonitoringSpec) SetNamespace(newNamespace string) {
	m.Config["NAMESPACE"] = newNamespace
}

func (m *MonitoringSpec) GetNamespacePrefix() string {
	return m.Config["NAMESPACE_PREFIX"]
}

func (m *MonitoringSpec) SetNamespacePrefix(newNamespacePrefix string) {
	m.Config["NAMESPACE_PREFIX"] = newNamespacePrefix
}

func (m *MonitoringSpec) GetHost() string {
	return m.Config["HOST"]
}

func (m *MonitoringSpec) SetHost(newHost string) {
	m.Config["HOST"] = newHost
}

func (m *MonitoringSpec) Read() ProductConfig {
	return m.Config
}

func (m *MonitoringSpec) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductMonitoringSpec
}

func (m *MonitoringSpec) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionMonitoringSpec
}

func (m *MonitoringSpec) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionMonitoringSpec
}

func (m *MonitoringSpec) SetProductVersion(version string) {
	m.Config["VERSION"] = version
}

func (m *MonitoringSpec) Validate() error {
	if m.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if m.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}

	if m.GetProductVersion() == "" {
		return errors.New("config product version is not defined")
	}

	return nil
}
