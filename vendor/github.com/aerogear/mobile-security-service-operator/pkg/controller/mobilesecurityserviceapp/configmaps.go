package mobilesecurityserviceapp

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Returns the ConfigMap with the properties used to setup/config the Mobile Security Service Project
func (r *ReconcileMobileSecurityServiceApp) buildAppSDKConfigMap(m *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfigMapName(m),
			Namespace: m.Namespace,
			Labels:    getAppLabels(m.Name),
		},
		Data: getConfigMapSDKForMobileSecurityService(m),
	}
	// Set MobileSecurityService instance as the owner and controller
	controllerutil.SetControllerReference(m, configMap, r.scheme)
	return configMap
}