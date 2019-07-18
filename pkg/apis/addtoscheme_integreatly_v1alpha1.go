package apis

import (
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	enmasseAdmin "github.com/enmasseproject/enmasse/pkg/apis/admin/v1beta1"
	enmasse "github.com/enmasseproject/enmasse/pkg/apis/enmasse/v1beta1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(
		AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
		operatorsv1alpha1.AddToScheme,
		operatorsv1.AddToScheme,
		marketplacev1.SchemeBuilder.AddToScheme,
		kafkav1.SchemeBuilder.AddToScheme,
		aerogearv1.SchemeBuilder.AddToScheme,
		chev1.SchemeBuilder.AddToScheme,
		syn.SchemeBuilder.AddToScheme,
		enmasse.SchemeBuilder.AddToScheme,
		enmassev1beta2.SchemeBuilder.AddToScheme,
		enmassev1beta1.SchemeBuilder.AddToScheme,
		enmasseAdmin.SchemeBuilder.AddToScheme,
	)
}
