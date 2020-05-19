package config

import (
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Heimdall struct {
	Config ProductConfig
}

func NewHeimdall(config ProductConfig) *Heimdall {
	return &Heimdall{Config: config}
}
func (c *Heimdall) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&chev1.CheCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CheCluster",
				APIVersion: chev1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (c *Heimdall) GetHost() string {
	return c.Config["HOST"]
}

func (c *Heimdall) SetHost(newHost string) {
	c.Config["HOST"] = newHost
}

func (c *Heimdall) GetNamespace() string {
	return c.Config["NAMESPACE"]
}

func (c *Heimdall) GetOperatorNamespace() string {
	return c.Config["OPERATOR_NAMESPACE"]
}

func (c *Heimdall) SetOperatorNamespace(newNamespace string) {
	c.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (c *Heimdall) GetLabelSelector() string {
	return "middleware"
}

func (c *Heimdall) SetNamespace(newNamespace string) {
	c.Config["NAMESPACE"] = newNamespace
}

func (c *Heimdall) Read() ProductConfig {
	return c.Config
}

func (c *Heimdall) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductHeimdall
}

func (c *Heimdall) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionHeimdall
}

func (c *Heimdall) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionHeimdall
}
