package config

import (
	"errors"

	enmasseadminv1beta1 "github.com/integr8ly/integreatly-operator/apis-products/enmasse/admin/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/apis-products/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/apis-products/enmasse/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

type AMQOnline struct {
	config ProductConfig
}

func NewAMQOnline(config ProductConfig) *AMQOnline {
	return &AMQOnline{config: config}
}

func (a *AMQOnline) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&enmassev1beta2.AddressPlan{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AddressPlan",
				APIVersion: enmassev1beta2.SchemeGroupVersion.String(),
			},
		},
		&enmassev1beta2.AddressSpacePlan{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AddressSpacePlan",
				APIVersion: enmassev1beta2.SchemeGroupVersion.String(),
			},
		},
		&enmassev1beta1.BrokeredInfraConfig{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BrokeredInfraConfig",
				APIVersion: enmassev1beta1.SchemeGroupVersion.String(),
			},
		},
		&enmasseadminv1beta1.AuthenticationService{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AuthenticationService",
				APIVersion: enmasseadminv1beta1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (a *AMQOnline) GetHost() string {
	return a.config["HOST"]
}

func (a *AMQOnline) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *AMQOnline) GetBlackboxTargetPath() string {
	return a.config["BLACKBOX_TARGET_PATH"]
}

func (a *AMQOnline) SetBlackboxTargetPath(newBlackboxTargetPath string) {
	a.config["BLACKBOX_TARGET_PATH"] = newBlackboxTargetPath
}

func (a *AMQOnline) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *AMQOnline) GetOperatorNamespace() string {
	return a.config["OPERATOR_NAMESPACE"]
}

func (a *AMQOnline) SetOperatorNamespace(newNamespace string) {
	a.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (a *AMQOnline) GetLabelSelector() string {
	return "middleware"
}

func (a *AMQOnline) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *AMQOnline) Read() ProductConfig {
	return a.config
}

func (a *AMQOnline) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductAMQOnline
}

func (a *AMQOnline) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionAMQOnline
}

func (a *AMQOnline) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionAMQOnline
}

func (a *AMQOnline) GetBackupsSecretName() string {
	return "backups-s3-credentials"
}

func (c *AMQOnline) GetPostgresBackupSecretName() string {
	return "enmasse-postgres-secret"
}

func (a *AMQOnline) GetBackupSchedule() string {
	return "30 2 * * *"
}

func (a *AMQOnline) Validate() error {
	if a.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if a.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if a.GetHost() == "" {
		return errors.New("config host is not defined")
	}
	return nil
}
