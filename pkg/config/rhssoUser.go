package config

import (
	"errors"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	testResources "github.com/integr8ly/integreatly-operator/test/resources"
)

type RHSSOUser struct {
	*RHSSOCommon
}

func NewRHSSOUser(config ProductConfig) *RHSSOUser {
	return &RHSSOUser{&RHSSOCommon{Config: config}}
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

func (r *RHSSOUser) GetBlackboxTargetPath() string {
	return r.Config["BLACKBOX_TARGET_PATH"]
}

func (r *RHSSOUser) SetBlackboxTargetPath(newBlackboxTargetPath string) {
	r.Config["BLACKBOX_TARGET_PATH"] = newBlackboxTargetPath
}

func (r *RHSSOUser) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductRHSSOUser
}

func (r *RHSSOUser) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionRHSSOUser
}

func (r *RHSSOUser) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	return r.ValidateCommon()
}

func (r *RHSSOUser) GetReplicasConfig(inst *integreatlyv1alpha1.RHMI) int {
	if testResources.RunningInProw(inst) {
		return 1
	}

	if inst.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return 3
	}

	return 2
}
