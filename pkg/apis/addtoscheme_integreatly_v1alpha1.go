package apis

import (
	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	syndesisv1alpha1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	apicurio "github.com/integr8ly/integreatly-operator/pkg/apis/apicur/v1alpha1"
	enmasseAdmin "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/admin/v1beta1"
	enmasse "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/enmasse/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	solutionExplorer "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	monitoring "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"

	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
	usersv1 "github.com/openshift/api/user/v1"
	samplesv1 "github.com/openshift/cluster-samples-operator/pkg/apis/samples/v1"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	marketplacev2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"

	rbacv1 "k8s.io/api/rbac/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(
		AddToSchemes,
		integreatlyv1alpha1.SchemeBuilder.AddToScheme,
		operatorsv1alpha1.AddToScheme,
		operatorsv1.AddToScheme,
		marketplacev1.SchemeBuilder.AddToScheme,
		marketplacev2.SchemeBuilder.AddToScheme,
		kafkav1.SchemeBuilder.AddToScheme,
		keycloak.SchemeBuilder.AddToScheme,
		chev1.SchemeBuilder.AddToScheme,
		syndesisv1alpha1.SchemeBuilder.AddToScheme,
		enmasse.SchemeBuilder.AddToScheme,
		enmassev1beta2.SchemeBuilder.AddToScheme,
		enmassev1beta1.SchemeBuilder.AddToScheme,
		enmasseAdmin.SchemeBuilder.AddToScheme,
		threescalev1.SchemeBuilder.AddToScheme,
		solutionExplorer.SchemeBuilder.AddToScheme,
		monitoring.SchemeBuilder.AddToScheme,
		grafanav1alpha1.SchemeBuilder.AddToScheme,
		crov1.SchemeBuilder.AddToScheme,
		apicurio.SchemeBuilder.AddToScheme,
		routev1.AddToScheme,

		appsv1.AddToScheme,
		imagev1.AddToScheme,
		oauthv1.AddToScheme,
		templatev1.AddToScheme,
		rbacv1.SchemeBuilder.AddToScheme,
		usersv1.AddToScheme,
		samplesv1.SchemeBuilder.AddToScheme,
		upsv1alpha1.SchemeBuilder.AddToScheme,
		prometheusmonitoringv1.SchemeBuilder.AddToScheme,
		projectv1.AddToScheme,
	)
}
