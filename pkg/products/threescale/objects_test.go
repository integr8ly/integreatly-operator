package threescale

import (
	"bytes"
	"fmt"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	"github.com/RHsyseng/operator-utils/pkg/olm"
	crov1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	crotypes "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	v1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v12 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

var subscription3scale = &operatorsv1alpha1.Subscription{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rhmi-3scale",
		Namespace: "3scale",
	},
	Status: operatorsv1alpha1.SubscriptionStatus{
		InstalledCSV: "rhmi-3scale",
		Install: &operatorsv1alpha1.InstallPlanReference{
			Name: "installplan-for-3scale",
		},
	},
}

var installPlanFor3ScaleSubscription = &operatorsv1alpha1.InstallPlan{
	ObjectMeta: metav1.ObjectMeta{
		Name: "installplan-for-3scale",
	},
	Status: operatorsv1alpha1.InstallPlanStatus{
		Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
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

var systemApp = appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-app",
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  "SUPPORT_EMAIL",
								Value: "wrong@example.com",
							},
						},
					},
				},
			},
		},
	},
	Status: appsv1.DeploymentStatus{
		ObservedGeneration: 1,
	},
}

var systemSidekiq = appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "system-sidekiq",
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  "SUPPORT_EMAIL",
								Value: "wrong@example.com",
							},
						},
					},
				},
			},
		},
	},
	Status: appsv1.DeploymentStatus{
		ObservedGeneration: 1,
	},
}

var successfulTestAppsV1Objects = map[string]*appsv1.Deployment{
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
	Status: crotypes.ResourceTypeStatus{
		Phase: crotypes.PhaseComplete,
		SecretRef: &crotypes.SecretRef{
			Name:      "threescale-blobstorage-test",
			Namespace: "integreatly-operator-ns",
		},
	},
	Spec: types.ResourceTypeSpec{
		SecretRef: &crotypes.SecretRef{
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
	Status: v1.RouteStatus{
		Ingress: []v1.RouteIngress{
			{
				Host: "127.0.0.1:10620/system-master",
			},
		},
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
	Status: v1.RouteStatus{
		Ingress: []v1.RouteIngress{
			{
				Host: "127.0.0.1:10620/system-developer",
			},
		},
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
	Status: v1.RouteStatus{
		Ingress: []v1.RouteIngress{
			{
				Host: "127.0.0.1:10620/system-provider",
			},
		},
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
	Status: crotypes.ResourceTypeStatus{
		Message:  "reconcile complete",
		Phase:    crotypes.PhaseComplete,
		Provider: "openshift-postgres",
		SecretRef: &crotypes.SecretRef{
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
	Status: crotypes.ResourceTypeStatus{
		Message:  "reconcile complete",
		Phase:    crotypes.PhaseComplete,
		Provider: "openshift-redis",
		SecretRef: &crotypes.SecretRef{
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
	Status: crotypes.ResourceTypeStatus{
		Message: "reconcile complete",
		Phase:   crotypes.PhaseComplete,
		SecretRef: &crotypes.SecretRef{
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

var apicastProduction = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "apicast-production",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "apicast-production",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var apicastStaging = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "apicast-staging",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "apicast-staging",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var backendCron = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "backend-cron",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "backend-cron",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var backendListener = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "backend-listener",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "backend-listener",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var backendWorker = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "backend-worker",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "backend-worker",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var systemAppDep = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-app",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-app",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var systemMemcache = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-memcache",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-memcache",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var systemSidekiqDep = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-sidekiq",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-sidekiq",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var systemSearchd = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "system-searchd",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "system-searchd",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var zync = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zync",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "zync",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var zyncDatabase = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zync-database",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "zync-database",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
}

var zyncQue = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zync-que",
		Namespace: nsPrefix + defaultInstallationNamespace,
		Labels: map[string]string{
			"app": "zync-que",
		},
	},
	Spec: appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
	},
	Status: appsv1.DeploymentStatus{},
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

var ingressRouterService = &corev1.Service{
	TypeMeta: metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "router-default",
		Namespace: "openshift-ingress",
	},
	Spec: corev1.ServiceSpec{},
	Status: corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					Hostname: "xxx.eu-west-1.elb.amazonaws.com",
				},
			},
		},
	},
}
var rhssoPostgres = &crov1.Postgres{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, "test-installation"),
		Namespace: nsPrefix + defaultInstallationNamespace,
	},
	Status: crotypes.ResourceTypeStatus{Phase: crotypes.PhaseComplete},
}

var cloudCredential = &cloudcredentialv1.CloudCredential{
	ObjectMeta: metav1.ObjectMeta{
		Name: sts.ClusterCloudCredentialName,
	},
	Spec: cloudcredentialv1.CloudCredentialSpec{
		CredentialsMode: cloudcredentialv1.CloudCredentialsModeDefault,
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
	systemSearchd.Namespace = threeScaleInstallationNamespace
	zync.Namespace = threeScaleInstallationNamespace
	zyncDatabase.Namespace = threeScaleInstallationNamespace
	zyncQue.Namespace = threeScaleInstallationNamespace
	threescale.Namespace = threeScaleInstallationNamespace
	rhssoPostgres.Namespace = integreatlyOperatorNamespace

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
		installPlanFor3ScaleSubscription,
		subscription3scale,
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
		systemSearchd,
		zync,
		zyncDatabase,
		zyncQue,
		threescale,
		clusterVersion,
		rhssoPostgres,
		ingressRouterService,
		smtpSec,
		cloudCredential,
	}
}
func getValidInstallation(installationType integreatlyv1alpha1.InstallationType) *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-installation",
			Namespace:       integreatlyOperatorNamespace,
			Finalizers:      []string{"finalizer.3scale.integreatly.org"},
			Generation:      1,
			ResourceVersion: "1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			MasterURL:        "https://console.apps.example.com",
			RoutingSubdomain: "apps.example.com",
			SMTPSecret:       "test-smtp",
			Type:             string(installationType),
		},
	}
}

func getTestInstallation(installType string) *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: "test",
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			Type: installType,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
	}
}

func getSuccessfullRHOAMTestPreReqs(integreatlyOperatorNamespace, threeScaleInstallationNamespace string) []runtime.Object {
	return append(getSuccessfullTestPreReqs(integreatlyOperatorNamespace, threeScaleInstallationNamespace),
		&v1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backend",
				Namespace: "3scale",
				Labels: map[string]string{
					"threescale_component": "backend",
				},
			},
			Spec: v1.RouteSpec{
				Host: "backend-3scale.apps",
			},
		},
		&v1.RouteList{
			Items: []v1.Route{
				{ObjectMeta: metav1.ObjectMeta{Name: "master", Namespace: "3scale"},
					Spec: v1.RouteSpec{Host: "master.apps.example.com"}},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apicast-staging",
				Namespace: "3scale",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apicast-production",
				Namespace: "3scale",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ratelimit",
				Namespace: "marin3r",
				UID:       "1",
			},
		},
		&usersv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"tenant": "true",
				},
				Name: "test_user",
			},
		},
		&corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "system-app-1-abcde",
						Namespace: "3scale",
						Labels: map[string]string{
							"deployment": "system-app",
						},
					},
					Status: corev1.PodStatus{
						Phase: "Running",
					},
				},
			},
		},
	)
}

func getBasicConfigMoc() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadThreeScaleFunc: func() (*config.ThreeScale, error) {
			return config.NewThreeScale(config.ProductConfig{
				"NAMESPACE": "3scale",
				"HOST":      "threescale.openshift-cluster.com",
			}), nil
		},
		GetOperatorNamespaceFunc: func() string {
			return integreatlyOperatorNamespace
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": testRhssoNamespace,
				"REALM":     "openshift",
			}), nil
		},
		GetOauthClientsSecretNameFunc: func() string {
			return "oauth-client-secrets"
		},
		ReadMarin3rFunc: func() (*config.Marin3r, error) {
			return &config.Marin3r{
				Config: config.ProductConfig{
					"NAMESPACE": "marin3r",
				},
			}, nil
		},
	}
}

func getTestBlobStorage() *crov1.BlobStorage {
	return &crov1.BlobStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "threescale-blobstorage-rhmi",
			Namespace: "test",
		},
		Status: crotypes.ResourceTypeStatus{
			Phase: crotypes.PhaseComplete,
			SecretRef: &crotypes.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
		Spec: crotypes.ResourceTypeSpec{
			SecretRef: &crotypes.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
	}
}
