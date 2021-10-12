package threescale

import (
	"bytes"
	"fmt"
	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/RHsyseng/operator-utils/pkg/olm"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	appsv1 "github.com/openshift/api/apps/v1"
	v1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"

	v12 "github.com/openshift/api/config/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	testRhssoNamespace = "test-rhsso"
	testRhssoRealm     = "test-realm"
	testRhssoURL       = "https://test.rhsso.url"
	nsPrefix           = "testing-namespaces-"
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
		Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
		Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
	},
}

var ComponentDockerSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
		Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
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
		Name: "dummy-bucket",
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
	Data: map[string][]byte{},
}

var threeScaleApiCastSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-master-apicast",
	},
	Data: map[string][]byte{},
}
var threeScaleServiceDiscoveryConfigMap = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system",
	},
	Data: map[string]string{
		"service_discovery.yml": "",
	},
}

var rhssoTest1 = &keycloak.KeycloakUser{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-user1",
		Namespace: testRhssoNamespace,
		Labels: map[string]string{
			rhsso.SSOLabelKey: rhsso.SSOLabelValue,
		},
	},
	Spec: keycloak.KeycloakUserSpec{
		User: keycloak.KeycloakAPIUser{
			UserName: "test1",
			Email:    "test1@example.com",
		},
	},
}

var rhssoTest2 = &keycloak.KeycloakUser{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-user2",
		Namespace: testRhssoNamespace,
		Labels: map[string]string{
			rhsso.SSOLabelKey: rhsso.SSOLabelValue,
		},
	},
	Spec: keycloak.KeycloakUserSpec{
		User: keycloak.KeycloakAPIUser{
			UserName: "test2",
			Email:    "test2@example.com",
		},
	},
}

var rhssoTest3 = &keycloak.KeycloakUser{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "alongusernamethatisabovefourtycharacterslong",
		Namespace: testRhssoNamespace,
		Labels: map[string]string{
			rhsso.SSOLabelKey: rhsso.SSOLabelValue,
		},
	},
	Spec: keycloak.KeycloakUserSpec{
		User: keycloak.KeycloakAPIUser{
			UserName: "alongusernamethatisabovefourtycharacterslong",
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
	systemApp.Name:         &systemApp,
	systemSidekiq.Name:     &systemSidekiq,
	apicastProduction.Name: apicastProduction,
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

var installation = &integreatlyv1alpha1.RHMI{
	ObjectMeta: metav1.ObjectMeta{
		Name:       "test-installation",
		Namespace:  "integreatly-operator-ns",
		Finalizers: []string{"finalizer.3scale.integreatly.org"},
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "RHMI",
		APIVersion: integreatlyv1alpha1.GroupVersion.String(),
	},
	Spec: integreatlyv1alpha1.RHMISpec{
		MasterURL:        "https://console.apps.example.com",
		RoutingSubdomain: "apps.example.com",
	},
}

var smtpSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-smtp",
		Namespace: "integreatly-operator-ns",
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
		Namespace: "integreatly-operator-ns",
	},
	Status: types.ResourceTypeStatus{
		Phase: types.PhaseComplete,
		SecretRef: &types.SecretRef{
			Name:      "threescale-blobstorage-test",
			Namespace: "integreatly-operator-ns",
		},
	},
	Spec: types.ResourceTypeSpec{
		SecretRef: &types.SecretRef{
			Name:      "threescale-blobstorage-test",
			Namespace: "integreatly-operator-ns",
		},
	},
}

var blobStorageSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-blobstorage-test",
		Namespace: "integreatly-operator-ns",
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

// Have two system-developer routes, the reconcile should pick up on 3scale.
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

var threescaleRoute3 = &v1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale-system-developer-route-2",
		Namespace: "3scale",
		Labels: map[string]string{
			"zync.3scale.net/route-to": "system-developer",
		},
	},
	Spec: v1.RouteSpec{
		Host: "3scale.system-developer",
	},
}

// Have two system-provider routes, the reconcile should pick up on 3scale-admin.
var threescaleRoute4 = &v1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale-system-provider-route",
		Namespace: "3scale",
		Labels: map[string]string{
			"zync.3scale.net/route-to": "system-provider",
		},
	},
	Spec: v1.RouteSpec{
		Host: "system-provider",
	},
}

var threescaleRoute5 = &v1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale-system-provider-route-2",
		Namespace: "3scale",
		Labels: map[string]string{
			"zync.3scale.net/route-to": "system-provider",
		},
	},
	Spec: v1.RouteSpec{
		Host: "3scale-admin.system-provider",
	},
}

var threescaleRoute6 = &v1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale-system-provider-route-6",
		Namespace: "3scale",
		Labels: map[string]string{
			"zync.3scale.net/route-to": "backend",
		},
	},
	Spec: v1.RouteSpec{
		Host: "3scale-admin.backend",
	},
}

var postgres = &crov1.Postgres{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-postgres-test-installation",
		Namespace: "integreatly-operator-ns",
	},
	Status: types.ResourceTypeStatus{
		Message:  "reconcile complete",
		Phase:    types.PhaseComplete,
		Provider: "openshift-postgres",
		SecretRef: &types.SecretRef{
			Name:      "test-postgres",
			Namespace: "integreatly-operator-ns",
		},
		Strategy: "openshift",
	},
}

var postgresSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-postgres",
		Namespace: "integreatly-operator-ns",
	},
	Data: map[string][]byte{
		"host":     []byte("test"),
		"password": []byte("test"),
		"port":     []byte("test"),
		"tls":      []byte("test"),
		"username": []byte("test"),
	},
}

var redis = &crov1.Redis{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-redis-test-installation",
		Namespace: "integreatly-operator-ns",
	},
	Status: types.ResourceTypeStatus{
		Message:  "reconcile complete",
		Phase:    types.PhaseComplete,
		Provider: "openshift-redis",
		SecretRef: &types.SecretRef{
			Name:      "test-redis",
			Namespace: "integreatly-operator-ns",
		},
		Strategy: "openshift",
	},
}

var redisSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-redis",
		Namespace: "integreatly-operator-ns",
	},
	Data: map[string][]byte{
		"uri":  []byte("test"),
		"port": []byte("test"),
	},
}

var backendRedis = &crov1.Redis{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "threescale-backend-redis-test-installation",
		Namespace: "integreatly-operator-ns",
	},
	Status: types.ResourceTypeStatus{
		Message: "reconcile complete",
		Phase:   types.PhaseComplete,
		SecretRef: &types.SecretRef{
			Name:      "test-backend-redis",
			Namespace: "integreatly-operator-ns",
		},
		Strategy: "openshift",
	},
}

var backendRedisSec = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-backend-redis",
		Namespace: "integreatly-operator-ns",
	},
	Data: map[string][]byte{
		"uri":  []byte("test"),
		"port": []byte("test"),
	},
}

var ns = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: defaultInstallationNamespace,
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
		Name: defaultInstallationNamespace + "-operator",
		Labels: map[string]string{
			resources.OwnerLabelKey: string(installation.GetUID()),
		},
	},
	Status: corev1.NamespaceStatus{
		Phase: corev1.NamespaceActive,
	},
}

var apicastProduction = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "apicast-production",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "apicast-production",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var apicastStaging = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "apicast-staging",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "apicast-staging",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var backendCron = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "backend-cron",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "backend-cron",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var backendListener = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "backend-listener",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "backend-listener",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var backendWorker = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "backend-worker",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "backend-worker",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var systemAppDep = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-app",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-app",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var systemMemcache = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-memcache",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-memcache",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var systemSidekiqDep = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-sidekiq",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-sidekiq",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var systemSphinx = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-sphinx",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-sphinx",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var zync = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zync",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "zync",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var zyncDatabase = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zync-database",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "zync-database",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var zyncQue = &appsv1.DeploymentConfig{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zync-que",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "zync-que",
		},
	},
	Spec: appsv1.DeploymentConfigSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentConfigStatus{},
}

var threescale = &threescalev1.APIManager{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "3scale",
		Namespace: nsPrefix + defaultInstallationNamespace,
	},
	Spec: threescalev1.APIManagerSpec{},
	Status: threescalev1.APIManagerStatus{
		Deployments: olm.DeploymentStatus{
			Ready:    []string{"Ready status is when there is at least one ready and none starting or stopped"},
			Starting: []string{},
			Stopped:  []string{},
		},
	},
}

var clusterVersion = &v12.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: v12.ClusterVersionStatus{
		History: []v12.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "4.9.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

func getSuccessfullTestPreReqs(integreatlyOperatorNamespace, threeScaleInstallationNamespace string) []runtime.Object {
	configManagerConfigMap.Namespace = integreatlyOperatorNamespace
	s3BucketSecret.Namespace = integreatlyOperatorNamespace
	s3CredentialsSecret.Namespace = integreatlyOperatorNamespace
	threeScaleAdminDetailsSecret.Namespace = threeScaleInstallationNamespace
	threeScaleApiCastSecret.Namespace = threeScaleInstallationNamespace
	threeScaleServiceDiscoveryConfigMap.Namespace = threeScaleInstallationNamespace
	systemEnvConfigMap.Namespace = threeScaleInstallationNamespace
	oauthClientSecrets.Namespace = integreatlyOperatorNamespace
	installation.Namespace = integreatlyOperatorNamespace
	apicastProduction.Namespace = threeScaleInstallationNamespace
	apicastStaging.Namespace = threeScaleInstallationNamespace
	backendCron.Namespace = threeScaleInstallationNamespace
	backendListener.Namespace = threeScaleInstallationNamespace
	backendWorker.Namespace = threeScaleInstallationNamespace
	systemApp.Namespace = threeScaleInstallationNamespace
	systemAppDep.Namespace = threeScaleInstallationNamespace
	systemMemcache.Namespace = threeScaleInstallationNamespace
	systemSidekiq.Namespace = threeScaleInstallationNamespace
	systemSidekiqDep.Namespace = threeScaleInstallationNamespace
	systemSphinx.Namespace = threeScaleInstallationNamespace
	zync.Namespace = threeScaleInstallationNamespace
	zyncDatabase.Namespace = threeScaleInstallationNamespace
	zyncQue.Namespace = threeScaleInstallationNamespace
	threescale.Namespace = threeScaleInstallationNamespace

	return []runtime.Object{
		s3BucketSecret,
		s3CredentialsSecret,
		configManagerConfigMap,
		threeScaleAdminDetailsSecret,
		threeScaleApiCastSecret,
		threeScaleServiceDiscoveryConfigMap,
		systemEnvConfigMap,
		testDedicatedAdminsGroup,
		OpenshiftDockerSecret,
		oauthClientSecrets,
		installation,
		blobStorage,
		blobStorageSec,
		threescaleRoute1,
		threescaleRoute2,
		threescaleRoute3,
		threescaleRoute4,
		threescaleRoute5,
		threescaleRoute6,
		postgres,
		postgresSec,
		redis,
		redisSec,
		backendRedis,
		backendRedisSec,
		rhssoTest2,
		rhssoTest1,
		rhssoTest3,
		ns,
		operatorNS,
		apicastProduction,
		apicastStaging,
		backendCron,
		backendListener,
		backendWorker,
		systemAppDep,
		systemMemcache,
		systemSidekiqDep,
		systemSphinx,
		zync,
		zyncDatabase,
		zyncQue,
		threescale,
		clusterVersion,
	}
}
