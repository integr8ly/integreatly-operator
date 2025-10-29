package config

import (
	"errors"

	"github.com/integr8ly/integreatly-operator/utils"

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
	if utils.RunningInProw(inst) {
		return 1
	}
	return 2
}
