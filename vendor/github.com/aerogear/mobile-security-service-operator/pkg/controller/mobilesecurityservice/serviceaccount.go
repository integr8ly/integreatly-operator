package mobilesecurityservice

import (
	"fmt"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns the Service with the properties used to setup/config the Mobile Security Service Project
func (r *ReconcileMobileSecurityService) buildServiceAccount(mss *mobilesecurityservicev1alpha1.MobileSecurityService) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        mss.Name,
			Namespace:   mss.Namespace,
			Annotations: buildOauthAnnotationWithRoute(mss.Spec.RouteName),
		},
	}
}

// buildOauthAnnotationWithRoute return required annotations for the Oauth setup
func buildOauthAnnotationWithRoute(route string) map[string]string {
	annotation := fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}", route)
	return map[string]string{
		"serviceaccounts.openshift.io/oauth-redirectreference.mobile-security-service-app": annotation,
	}
}
