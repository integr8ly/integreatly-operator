package mobilesecurityservice

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/api/extensions/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//buildAppDeployment returns the Deployment object using as image the MobileSecurityService App ( UI + REST API)
func (r *ReconcileMobileSecurityService) buildAppDeployment(m *mobilesecurityservicev1alpha1.MobileSecurityService) *v1beta1.Deployment {
	ls := getAppLabels(m.Name)
	replicas := m.Spec.Size
	envinronment := buildAppEnvVars(m)
	dep := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels:    ls,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: v1beta1.DeploymentStrategy{
				Type: v1beta1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           m.Spec.Image,
						Name:            m.Spec.ContainerName,
						ImagePullPolicy: corev1.PullAlways,
						Ports: []corev1.ContainerPort{{
							ContainerPort: m.Spec.Port,
							Name:          "http",
							Protocol:      "TCP",
						}},
						// Get the value from the ConfigMap
						Env: *envinronment,
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/api/healthz",
									Port: intstr.IntOrString{
										Type:   intstr.Int,
										IntVal: m.Spec.Port,
									},
									Scheme: corev1.URISchemeHTTP,
								},
							},
							FailureThreshold: 3,
							InitialDelaySeconds: 5,
							PeriodSeconds: 10,
							TimeoutSeconds:      1,
							SuccessThreshold: 1,
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/api/ping",
									Port: intstr.IntOrString{
										Type:   intstr.Int,
										IntVal: m.Spec.Port,
									},
									Scheme: corev1.URISchemeHTTP,
								},
							},
							FailureThreshold: 3,
							InitialDelaySeconds: 120,
							PeriodSeconds: 10,
							TimeoutSeconds:      10,
							SuccessThreshold: 1,
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse(m.Spec.MemoryLimit),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse(m.Spec.MemoryRequest),
							},
						},
					}},
				},
			},
		},
	}
	// Set MobileSecurityService instance as the owner and controller
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}
