package mobilesecurityservice

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

//buildAppIngress returns the ingress/route resource
func (r *ReconcileMobileSecurityService) buildAppIngress(m *mobilesecurityservicev1alpha1.MobileSecurityService) *v1beta1.Ingress {
	ls := getAppLabels(m.Name)
	ing := &v1beta1.Ingress{
		TypeMeta: v1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels:    ls,
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: m.Name,
				ServicePort: intstr.FromInt(int(m.Spec.Port)),
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: utils.GetAppIngress(m.Spec.ClusterHost, m.Spec.HostSufix),
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Backend: v1beta1.IngressBackend{
										ServiceName: m.Name,
										ServicePort: intstr.FromInt(int(m.Spec.Port)),
									},
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
	}

	// Set MobileSecurityService instance as the owner and controller
	controllerutil.SetControllerReference(m, ing, r.scheme)
	return ing
}