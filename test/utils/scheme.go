package utils

import (
	threescaleAppsv1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	keycloakv1alpha1 "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	configv1 "github.com/openshift/api/config/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// NewTestScheme returns a scheme for use in unit tests
func NewTestScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := threescaleAppsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := keycloakv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := olmv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := olmv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := usersv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := oauthv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := routev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := crov1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := monitoringv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := admissionv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := openshiftappsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rbacv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := batchv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := batchv1beta1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := configv1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}
