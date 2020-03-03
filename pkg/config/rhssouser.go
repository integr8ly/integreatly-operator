package config

import (
	"errors"
	"strconv"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type RHSSOUser struct {
	Config ProductConfig
}

func NewRHSSOUser(config ProductConfig) *RHSSOUser {
	return &RHSSOUser{Config: config}
}
func (r *RHSSOUser) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&keycloak.Keycloak{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Keycloak",
				APIVersion: keycloak.SchemeGroupVersion.String(),
			},
		},
		&keycloak.KeycloakRealm{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KeycloakRealm",
				APIVersion: keycloak.SchemeGroupVersion.String(),
			},
		},
		&keycloak.KeycloakUser{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KeycloakUser",
				APIVersion: keycloak.SchemeGroupVersion.String(),
			},
		},
		&keycloak.KeycloakClient{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KeycloakClient",
				APIVersion: keycloak.SchemeGroupVersion.String(),
			},
		},
		&keycloak.KeycloakBackup{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KeycloakBackup",
				APIVersion: keycloak.SchemeGroupVersion.String(),
			},
		},
	}
}

func (r *RHSSOUser) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *RHSSOUser) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *RHSSOUser) GetOperatorNamespace() string {
	return r.Config["OPERATOR_NAMESPACE"]
}

func (r *RHSSOUser) SetOperatorNamespace(newNamespace string) {
	r.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (r *RHSSOUser) GetRealm() string {
	return r.Config["REALM"]
}

func (r *RHSSOUser) SetRealm(newRealm string) {
	r.Config["REALM"] = newRealm
}

func (r *RHSSOUser) GetHost() string {
	return r.Config["HOST"]
}

func (r *RHSSOUser) SetHost(newHost string) {
	r.Config["HOST"] = newHost
}

func (r *RHSSOUser) Read() ProductConfig {
	return r.Config
}

func (r *RHSSOUser) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductRHSSOUser
}

func (r *RHSSOUser) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(r.Config["VERSION"])
}

func (r *RHSSOUser) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionRHSSOUser
}

func (r *RHSSOUser) SetProductVersion(version string) {
	r.Config["VERSION"] = version
}

func (r *RHSSOUser) SetOperatorVersion(operator string) {
	r.Config["OPERATOR"] = operator
}

func (r *RHSSOUser) SetDevelopersGroupConfigured(configured bool) {
	r.Config["DEVELOPERS_GROUP_CONFIGURED"] = strconv.FormatBool(configured)
}

func (r *RHSSOUser) GetDevelopersGroupConfigured() (bool, error) {
	if r.Config["DEVELOPERS_GROUP_CONFIGURED"] == "" {
		return false, nil
	}
	return strconv.ParseBool(r.Config["DEVELOPERS_GROUP_CONFIGURED"])
}

func (r *RHSSOUser) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if r.GetRealm() == "" {
		return errors.New("config realm is not defined")
	}
	if r.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if r.GetHost() == "" {
		return errors.New("config url is not defined")
	}
	return nil
}
