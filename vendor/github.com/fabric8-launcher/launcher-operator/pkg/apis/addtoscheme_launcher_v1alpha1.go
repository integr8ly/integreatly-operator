package apis

import (
	"fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"

	"github.com/integr8ly/operator-sdk-openshift-utils/pkg/api/schemes"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha2.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, schemes.AddToScheme)
}
