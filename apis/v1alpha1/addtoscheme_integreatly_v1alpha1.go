package v1alpha1

import (
	obo "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	envoyconfigv1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	discoveryservicev1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	prometheusmonitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	consolev1 "github.com/openshift/api/console/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"

	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	authv1 "github.com/openshift/api/authorization/v1"
	confv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
	usersv1 "github.com/openshift/api/user/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"

	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"

	cloudcredentialv1 "github.com/openshift/api/operator/v1"

	addonv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(
		AddToSchemes,
		monv1.AddToScheme,
		operatorsv1.AddToScheme,
		packageOperatorv1alpha1.AddToScheme,
		operatorsv1alpha1.AddToScheme,
		authv1.Install,
		keycloak.SchemeBuilder.AddToScheme,
		threescalev1.SchemeBuilder.AddToScheme,
		obo.SchemeBuilder.AddToScheme,
		crov1.SchemeBuilder.AddToScheme,
		routev1.Install,
		appsv1.Install,
		imagev1.Install,
		oauthv1.Install,
		templatev1.Install,
		rbacv1.SchemeBuilder.AddToScheme,
		usersv1.Install,
		confv1.Install,
		prometheusmonitoringv1.SchemeBuilder.AddToScheme,
		projectv1.Install,
		consolev1.Install,
		envoyconfigv1.SchemeBuilder.AddToScheme,
		discoveryservicev1.SchemeBuilder.AddToScheme,
		apiextensionv1beta1.SchemeBuilder.AddToScheme,
		apiextensionv1.SchemeBuilder.AddToScheme,
		customdomainv1alpha1.AddToScheme,
		cloudcredentialv1.Install,
		addonv1alpha1.AddToScheme,
	)
}
