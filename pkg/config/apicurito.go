package config

import (
	"errors"

	apicuritov1alpha1 "github.com/apicurio/apicurio-operators/apicurito/pkg/apis/apicur/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Apicurito struct {
	Config ProductConfig
}

func NewApicurito(config ProductConfig) *Apicurito {
	return &Apicurito{Config: config}
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
	return r.Config["NAMESPACE"]
}

func (r *Apicurito) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *Apicurito) GetOperatorNamespace() string {
	return r.Config["NAMESPACE"] + "-operator"
}

func (r *Apicurito) Read() ProductConfig {
	return r.Config
}

func (r *Apicurito) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductApicurito
}

func (r *Apicurito) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(r.Config["VERSION"])
}

func (r *Apicurito) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionApicurito
}

func (r *Apicurito) SetProductVersion(version string) {
	r.Config["VERSION"] = version
}

func (r *Apicurito) SetOperatorVersion(operator string) {
	r.Config["OPERATOR"] = operator
}

func (c *Apicurito) GetHost() string {
	return c.Config["HOST"]
}

func (c *Apicurito) SetHost(newHost string) {
	c.Config["HOST"] = newHost
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
