package config

import (
	"errors"

	testResources "github.com/integr8ly/integreatly-operator/test/resources"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

type RHSSO struct {
	*RHSSOCommon
}

func NewRHSSO(config ProductConfig) *RHSSO {
	return &RHSSO{&RHSSOCommon{Config: config}}
}

func (r *RHSSO) GetLabelSelector() string {
	return "middleware"
}

func (r *RHSSO) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductRHSSO
}

func (r *RHSSO) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionRHSSO
}

func (r *RHSSO) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	return r.ValidateCommon()
}

func (r *RHSSO) GetReplicasConfig(inst *integreatlyv1alpha1.RHMI) int {
	if testResources.RunningInProw(inst) {
		return 1
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(inst.Spec.Type)) {
		return 3
	}
	return 2
}
