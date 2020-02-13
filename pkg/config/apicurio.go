package config

import (
	"errors"
	apicurio "github.com/integr8ly/integreatly-operator/pkg/apis/apicur/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Apicurio struct {
	Config ProductConfig
}

func NewApicurio(config ProductConfig) *Apicurio {
	return &Apicurio{Config: config}
}

//GetWatchableCRDs to trigger a reconcile of the integreatly installation when these are updated
func (r *Apicurio) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&apicurio.Apicurito{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Apicurito",
				APIVersion: apicurio.SchemeGroupVersion.String(),
			},
		},
	}
}

func (r *Apicurio) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *Apicurio) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *Apicurio) GetOperatorNamespace() string {
	return r.Config["NAMESPACE"] + "-operator"
}

func (r *Apicurio) Read() ProductConfig {
	return r.Config
}

func (r *Apicurio) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductRHSSO
}

func (r *Apicurio) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(r.Config["VERSION"])
}

func (r *Apicurio) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionRHSSO
}

func (r *Apicurio) SetProductVersion(version string) {
	r.Config["VERSION"] = version
}

func (r *Apicurio) SetOperatorVersion(operator string) {
	r.Config["OPERATOR"] = operator
}

func (c *Apicurio) GetHost() string {
	return c.Config["HOST"]
}

func (c *Apicurio) SetHost(newHost string) {
	c.Config["HOST"] = newHost
}

func (r *Apicurio) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if r.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	return nil
}
