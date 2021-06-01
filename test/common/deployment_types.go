package common

import (
	goctx "context"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"golang.org/x/net/context"
	k8sappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	appsv1 "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	rhmi2DeploymentsList = []string{
		"aMQOnlineOperatorDeployment",
		"codeReadyOperatorDeployment",
		"codereadyWorkspacesDeployment",
		"fuseOperatorDeployment",
		"solutionExplorerOperatorDeployment",
		"upsOperatorDeployment",
		"upsDeployment",
	}
	commonApiDeploymentsList = []string{
		"threeScaleDeployment",
		"cloudResourceOperatorDeployment",
		"monitoringOperatorDeployment",
		"rhssoOperatorDeployment",
		"rhssoUserOperatorDeployment",
	}
	managedApiDeploymentsList = []string{
		"marin3rOperatorDeployment",
		"marin3rDeployment",
	}
)

func getDeploymentConfiguration(deploymentName string, inst *integreatlyv1alpha1.RHMI, t TestingTB, ctx *TestingContext) Namespace {
	threescaleConfig := config.NewThreeScale(map[string]string{})
	replicas := threescaleConfig.GetReplicasConfig(inst)
	deployment := map[string]Namespace{
		"threeScaleDeployment": {
			Name: ThreeScaleOperatorNamespace,
			Products: []Product{
				{Name: "3scale-operator", ExpectedReplicas: 1},
			},
		},
		"aMQOnlineOperatorDeployment": {
			Name: AMQOnlineOperatorNamespace,
			Products: []Product{
				{Name: "address-space-controller", ExpectedReplicas: 1},
				{Name: "console", ExpectedReplicas: 1},
				{Name: "enmasse-operator", ExpectedReplicas: 1},
				{Name: "none-authservice", ExpectedReplicas: 1},
				{Name: "standard-authservice", ExpectedReplicas: 1},
			},
		},
		"cloudResourceOperatorDeployment": {
			Name: CloudResourceOperatorNamespace,
			Products: []Product{
				{Name: "cloud-resource-operator", ExpectedReplicas: 1},
			},
		},
		"codeReadyOperatorDeployment": {
			Name: CodeReadyOperatorNamespace,
			Products: []Product{
				{Name: "codeready-operator", ExpectedReplicas: 1},
			},
		},
		"codereadyWorkspacesDeployment": {
			Name: NamespacePrefix + "codeready-workspaces",
			Products: []Product{
				{Name: "codeready", ExpectedReplicas: 1},
				{Name: "devfile-registry", ExpectedReplicas: 1},
				{Name: "plugin-registry", ExpectedReplicas: 1},
			},
		},
		"fuseOperatorDeployment": {
			Name: FuseOperatorNamespace,
			Products: []Product{
				{Name: "syndesis-operator", ExpectedReplicas: 1},
			},
		},
		"monitoringOperatorDeployment": {
			Name: MonitoringOperatorNamespace,
			Products: []Product{
				{Name: "application-monitoring-operator", ExpectedReplicas: 1},
				{Name: "grafana-deployment", ExpectedReplicas: 1},
				{Name: "grafana-operator", ExpectedReplicas: 1},
				{Name: "prometheus-operator", ExpectedReplicas: 1},
			},
		},
		"rhmiOperatorDeploymentForRhmi2": {
			Name: RHMIOperatorNamespace,
			Products: []Product{
				{Name: "standard-authservice-postgresql", ExpectedReplicas: 1},
			},
		},
		"rhmiOperatorDeploymentForManagedApi": {
			Name:     RHMIOperatorNamespace,
			Products: []Product{},
		},
		"rhssoOperatorDeployment": {
			Name: RHSSOOperatorNamespace,
			Products: []Product{
				{Name: "keycloak-operator", ExpectedReplicas: 1},
			},
		},
		"solutionExplorerOperatorDeployment": {
			Name: SolutionExplorerOperatorNamespace,
			Products: []Product{
				{Name: "tutorial-web-app-operator", ExpectedReplicas: 1},
			},
		},
		"upsOperatorDeployment": {
			Name: UPSOperatorNamespace,
			Products: []Product{
				{Name: "unifiedpush-operator", ExpectedReplicas: 1},
			},
		},
		"upsDeployment": {
			Name: NamespacePrefix + "ups",
			Products: []Product{
				{Name: "ups", ExpectedReplicas: 1},
			},
		},
		"rhssoUserOperatorDeployment": {
			Name: RHSSOUserOperatorNamespace,
			Products: []Product{
				{Name: "keycloak-operator", ExpectedReplicas: 1},
			},
		},
		"marin3rOperatorDeployment": {
			Name: Marin3rOperatorNamespace,
			Products: []Product{
				{Name: "marin3r-controller-manager", ExpectedReplicas: 1},
				{Name: "marin3r-controller-webhook", ExpectedReplicas: 2},
			},
		},
		"threeScaleDeploymentConfig": {
			Name: NamespacePrefix + "3scale",
			Products: []Product{
				{Name: "apicast-production", ExpectedReplicas: int32(replicas["apicastProd"])},
				{Name: "apicast-staging", ExpectedReplicas: int32(replicas["apicastStage"])},
				{Name: "backend-cron", ExpectedReplicas: int32(replicas["backendCron"])},
				{Name: "backend-listener", ExpectedReplicas: int32(replicas["backendListener"])},
				{Name: "backend-worker", ExpectedReplicas: int32(replicas["backendWorker"])},
				{Name: "system-app", ExpectedReplicas: int32(replicas["systemApp"])},
				{Name: "system-memcache", ExpectedReplicas: 1},
				{Name: "system-sidekiq", ExpectedReplicas: int32(replicas["systemSidekiq"])},
				{Name: "system-sphinx", ExpectedReplicas: 1},
				{Name: "zync", ExpectedReplicas: 1},
				{Name: "zync-database", ExpectedReplicas: int32(replicas["zyncDatabase"])},
				{Name: "zync-que", ExpectedReplicas: int32(replicas["zyncQue"])},
			},
		},
		"fuseDeploymentConfig": {
			Name: NamespacePrefix + "fuse",
			Products: []Product{
				{Name: "syndesis-meta", ExpectedReplicas: 1},
				{Name: "syndesis-oauthproxy", ExpectedReplicas: 1},
				{Name: "syndesis-prometheus", ExpectedReplicas: 1},
				{Name: "syndesis-server", ExpectedReplicas: 1},
				{Name: "syndesis-ui", ExpectedReplicas: 1},
				{Name: "broker-amq", ExpectedReplicas: 1},
			},
		},
		"solutionExplorerDeploymentConfig": {
			Name: NamespacePrefix + "solution-explorer",
			Products: []Product{
				{Name: "tutorial-web-app", ExpectedReplicas: 1},
			},
		},
	}

	if inst.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		ratelimitCR := &k8sappsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quota.RateLimitName,
				Namespace: Marin3rProductNamespace,
			},
		}

		key, err := k8sclient.ObjectKeyFromObject(ratelimitCR)
		if err != nil {
			t.Fatalf("Error getting key from ratelimit Deployment: %v", err)
		}

		err = ctx.Client.Get(context.TODO(), key, ratelimitCR)
		if err != nil {
			if !k8sError.IsNotFound(err) {
				t.Fatalf("Error obtaining ratelimit CR: %v", err)
			}
		}
		deployment["marin3rDeployment"] = Namespace{
			Name: Marin3rProductNamespace,
			Products: []Product{
				{Name: "prom-statsd-exporter", ExpectedReplicas: 1},
				{Name: "ratelimit", ExpectedReplicas: *ratelimitCR.Spec.Replicas},
			},
		}
	}

	return deployment[deploymentName]
}

func getClusterStorageDeployments(installationName string, installType string) []Namespace {

	rhmi2ClusterStorageDeployments := []Namespace{
		{
			Name: NamespacePrefix + "operator",
			Products: []Product{
				{Name: constants.CodeReadyPostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScaleBackendRedisPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScalePostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScaleSystemRedisPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.UPSPostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RHSSOPostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RHSSOUserProstgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.AMQAuthServicePostgres, ExpectedReplicas: 1},
			},
		},
	}
	managedApiClusterStorageDeployments := []Namespace{
		{
			Name: NamespacePrefix + "operator",
			Products: []Product{
				{Name: constants.ThreeScaleBackendRedisPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScalePostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScaleSystemRedisPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RHSSOPostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RHSSOUserProstgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RateLimitRedisPrefix + installationName, ExpectedReplicas: 1},
			},
		},
	}

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return managedApiClusterStorageDeployments
	} else {
		return rhmi2ClusterStorageDeployments
	}
}

func TestDeploymentExpectedReplicas(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	deployments := getDeployments(rhmi, t, ctx)
	clusterStorageDeployments := getClusterStorageDeployments(rhmi.Name, rhmi.Spec.Type)

	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	// If the cluster is using in cluster storage instead of AWS resources
	// These deployments will also need to be checked
	if isClusterStorage {
		for _, d := range clusterStorageDeployments {
			deployments = append(deployments, d)
		}
	}

	for _, namespace := range deployments {
		for _, product := range namespace.Products {

			deployment, err := ctx.KubeClient.AppsV1().Deployments(namespace.Name).Get(goctx.TODO(), product.Name, metav1.GetOptions{})
			if err != nil {
				// Fail the test without failing immideatlly
				t.Errorf("Failed to get Deployment %s in namespace %s with error: %s", product.Name, namespace.Name, err)
				continue
			}

			if deployment.Status.Replicas < product.ExpectedReplicas {
				t.Errorf("Deployment %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deployment.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}

			if rhmi.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) && product.Name == "ratelimit" {
				pods := &corev1.PodList{}
				err = ctx.Client.List(context.TODO(), pods, GetListOptions(Marin3rProductNamespace, "app=ratelimit")...)
				if err != nil {
					t.Fatalf("failed to get pods for Ratelimit: %v", err)
				}
				checkDeploymentPods(t, pods, product, namespace, deployment)

			}
			// Verify that the expected replicas are also available, means they are up and running and consumable by users
			if deployment.Status.AvailableReplicas < product.ExpectedReplicas {
				t.Errorf("Deployment %s in namespace %s doesn't match the number of expected available replicas. Available Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deployment.Status.AvailableReplicas,
					product.ExpectedReplicas,
				)
				continue

			}
		}
	}
}

func checkDeploymentPods(t TestingTB, pods *corev1.PodList, product Product, namespace Namespace, deployment *k8sappsv1.Deployment) {
	if int32(len(pods.Items)) < product.ExpectedReplicas {
		t.Errorf("Deployment %s in namespace %s doesn't match the number of expected available replicas. Available Replicas: %v / Expected Replicas: %v",
			product.Name,
			namespace.Name,
			deployment.Status.AvailableReplicas,
			product.ExpectedReplicas,
		)
	}
}

func checkDeploymentConfigPods(t TestingTB, pods *corev1.PodList, product Product, namespace Namespace, deploymentConfig *appsv1.DeploymentConfig) {
	if int32(len(pods.Items)) < product.ExpectedReplicas {
		t.Errorf("DeploymentConfig %s in namespace %s doesn't match the number of expected available replicas. Available Replicas: %v / Expected Replicas: %v",
			product.Name,
			namespace.Name,
			deploymentConfig.Status.AvailableReplicas,
			product.ExpectedReplicas,
		)
	}
}

func getDeployments(inst *integreatlyv1alpha1.RHMI, t TestingTB, ctx *TestingContext) []Namespace {
	var rhmi2Deployments []Namespace
	var commonApiDeployments []Namespace
	var managedApiDeployments []Namespace

	for _, deployment := range rhmi2DeploymentsList {
		rhmi2Deployments = append(rhmi2Deployments, getDeploymentConfiguration(deployment, inst, t, ctx))
	}
	for _, deployment := range commonApiDeploymentsList {
		commonApiDeployments = append(commonApiDeployments, getDeploymentConfiguration(deployment, inst, t, ctx))
	}
	for _, deployment := range managedApiDeploymentsList {
		managedApiDeployments = append(managedApiDeployments, getDeploymentConfiguration(deployment, inst, t, ctx))
	}

	if inst.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return append(append(commonApiDeployments, []Namespace{getDeploymentConfiguration("rhmiOperatorDeploymentForManagedApi", inst, t, ctx)}...), managedApiDeployments...)
	} else {
		return append(append(commonApiDeployments, rhmi2Deployments...), []Namespace{getDeploymentConfiguration("rhmiOperatorDeploymentForRhmi2", inst, t, ctx)}...)
	}
}

func TestDeploymentConfigExpectedReplicas(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	deploymentConfigs := getDeploymentConfigs(rhmi, t, ctx)

	for _, namespace := range deploymentConfigs {
		for _, product := range namespace.Products {

			deploymentConfig := &appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      product.Name,
					Namespace: namespace.Name,
				},
			}
			err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: product.Name, Namespace: namespace.Name}, deploymentConfig)
			if err != nil {
				t.Errorf("Failed to get DeploymentConfig %s in namespace %s with error: %s", product.Name, namespace.Name, err)
				continue
			}

			if deploymentConfig.Status.Replicas < product.ExpectedReplicas {
				t.Errorf("DeploymentConfig %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deploymentConfig.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}
			if rhmi.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
				if product.Name == "apicast-production" || product.Name == "backend-listener" || product.
					Name == "backend-worker" {
					pods := &corev1.PodList{}
					err = ctx.Client.List(context.TODO(), pods, GetListOptions(ThreeScaleProductNamespace,
						"deploymentconfig="+product.Name)...)
					if err != nil {
						t.Fatalf("failed to get %v pods for 3scale: %v", product.Name, err)
					}
					checkDeploymentConfigPods(t, pods, product, namespace, deploymentConfig)
				}
			} else if deploymentConfig.Status.AvailableReplicas < product.ExpectedReplicas {
				t.Errorf("DeploymentConfig %s in namespace %s doesn't match the number of expected available replicas. Available Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deploymentConfig.Status.AvailableReplicas,
					product.ExpectedReplicas,
				)
				continue
			}
		}
	}
}

func getDeploymentConfigs(inst *integreatlyv1alpha1.RHMI, t TestingTB, ctx *TestingContext) []Namespace {
	if inst.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return []Namespace{
			getDeploymentConfiguration("threeScaleDeploymentConfig", inst, t, ctx),
		}
	}
	return []Namespace{
		getDeploymentConfiguration("threeScaleDeploymentConfig", inst, t, ctx),
		getDeploymentConfiguration("fuseDeploymentConfig", inst, t, ctx),
		getDeploymentConfiguration("solutionExplorerDeploymentConfig", inst, t, ctx),
	}
}

func TestStatefulSetsExpectedReplicas(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	var rhssoExpectedReplicas int32 = 2
	var rhssoUserExpectedReplicas int32 = 2

	if rhmi.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		quotaConfig, _, err := getQuotaconfig(t, ctx.Client)
		if err != nil {
			t.Fatalf("Error retrieving Quota: %v", err)
		}

		rhssoUserExpectedReplicas = quotaConfig.GetProduct(integreatlyv1alpha1.ProductRHSSOUser).GetReplicas(
			quota.KeycloakName)

		keycloakCR := &v1alpha1.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quota.KeycloakName,
				Namespace: RHSSOUserProductNamespace,
			},
		}
		key, err := k8sclient.ObjectKeyFromObject(keycloakCR)
		if err != nil {
			t.Fatalf("Error getting Keycloak CR key: %v", err)
		}

		err = ctx.Client.Get(context.TODO(), key, keycloakCR)
		if err != nil {
			t.Fatalf("Error getting Keycloak CR: %v", err)
		}

		rhssoUserExpectedReplicas = int32(keycloakCR.Spec.Instances)
	}
	statefulSets := []Namespace{
		{
			Name: MonitoringOperatorNamespace,
			Products: []Product{
				{Name: "alertmanager-application-monitoring", ExpectedReplicas: 1},
				{Name: "prometheus-application-monitoring", ExpectedReplicas: 1},
			},
		},
		{
			Name: NamespacePrefix + "rhsso",
			Products: []Product{
				{Name: "keycloak", ExpectedReplicas: rhssoExpectedReplicas},
			},
		},
		{
			Name: NamespacePrefix + "user-sso",
			Products: []Product{
				{Name: "keycloak", ExpectedReplicas: rhssoUserExpectedReplicas},
			},
		},
	}

	for _, namespace := range statefulSets {
		for _, product := range namespace.Products {
			statefulSet, err := ctx.KubeClient.AppsV1().StatefulSets(namespace.Name).Get(goctx.TODO(), product.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("Failed to get StatefulSet %s in namespace %s with error: %s", product.Name, namespace.Name, err)
				continue
			}

			if statefulSet.Status.Replicas < product.ExpectedReplicas {
				t.Errorf("StatefulSet %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					statefulSet.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}

			if namespace.Name == RHSSOUserProductNamespace && product.Name == "keycloak" {
				pods := &corev1.PodList{}
				err = ctx.Client.List(context.TODO(), pods, GetListOptions(RHSSOUserProductNamespace, "component=keycloak")...)
				if err != nil {
					t.Fatalf("failed to get pods for Keycloak: %v", err)
				}

				if int32(len(pods.Items)) < product.ExpectedReplicas {
					t.Errorf("StatefulSet %s in namespace %s doesn't match the number of expected ready replicas. Ready Replicas: %v / Expected Replicas: %v",
						product.Name,
						namespace.Name,
						statefulSet.Status.ReadyReplicas,
						product.ExpectedReplicas,
					)
					continue
				}
			}
			// Verify the number of ReadyReplicas because the SatefulSet doesn't have the concept of AvailableReplicas
			if statefulSet.Status.ReadyReplicas < product.ExpectedReplicas {
				t.Errorf("StatefulSet %s in namespace %s doesn't match the number of expected ready replicas. Ready Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					statefulSet.Status.ReadyReplicas,
					product.ExpectedReplicas,
				)
				continue
			}
		}
	}
}

func GetListOptions(namespace string, podLabels ...string) []k8sclient.ListOption {
	selector := labels.NewSelector()
	for _, label := range podLabels {
		selector, _ = labels.Parse(label)
	}
	return []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
}
