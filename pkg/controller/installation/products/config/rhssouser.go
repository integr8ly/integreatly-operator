package config

import (
	"errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type RHSSOUser struct {
	Config ProductConfig
}

func NewRHSSOUser(config ProductConfig) *RHSSOUser {
	return &RHSSOUser{Config: config}
}

func (r *RHSSOUser) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *RHSSOUser) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
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
