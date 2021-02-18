package config

import (
	"errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DataSync struct {
	config ProductConfig
}

func NewDataSync(config ProductConfig) *DataSync {
	return &DataSync{config: config}
}

func (f *DataSync) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{}
}

func (f *DataSync) GetNamespace() string {
	return f.config["NAMESPACE"]
}

func (f *DataSync) SetNamespace(newNamespace string) {
	f.config["NAMESPACE"] = newNamespace
}

func (f *DataSync) Read() ProductConfig {
	return f.config
}

func (f *DataSync) GetHost() string {
	return f.config["HOST"]
}

func (f *DataSync) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductDataSync
}

func (f *DataSync) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionDataSync
}

func (f *DataSync) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return ""
}

func (f *DataSync) Validate() error {
	if f.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if f.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}

	if f.GetProductVersion() == "" {
		return errors.New("config product version is not defined")
	}

	return nil
}
