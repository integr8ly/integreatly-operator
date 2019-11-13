package launcher

import (
	"fmt"
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	testRhssoNamespace = "test-rhsso"
	testRhssoRealm     = "test-realm"
	testRhssoURL       = "https://test.rhsso.url"
)

var configManagerConfigMap = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "integreatly-installation-config",
	},
	Data: map[string]string{
		"rhsso": fmt.Sprintf("NAMESPACE: %s\nREALM: %s\nURL: %s", testRhssoNamespace, testRhssoRealm, testRhssoURL),
	},
}

var installPlanForLauncherSubscription = &coreosv1alpha1.InstallPlan{
	ObjectMeta: metav1.ObjectMeta{
		Name: "installplan-for-launcher",
	},
	Status: coreosv1alpha1.InstallPlanStatus{
		Phase: coreosv1alpha1.InstallPlanPhaseComplete,
	},
}

func getClusterPreReqObjects(integreatlyOperatorNamespace string) []runtime.Object {
	configManagerConfigMap.Namespace = integreatlyOperatorNamespace

	return []runtime.Object{
		configManagerConfigMap,
		launcherConfigMap,
		mockLauncherRoute,
	}
}

var mockLauncherRoute = &routev1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "launcher",
		Namespace: defaultInstallationNamespace,
	},
	Spec: routev1.RouteSpec{
		Host: "example.com",
	},
}

var launcherDeploymentConfigs = []runtime.Object{
	&appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "launcher-application",
			Namespace: defaultInstallationNamespace,
			Labels: map[string]string{
				"app": "fabric8-launcher",
			},
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
		},
		Status: appsv1.DeploymentConfigStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
	},
}

var launcherConfigMap = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      defaultLauncherConfigMapName,
		Namespace: defaultInstallationNamespace,
	},
	Data: map[string]string{
		"launcher.keycloak.client.id": "",
	},
}
