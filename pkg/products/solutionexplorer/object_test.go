package solutionexplorer

import (
	solutionExplorerv1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/tutorial-web-app-operator/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var webappCR = &solutionExplorerv1alpha1.WebApp{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "solution-explorer",
		Name:      "solution-explorer",
	},
	Status: solutionExplorerv1alpha1.WebAppStatus{
		Message: "OK",
	},
}

var webappRoute = &routev1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      defaultRouteName,
		Namespace: DefaultName,
	},
}

var installation = &integreatlyv1alpha1.RHMI{
	TypeMeta: metav1.TypeMeta{
		Kind:       "RHMI",
		APIVersion: integreatlyv1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example-installation",
		Namespace: "integreatly-operator",
		UID:       types.UID("xyz"),
	},
	Status: integreatlyv1alpha1.RHMIStatus{
		Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
			"products": integreatlyv1alpha1.RHMIStageStatus{
				Name:  "products",
				Phase: integreatlyv1alpha1.PhaseCompleted,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductFuse: {
						Name:    integreatlyv1alpha1.ProductFuse,
						Host:    "http://syndesis.example.com",
						Status:  integreatlyv1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					integreatlyv1alpha1.ProductRHSSOUser: {
						Name:    integreatlyv1alpha1.ProductRHSSOUser,
						Host:    "http://sso.example.com",
						Status:  integreatlyv1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					integreatlyv1alpha1.ProductCodeReadyWorkspaces: {
						Name:    integreatlyv1alpha1.ProductCodeReadyWorkspaces,
						Host:    "http://codeready.example.com",
						Status:  integreatlyv1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					integreatlyv1alpha1.ProductAMQStreams: {
						Name:    integreatlyv1alpha1.ProductCodeReadyWorkspaces,
						Host:    "",
						Status:  integreatlyv1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
					integreatlyv1alpha1.Product3Scale: {
						Name:    integreatlyv1alpha1.Product3Scale,
						Host:    "http://3scale.example.com",
						Status:  integreatlyv1alpha1.PhaseCompleted,
						Version: "0.0.1",
					},
				},
			},
		},
	},
}

var webappNS = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: DefaultName,
		Labels: map[string]string{
			resources.OwnerLabelKey: string(installation.GetUID()),
		},
	},
	Status: corev1.NamespaceStatus{
		Phase: corev1.NamespaceActive,
	},
}

var operatorNS = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: DefaultName + "-operator",
		Labels: map[string]string{
			resources.OwnerLabelKey: string(installation.GetUID()),
		},
	},
	Status: corev1.NamespaceStatus{
		Phase: corev1.NamespaceActive,
	},
}
