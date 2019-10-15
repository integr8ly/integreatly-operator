package mobilesecurityservice

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//buildService returns the service resource
func (r *ReconcileMobileSecurityService) buildProxyService(mss *mobilesecurityservicev1alpha1.MobileSecurityService) *corev1.Service {
	ls := getAppLabels(mss.Name)
	targetPort := intstr.FromInt(oauthProxyPort)
	ser := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.ProxyServiceInstanceName,
			Namespace: mss.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					TargetPort: targetPort,
					Port:       80,
					Name:       "web",
				},
			},
		},
	}
	// Set MobileSecurityService mss as the owner and controller
	controllerutil.SetControllerReference(mss, ser, r.scheme)
	return ser
}

func (r *ReconcileMobileSecurityService) buildApplicationService(mss *mobilesecurityservicev1alpha1.MobileSecurityService) *corev1.Service {
	ls := getAppLabels(mss.Name)
	ser := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.ApplicationServiceInstanceName,
			Namespace: mss.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: mss.Spec.Port,
					Name: "server",
				},
			},
		},
	}
	// Set MobileSecurityService mss as the owner and controller
	controllerutil.SetControllerReference(mss, ser, r.scheme)
	return ser
}
