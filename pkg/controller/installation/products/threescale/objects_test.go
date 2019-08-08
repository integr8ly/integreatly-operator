package threescale

import (
	"bytes"
	"fmt"

	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	appsv1 "github.com/openshift/api/apps/v1"
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

var installPlanFor3ScaleSubscription = &coreosv1alpha1.InstallPlan{
	ObjectMeta: metav1.ObjectMeta{
		Name: "installplan-for-3scale",
	},
	Status: coreosv1alpha1.InstallPlanStatus{
		Phase: coreosv1alpha1.InstallPlanPhaseComplete,
	},
}

var s3BucketSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: s3BucketSecretName,
	},
}

var s3CredentialsSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: s3CredentialsSecretName,
	},
}

var keycloakrealm = &aerogearv1.KeycloakRealm{
	ObjectMeta: metav1.ObjectMeta{
		Name:      testRhssoRealm,
		Namespace: testRhssoNamespace,
	},
	Spec: aerogearv1.KeycloakRealmSpec{
		KeycloakApiRealm: &aerogearv1.KeycloakApiRealm{
			Users: []*aerogearv1.KeycloakUser{
				rhsso.CustomerAdminUser,
			},
			Clients: []*aerogearv1.KeycloakClient{},
		},
	},
}

var threeScaleAdminDetailsSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-seed",
	},
	Data: map[string][]byte{
		"ADMIN_USER":  bytes.NewBufferString(threeScaleAdminUser.UserDetails.Username).Bytes(),
		"ADMIN_EMAIL": bytes.NewBufferString(threeScaleAdminUser.UserDetails.Email).Bytes(),
	},
}

var threeScaleServiceDiscoveryConfigMap = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system",
	},
	Data: map[string]string{
		"service_discovery.yml": "",
	},
}

var threeScaleAdminUser = &User{
	UserDetails: UserDetails{
		Email:    "not" + rhsso.CustomerAdminUser.Email,
		Username: "not" + rhsso.CustomerAdminUser.UserName,
	},
}

var systemApp = appsv1.DeploymentConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-app",
	},
	Status: appsv1.DeploymentConfigStatus{
		LatestVersion: 1,
	},
}

var systemSidekiq = appsv1.DeploymentConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-sidekiq",
	},
	Status: appsv1.DeploymentConfigStatus{
		LatestVersion: 1,
	},
}

var successfulTestAppsV1Objects = map[string]*appsv1.DeploymentConfig{
	systemApp.Name:     &systemApp,
	systemSidekiq.Name: &systemSidekiq,
}

var cm = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-environment",
		Namespace: defaultInstallationNamespace,
	},
}

func getSuccessfullTestPreReqs(integreatlyOperatorNamespace, threeScaleInstallationNamepsace string) []runtime.Object {
	configManagerConfigMap.Namespace = integreatlyOperatorNamespace
	s3BucketSecret.Namespace = integreatlyOperatorNamespace
	s3CredentialsSecret.Namespace = integreatlyOperatorNamespace
	threeScaleAdminDetailsSecret.Namespace = threeScaleInstallationNamepsace
	threeScaleServiceDiscoveryConfigMap.Namespace = threeScaleInstallationNamepsace
	cm.Namespace = threeScaleInstallationNamepsace

	return []runtime.Object{
		s3BucketSecret,
		s3CredentialsSecret,
		keycloakrealm,
		configManagerConfigMap,
		threeScaleAdminDetailsSecret,
		threeScaleServiceDiscoveryConfigMap,
		cm,
	}
}
