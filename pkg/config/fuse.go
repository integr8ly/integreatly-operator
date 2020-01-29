package config

import (
	"errors"

	syndesisv1alpha1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

func NewFuse(config ProductConfig) *Fuse {
	return &Fuse{config: config}
}

type Fuse struct {
	config ProductConfig
}

func (f *Fuse) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&syndesisv1alpha1.Syndesis{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Syndesis",
				APIVersion: syndesisv1alpha1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (f *Fuse) GetNamespace() string {
	return f.config["NAMESPACE"]
}

func (f *Fuse) SetNamespace(newNamespace string) {
	f.config["NAMESPACE"] = newNamespace
}

func (f *Fuse) GetHost() string {
	return f.config["HOST"]
}

func (f *Fuse) SetHost(newHost string) {
	f.config["HOST"] = newHost
}

func (f *Fuse) Read() ProductConfig {
	return f.config
}

func (f *Fuse) GetLabelSelector() string {
	return "middleware"
}

func (r *Fuse) GetTemplateList() []string {
	templateList := []string{
		"kube_state_metrics_fuse_online_alerts.yaml",
	}
	return templateList
}

func (f *Fuse) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductFuse
}

func (f *Fuse) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(f.config["VERSION"])
}

func (f *Fuse) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersion(f.config["VERSION"])
}

func (f *Fuse) SetProductVersion(newVersion string) {
	f.config["VERSION"] = newVersion
}

func (f *Fuse) SetOperatorVersion(operator string) {
	f.config["OPERATOR"] = operator
}

func (f *Fuse) Validate() error {
	if f.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if f.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	return nil
}
