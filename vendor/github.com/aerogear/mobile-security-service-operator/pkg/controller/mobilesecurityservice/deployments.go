package mobilesecurityservice

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//buildDeployment returns the Deployment object using as image the MobileSecurityService App ( UI + REST API)
func (r *ReconcileMobileSecurityService) buildDeployment(mss *mobilesecurityservicev1alpha1.MobileSecurityService) *appsv1.Deployment {

	ls := getAppLabels(mss.Name)
	replicas := mss.Spec.Size
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mss.Name,
			Namespace: mss.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: mss.Name,
					Containers:         getDeploymentContainers(mss),
				},
			},
		},
	}

	// Set MobileSecurityService mss as the owner and controller
	controllerutil.SetControllerReference(mss, dep, r.scheme)
	return dep
}

func getDeploymentContainers(mss *mobilesecurityservicev1alpha1.MobileSecurityService) []corev1.Container {
	var containers []corev1.Container
	containers = append(containers, buildOAuthContainer(mss))
	containers = append(containers, buildApplicationContainer(mss))
	return containers
}

func buildOAuthContainer(mss *mobilesecurityservicev1alpha1.MobileSecurityService) corev1.Container {
	return corev1.Container{
		Image:           mss.Spec.OAuthImage,
		Name:            mss.Spec.OAuthContainerName,
		ImagePullPolicy: mss.Spec.OAuthContainerImagePullPolicy,
		Ports: []corev1.ContainerPort{{
			ContainerPort: oauthProxyPort,
			Name:          "public",
			Protocol:      "TCP",
		}},
		Args:                     getOAuthArgsMap(mss),
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(mss.Spec.OAuthMemoryLimit),
				corev1.ResourceCPU:    resource.MustParse(mss.Spec.OAuthResourceCpuLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(mss.Spec.OAuthMemoryRequest),
				corev1.ResourceCPU:    resource.MustParse(mss.Spec.OAuthResourceCpu),
			},
		},
	}
}

func buildApplicationContainer(mss *mobilesecurityservicev1alpha1.MobileSecurityService) corev1.Container {
	environment := buildAppEnvVars(mss)
	return corev1.Container{
		Image:           mss.Spec.Image,
		Name:            mss.Spec.ContainerName,
		ImagePullPolicy: mss.Spec.ContainerImagePullPolicy,
		Ports: []corev1.ContainerPort{{
			ContainerPort: mss.Spec.Port,
			Name:          "http",
			Protocol:      "TCP",
		}},
		// Get the value from the ConfigMap
		Env: *environment,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/api/healthz",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: mss.Spec.Port,
					},
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 10,
			FailureThreshold:    3,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/api/ping",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: mss.Spec.Port,
					},
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 10,
			FailureThreshold:    3,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(mss.Spec.MemoryLimit),
				corev1.ResourceCPU:    resource.MustParse(mss.Spec.ResourceCpuLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(mss.Spec.MemoryRequest),
				corev1.ResourceCPU:    resource.MustParse(mss.Spec.ResourceCpu),
			},
		},
	}
}
