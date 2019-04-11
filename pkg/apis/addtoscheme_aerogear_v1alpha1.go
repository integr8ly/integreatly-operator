package apis

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	operatorsv1"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme, operatorsv1alpha1.AddToScheme, operatorsv1.AddToScheme)
}
