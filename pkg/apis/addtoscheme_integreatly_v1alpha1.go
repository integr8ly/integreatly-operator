package apis

import (
	mobileSecurityService "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	enmasseAdmin "github.com/enmasseproject/enmasse/pkg/apis/admin/v1beta1"
	enmasse "github.com/enmasseproject/enmasse/pkg/apis/enmasse/v1beta1"
	launcherv1alpha2 "github.com/fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"
	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	solutionExplorer "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	monitoring "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	templatev1 "github.com/openshift/api/template/v1"
	samplesv1 "github.com/openshift/cluster-samples-operator/pkg/apis/samples/v1"

	nexusv1 "github.com/integr8ly/integreatly-operator/pkg/apis/gpte/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
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
		threescalev1.SchemeBuilder.AddToScheme,
		solutionExplorer.SchemeBuilder.AddToScheme,
		monitoring.SchemeBuilder.AddToScheme,
		mobileSecurityService.SchemeBuilder.AddToScheme,
		routev1.AddToScheme,

		appsv1.AddToScheme,
		imagev1.AddToScheme,
		oauthv1.AddToScheme,
		templatev1.AddToScheme,
		nexusv1.SchemeBuilder.AddToScheme,
		rbacv1.SchemeBuilder.AddToScheme,
		usersv1.AddToScheme,
		launcherv1alpha2.SchemeBuilder.AddToScheme,
		samplesv1.SchemeBuilder.AddToScheme,
		upsv1alpha1.SchemeBuilder.AddToScheme,
	)
}
