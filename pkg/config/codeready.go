package config

import (
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type CodeReady struct {
	Config ProductConfig
}

func NewCodeReady(config ProductConfig) *CodeReady {
	return &CodeReady{Config: config}
}
func (c *CodeReady) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&chev1.CheCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CheCluster",
				APIVersion: chev1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (c *CodeReady) GetHost() string {
	return c.Config["HOST"]
}

func (c *CodeReady) SetHost(newHost string) {
	c.Config["HOST"] = newHost
}

func (c *CodeReady) GetNamespace() string {
	return c.Config["NAMESPACE"]
}

func (c *CodeReady) GetOperatorNamespace() string {
	return c.Config["OPERATOR_NAMESPACE"]
}

func (c *CodeReady) SetOperatorNamespace(newNamespace string) {
	c.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (c *CodeReady) GetLabelSelector() string {
	return "middleware"
}

func (c *CodeReady) SetNamespace(newNamespace string) {
	c.Config["NAMESPACE"] = newNamespace
}

func (c *CodeReady) Read() ProductConfig {
	return c.Config
}

func (c *CodeReady) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductCodeReadyWorkspaces
}

func (c *CodeReady) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionCodeReadyWorkspaces
}

func (c *CodeReady) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionCodeReadyWorkspaces
}

func (c *CodeReady) GetBackupsSecretName() string {
	return "backups-s3-credentials"
}

func (c *CodeReady) GetPostgresBackupSecretName() string {
	return "codeready-postgres-secret"
}

func (c *CodeReady) GetBackupSchedule() string {
	return "30 2 * * *"
}
