package solutionexplorer

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	webappv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var webappNs = &corev1.Namespace{
	ObjectMeta: v1.ObjectMeta{
		Name: defaultName,
		OwnerReferences: []v1.OwnerReference{
			{
				Name:       installation.Name,
				APIVersion: installation.APIVersion,
			},
		},
	},
	Status: corev1.NamespaceStatus{
		Phase: corev1.NamespaceActive,
	},
}

var webappCR = &webappv1alpha1.WebApp{
	ObjectMeta: v1.ObjectMeta{
		Namespace: "solution-explorer",
		Name:      "solution-explorer",
	},
	Status: webappv1alpha1.WebAppStatus{
		Message: "OK",
	},
}

var webappRoute = &routev1.Route{
	ObjectMeta: v1.ObjectMeta{
		Name:      defaultRouteName,
		Namespace: defaultName,
	},
}

var webappDC = &appsv1.DeploymentConfig{
	ObjectMeta: v1.ObjectMeta{
		Name:      defaultDeploymentConfigName,
		Namespace: defaultName,
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						Name: "tutorial-web-app",
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "sample-env-var",
								Value: "sample-value",
							},
						},
					},
				},
			},
		},
	},
}

var installation = &v1alpha1.Installation{
	TypeMeta: v1.TypeMeta{
		Kind:       "Installation",
		APIVersion: v1alpha1.SchemeGroupVersion.String(),
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "example-installation",
		Namespace: "integreatly-operator",
	},
	Status: v1alpha1.InstallationStatus{
		Stages: map[v1alpha1.StageName]*v1alpha1.InstallationStageStatus{
			"products": &v1alpha1.InstallationStageStatus{
				Name:  "products",
				Phase: v1alpha1.PhaseCompleted,
				Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
					v1alpha1.ProductLauncher: &v1alpha1.InstallationProductStatus{
						Name:    v1alpha1.ProductLauncher,
						Host:    "http://launcher.example.com",
						Status:  v1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					v1alpha1.ProductFuse: &v1alpha1.InstallationProductStatus{
						Name:    v1alpha1.ProductFuse,
						Host:    "http://syndesis.example.com",
						Status:  v1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					v1alpha1.ProductRHSSOUser: &v1alpha1.InstallationProductStatus{
						Name:    v1alpha1.ProductRHSSOUser,
						Host:    "http://sso.example.com",
						Status:  v1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					v1alpha1.ProductCodeReadyWorkspaces: &v1alpha1.InstallationProductStatus{
						Name:    v1alpha1.ProductCodeReadyWorkspaces,
						Host:    "http://codeready.example.com",
						Status:  v1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					v1alpha1.ProductAMQStreams: &v1alpha1.InstallationProductStatus{
						Name:    v1alpha1.ProductCodeReadyWorkspaces,
						Host:    "",
						Status:  v1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					v1alpha1.Product3Scale: &v1alpha1.InstallationProductStatus{
						Name:    v1alpha1.Product3Scale,
						Host:    "http://3scale.example.com",
						Status:  v1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
				},
			},
		},
	},
}
