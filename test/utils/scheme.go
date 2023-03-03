package utils

import (
	envoyconfigv1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	marin3roperator "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	threescaleAppsv1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	keycloakv1alpha1 "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	clusterloggingv1 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	observabilityv1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// NewTestScheme returns a scheme for use in unit tests
func NewTestScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(
		policyv1.AddToScheme,
		corev1.AddToScheme,
		appsv1.AddToScheme,
		threescaleAppsv1.AddToScheme,
		keycloakv1alpha1.AddToScheme,
		integreatlyv1alpha1.AddToScheme,
		olmv1alpha1.AddToScheme,
		olmv1.AddToScheme,
		usersv1.Install,
		oauthv1.Install,
		routev1.Install,
		projectv1.Install,
		corev1.AddToScheme,
		prometheusv1.AddToScheme,
		admissionv1.AddToScheme,
		openshiftappsv1.Install,
		rbacv1.AddToScheme,
		batchv1.AddToScheme,
		configv1.Install,
		grafanav1alpha1.AddToScheme,
		consolev1.Install,
		marin3roperator.AddToScheme,
		apiextensionv1.AddToScheme,
		customdomainv1alpha1.AddToScheme,
		cloudcredentialv1.Install,
		envoyconfigv1.AddToScheme,
		observabilityv1.AddToScheme,
		crov1.AddToScheme,
		// TODO - Remove when released - https://issues.redhat.com/browse/MGDAPI-5308
		clusterloggingv1.AddToScheme,
	)

	if err := schemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}
