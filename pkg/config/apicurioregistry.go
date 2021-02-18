package config

import (
	apicurioregistry "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ApicurioRegistry struct {
	Config ProductConfig
}

func NewApicurioRegistry(config ProductConfig) *ApicurioRegistry {
	return &ApicurioRegistry{Config: config}
}

//GetWatchableCRDs to trigger a reconcile of the integreatly installation when these are updated
func (r *ApicurioRegistry) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&apicurioregistry.ApicurioRegistry{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ApicurioRegistry",
				APIVersion: apicurioregistry.SchemeGroupVersion.String(),
			},
		},
	}
}

func (r *ApicurioRegistry) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *ApicurioRegistry) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *ApicurioRegistry) GetOperatorNamespace() string {
	return r.Config["OPERATOR_NAMESPACE"]
}

func (r *ApicurioRegistry) SetOperatorNamespace(newNamespace string) {
	r.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (r *ApicurioRegistry) Read() ProductConfig {
	return r.Config
}

func (r *ApicurioRegistry) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductApicurioRegistry
}

func (r *ApicurioRegistry) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionApicurioRegistry
}

func (r *ApicurioRegistry) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionApicurioRegistry
}

func (c *ApicurioRegistry) GetHost() string {
	return c.Config["HOST"]
}

func (c *ApicurioRegistry) SetHost(newHost string) {
	c.Config["HOST"] = newHost
}
