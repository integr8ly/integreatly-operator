package config

import (
	"errors"

	pushv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

type Ups struct {
	config ProductConfig
}

func NewUps(config ProductConfig) *Ups {
	return &Ups{config: config}
}
func (u *Ups) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&pushv1alpha1.UnifiedPushServer{
			TypeMeta: metav1.TypeMeta{
				Kind:       "UnifiedPushServer",
				APIVersion: pushv1alpha1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (u *Ups) GetHost() string {
	return u.config["HOST"]
}

func (u *Ups) SetHost(newHost string) {
	u.config["HOST"] = newHost
}

func (u *Ups) GetBlackboxTargetPath() string {
	return u.config["BLACKBOX_TARGET_PATH"]
}

func (u *Ups) SetBlackboxTargetPath(newBlackboxTargetPath string) {
	u.config["BLACKBOX_TARGET_PATH"] = newBlackboxTargetPath
}

func (u *Ups) GetNamespace() string {
	return u.config["NAMESPACE"]
}

func (u *Ups) SetNamespace(newNamespace string) {
	u.config["NAMESPACE"] = newNamespace
}

func (u *Ups) GetOperatorNamespace() string {
	return u.config["OPERATOR_NAMESPACE"]
}

func (u *Ups) SetOperatorNamespace(newNamespace string) {
	u.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (u *Ups) Read() ProductConfig {
	return u.config
}

func (u *Ups) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductUps
}

func (u *Ups) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionUps
}

func (u *Ups) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionUPS
}

func (u *Ups) Validate() error {
	if u.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if u.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if u.GetHost() == "" {
		return errors.New("config host is not defined")
	}
	return nil
}
