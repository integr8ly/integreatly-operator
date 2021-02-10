package config

import (
	"errors"

	syndesisv1beta1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

func NewFuse(config ProductConfig) *Fuse {
	return &Fuse{config: config}
}

type Fuse struct {
	config ProductConfig
}

func (f *Fuse) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&syndesisv1beta1.Syndesis{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Syndesis",
				APIVersion: syndesisv1beta1.SchemeGroupVersion.String(),
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

func (f *Fuse) GetOperatorNamespace() string {
	return f.config["OPERATOR_NAMESPACE"]
}

func (f *Fuse) SetOperatorNamespace(newNamespace string) {
	f.config["OPERATOR_NAMESPACE"] = newNamespace
}
func (f *Fuse) GetBlackboxTargetPath() string {
	return f.config["BLACKBOX_TARGET_PATH"]
}
func (f *Fuse) SetBlackboxTargetPath(newBlackboxTargetPath string) {
	f.config["BLACKBOX_TARGET_PATH"] = newBlackboxTargetPath
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
		"fuseonline/addon-ops-api-dashboard.yml",
		"fuseonline/addon-ops-home-dashboard.yml",
		"fuseonline/addon-ops-integrations-alerting-rules.yml",
		"fuseonline/addon-ops-integrations-camel-dashboard.yml",
		"fuseonline/addon-ops-integrations-home-dashboard.yml",
		"fuseonline/addon-ops-integrations-jvm-dashboard.yml",
		"fuseonline/addon-ops-integrations-service.yml",
		"fuseonline/addon-ops-integrations-servicemonitor.yml",
		"fuseonline/addon-ops-jvm-dashboard.yml",
		"fuseonline/addon-ops-meta-alerting-rules.yml",
		"fuseonline/addon-ops-server-alerting-rules.yml",
		"fuseonline/addon-ops-servicemonitor.yml",
	}
	return templateList
}

func (f *Fuse) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductFuse
}

func (f *Fuse) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionFuseOnline
}

func (f *Fuse) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionFuse
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
