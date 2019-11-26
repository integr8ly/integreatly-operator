package threescale

import (
	"bytes"
	"fmt"
	v1 "github.com/openshift/api/route/v1"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	appsv1 "github.com/openshift/api/apps/v1"
	usersv1 "github.com/openshift/api/user/v1"
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

var OpenshiftDockerSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      resources.DefaultOriginPullSecretName,
		Namespace: resources.DefaultOriginPullSecretNamespace,
	},
}

var ComponentDockerSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      resources.DefaultOriginPullSecretName,
		Namespace: resources.DefaultOriginPullSecretNamespace,
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

var threeScaleAdminDetailsSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-seed",
	},
	Data: map[string][]byte{
		"ADMIN_USER":  bytes.NewBufferString(threeScaleDefaultAdminUser.UserDetails.Username).Bytes(),
		"ADMIN_EMAIL": bytes.NewBufferString(threeScaleDefaultAdminUser.UserDetails.Email).Bytes(),
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

var threeScaleDefaultAdminUser = &User{
	UserDetails: UserDetails{
		Id:       1,
		Email:    "not" + rhsso.CustomerAdminUser.Email,
		Username: "not" + rhsso.CustomerAdminUser.UserName,
		Role:     adminRole,
	},
}

var rhssoTest1 = &keycloak.KeycloakUser{
	Spec: keycloak.KeycloakUserSpec{
		User: keycloak.KeycloakAPIUser{
			UserName: "test1",
			Email:    "test1@example.com",
		},
	},
}

var rhssoTest2 = &keycloak.KeycloakUser{
	Spec: keycloak.KeycloakUserSpec{
		User: keycloak.KeycloakAPIUser{
			UserName: "test2",
			Email:    "test2@example.com",
		},
	},
}

var testDedicatedAdminsGroup = &usersv1.Group{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dedicated-admins",
	},
	Users: []string{
		rhssoTest1.Spec.User.UserName,
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

var systemEnvConfigMap = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-environment",
		Namespace: defaultInstallationNamespace,
	},
}

var oauthClientSecrets = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: "oauth-client-secrets",
	},
	Data: map[string][]byte{
		"3scale": bytes.NewBufferString("test").Bytes(),
	},
}

var installation = &v1alpha1.Installation{
	ObjectMeta: metav1.ObjectMeta{
		Name:       "test-installation",
		Namespace:  "integreatly-operator-namespace",
		Finalizers: []string{"finalizer.3scale.integreatly.org"},
	},
	TypeMeta: metav1.TypeMeta{
		APIVersion: v1alpha1.SchemeGroupVersion.String(),
	},
	Spec: v1alpha1.InstallationSpec{
		MasterURL:        "https://console.apps.example.com",
		RoutingSubdomain: "apps.example.com",
	},
}

var smtpCred = &crov1.SMTPCredentialSet{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-smtp-test-installation",
		Namespace: "integreatly-operator-namespace",
	},
	Status: crov1.SMTPCredentialSetStatus{
		Message:  "reconcile complete",
		Phase:    types.PhaseComplete,
		Provider: "openshift-smtp",
		SecretRef: &types.SecretRef{
			Name:      "test-smtp",
			Namespace: "integreatly-operator-namespace",
		},
		Strategy: "openshift",
	},
}

var smtpSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-smtp",
		Namespace: "integreatly-operator-namespace",
	},
	Data: map[string][]byte{
		"host":     []byte("test"),
		"password": []byte("test"),
		"port":     []byte("test"),
		"tls":      []byte("test"),
		"username": []byte("test"),
	},
}

var blobStorage = &crov1.BlobStorage{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-blobstorage-test-installation",
		Namespace: "integreatly-operator-namespace",
	},
	Status: crov1.BlobStorageStatus{
		Phase: types.PhaseComplete,
		SecretRef: &types.SecretRef{
			Name:      "threescale-blobstorage-test",
			Namespace: "integreatly-operator-namespace",
		},
	},
	Spec: crov1.BlobStorageSpec{
		SecretRef: &types.SecretRef{
			Name:      "threescale-blobstorage-test",
			Namespace: "integreatly-operator-namespace",
		},
	},
}

var blobStorageSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-blobstorage-test",
		Namespace: "integreatly-operator-namespace",
	},
	Data: map[string][]byte{
		"bucketName":          []byte("test"),
		"bucketRegion":        []byte("test"),
		"credentialKeyID":     []byte("test"),
		"credentialSecretKey": []byte("test"),
	},
}

var threescaleRoute1 = &v1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale-system-master-route",
		Namespace: "3scale",
		Labels: map[string]string{
			"zync.3scale.net/route-to": "system-master",
		},
	},
	Spec: v1.RouteSpec{
		Host: "system-master",
	},
}

var threescaleRoute2 = &v1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale-system-developer-route",
		Namespace: "3scale",
		Labels: map[string]string{
			"zync.3scale.net/route-to": "system-developer",
		},
	},
	Spec: v1.RouteSpec{
		Host: "system-developer",
	},
}

func getSuccessfullTestPreReqs(integreatlyOperatorNamespace, threeScaleInstallationNamepsace string) []runtime.Object {
	configManagerConfigMap.Namespace = integreatlyOperatorNamespace
	s3BucketSecret.Namespace = integreatlyOperatorNamespace
	s3CredentialsSecret.Namespace = integreatlyOperatorNamespace
	threeScaleAdminDetailsSecret.Namespace = threeScaleInstallationNamepsace
	threeScaleServiceDiscoveryConfigMap.Namespace = threeScaleInstallationNamepsace
	systemEnvConfigMap.Namespace = threeScaleInstallationNamepsace
	oauthClientSecrets.Namespace = integreatlyOperatorNamespace
	installation.Namespace = integreatlyOperatorNamespace

	return []runtime.Object{
		s3BucketSecret,
		s3CredentialsSecret,
		configManagerConfigMap,
		threeScaleAdminDetailsSecret,
		threeScaleServiceDiscoveryConfigMap,
		systemEnvConfigMap,
		testDedicatedAdminsGroup,
		OpenshiftDockerSecret,
		oauthClientSecrets,
		installation,
		smtpSec,
		smtpCred,
		blobStorage,
		blobStorageSec,
		threescaleRoute1,
		threescaleRoute2,
	}
}
