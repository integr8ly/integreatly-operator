package v1alpha1

import (
	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	envoyconfigv1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	discoveryservicev1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	consolev1 "github.com/openshift/api/console/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"

	apicurioregistry "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
	apicurito "github.com/apicurio/apicurio-operators/apicurito/pkg/apis/apicur/v1alpha1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	syndesisv1beta1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1beta1"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	enmasseAdmin "github.com/integr8ly/integreatly-operator/apis-products/enmasse/admin/v1beta1"
	enmasse "github.com/integr8ly/integreatly-operator/apis-products/enmasse/enmasse/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/apis-products/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/apis-products/enmasse/v1beta2"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/kafka.strimzi.io/v1alpha1"
	solutionExplorerv1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/tutorial-web-app-operator/v1alpha1"

	appsv1 "github.com/openshift/api/apps/v1"
	authv1 "github.com/openshift/api/authorization/v1"
	confv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
	usersv1 "github.com/openshift/api/user/v1"
	samplesv1 "github.com/openshift/cluster-samples-operator/pkg/apis/samples/v1"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(
		AddToSchemes,
		operatorsv1alpha1.AddToScheme,
		operatorsv1.AddToScheme,
		authv1.AddToScheme,
		kafkav1alpha1.SchemeBuilder.AddToScheme,
		keycloak.SchemeBuilder.AddToScheme,
		chev1.SchemeBuilder.AddToScheme,
		syndesisv1beta1.SchemeBuilder.AddToScheme,
		enmasse.SchemeBuilder.AddToScheme,
		enmassev1beta2.SchemeBuilder.AddToScheme,
		enmassev1beta1.SchemeBuilder.AddToScheme,
		enmasseAdmin.SchemeBuilder.AddToScheme,
		threescalev1.SchemeBuilder.AddToScheme,
		solutionExplorerv1alpha1.SchemeBuilder.AddToScheme,
		grafanav1alpha1.SchemeBuilder.AddToScheme,
		crov1.SchemeBuilder.AddToScheme,
		apicurioregistry.SchemeBuilder.AddToScheme,
		apicurito.SchemeBuilder.AddToScheme,
		routev1.AddToScheme,
		monitoring.SchemeBuilder.AddToScheme,
		appsv1.AddToScheme,
		imagev1.AddToScheme,
		oauthv1.AddToScheme,
		templatev1.AddToScheme,
		rbacv1.SchemeBuilder.AddToScheme,
		usersv1.AddToScheme,
		confv1.AddToScheme,
		samplesv1.SchemeBuilder.AddToScheme,
		upsv1alpha1.SchemeBuilder.AddToScheme,
		prometheusmonitoringv1.SchemeBuilder.AddToScheme,
		projectv1.AddToScheme,
		consolev1.AddToScheme,
		envoyconfigv1.SchemeBuilder.AddToScheme,
		discoveryservicev1.SchemeBuilder.AddToScheme,
		apiextensionv1beta1.SchemeBuilder.AddToScheme,
		apiextensionv1.SchemeBuilder.AddToScheme,
	)
}
