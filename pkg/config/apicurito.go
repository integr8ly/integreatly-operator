package config

import (
	"errors"

	apicuritov1alpha1 "github.com/apicurio/apicurio-operators/apicurito/pkg/apis/apicur/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Apicurito struct {
	config ProductConfig
}

func NewApicurito(config ProductConfig) *Apicurito {
	return &Apicurito{config: config}
}

//GetWatchableCRDs to trigger a reconcile of the integreatly installation when these are updated
func (r *Apicurito) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&apicuritov1alpha1.Apicurito{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Apicurito",
				APIVersion: apicuritov1alpha1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (r *Apicurito) GetNamespace() string {
	return r.config["NAMESPACE"]
}

func (r *Apicurito) SetNamespace(newNamespace string) {
	r.config["NAMESPACE"] = newNamespace
}
func (r *Apicurito) GetBlackboxTargetPath() string {
	return r.config["BLACKBOX_TARGET_PATH"]
}
func (r *Apicurito) SetBlackboxTargetPath(newBlackboxTargetPath string) {
	r.config["BLACKBOX_TARGET_PATH"] = newBlackboxTargetPath
}

func (r *Apicurito) GetOperatorNamespace() string {
	return r.config["NAMESPACE"] + "-operator"
}

func (r *Apicurito) SetOperatorNamespace(newNamespace string) {
	r.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (r *Apicurito) Read() ProductConfig {
	return r.config
}

func (r *Apicurito) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductApicurito
}

func (r *Apicurito) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionApicurito
}

func (r *Apicurito) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionApicurito
}

func (c *Apicurito) GetHost() string {
	return c.config["HOST"]
}

func (c *Apicurito) SetHost(newHost string) {
	c.config["HOST"] = newHost
}

func (r *Apicurito) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if r.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	return nil
}
