package common

import (
	goctx "context"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	appsv1 "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func getDeploymentConfiguration(deploymentName string) Namespace {
	deployment := map[string]Namespace{
		"threeScaleDeployment": Namespace{
			Name: ThreeScaleOperatorNamespace,
			Products: []Product{
				{Name: "3scale-operator", ExpectedReplicas: 1},
			},
		},
		"aMQOnlineOperatorDeployment": Namespace{
			Name: AMQOnlineOperatorNamespace,
			Products: []Product{
				{Name: "address-space-controller", ExpectedReplicas: 1},
				{Name: "console", ExpectedReplicas: 1},
				{Name: "enmasse-operator", ExpectedReplicas: 1},
				{Name: "none-authservice", ExpectedReplicas: 1},
				{Name: "standard-authservice", ExpectedReplicas: 1},
			},
		},
		"cloudResourceOperatorDeployment": Namespace{
			Name: CloudResourceOperatorNamespace,
			Products: []Product{
				{Name: "cloud-resource-operator", ExpectedReplicas: 1},
			},
		},
		"codeReadyOperatorDeployment": Namespace{
			Name: CodeReadyOperatorNamespace,
			Products: []Product{
				{Name: "codeready-operator", ExpectedReplicas: 1},
			},
		},
		"codereadyWorkspacesDeployment": Namespace{
			Name: NamespacePrefix + "codeready-workspaces",
			Products: []Product{
				{Name: "codeready", ExpectedReplicas: 1},
				{Name: "devfile-registry", ExpectedReplicas: 1},
				{Name: "plugin-registry", ExpectedReplicas: 1},
			},
		},
		"fuseOperatorDeployment": Namespace{
			Name: FuseOperatorNamespace,
			Products: []Product{
				{Name: "syndesis-operator", ExpectedReplicas: 1},
			},
		},
		"monitoringOperatorDeployment": Namespace{
			Name: MonitoringOperatorNamespace,
			Products: []Product{
				{Name: "application-monitoring-operator", ExpectedReplicas: 1},
				{Name: "grafana-deployment", ExpectedReplicas: 1},
				{Name: "grafana-operator", ExpectedReplicas: 1},
				{Name: "prometheus-operator", ExpectedReplicas: 1},
			},
		},
		"rhmiOperatorDeploymentForRhmi2": Namespace{
			Name: RHMIOperatorNamespace,
			Products: []Product{
				{Name: "standard-authservice-postgresql", ExpectedReplicas: 1},
			},
		},
		"rhmiOperatorDeploymentForManagedApi": Namespace{
			Name:     RHMIOperatorNamespace,
			Products: []Product{},
		},
		"rhssoOperatorDeployment": Namespace{
			Name: RHSSOOperatorNamespace,
			Products: []Product{
				{Name: "keycloak-operator", ExpectedReplicas: 1},
			},
		},
		"solutionExplorerOperatorDeployment": Namespace{
			Name: SolutionExplorerOperatorNamespace,
			Products: []Product{
				{Name: "tutorial-web-app-operator", ExpectedReplicas: 1},
			},
		},
		"upsOperatorDeployment": Namespace{
			Name: UPSOperatorNamespace,
			Products: []Product{
				{Name: "unifiedpush-operator", ExpectedReplicas: 1},
			},
		},
		"upsDeployment": Namespace{
			Name: NamespacePrefix + "ups",
			Products: []Product{
				{Name: "ups", ExpectedReplicas: 1},
			},
		},
		"rhssoUserOperatorDeployment": Namespace{
			Name: RHSSOUserOperatorNamespace,
			Products: []Product{
				{Name: "keycloak-operator", ExpectedReplicas: 1},
			},
		},
		"marin3rOperatorDeployment": Namespace{
			Name: Marin3rOperatorNamespace,
			Products: []Product{
				{Name: "marin3r-operator", ExpectedReplicas: 1},
			},
		},
		"marin3rDeployment": Namespace{
			Name: Marin3rProductNamespace,
			Products: []Product{
				{Name: "marin3r-instance", ExpectedReplicas: 1},
				{Name: "prom-statsd-exporter", ExpectedReplicas: 1},
				{Name: "ratelimit", ExpectedReplicas: 3},
			},
		},
		"threeScaleDeploymentConfig": Namespace{
			Name: NamespacePrefix + "3scale",
			Products: []Product{
				{Name: "apicast-production", ExpectedReplicas: 2},
				{Name: "apicast-staging", ExpectedReplicas: 2},
				{Name: "backend-cron", ExpectedReplicas: 2},
				{Name: "backend-listener", ExpectedReplicas: 2},
				{Name: "backend-worker", ExpectedReplicas: 2},
				{Name: "system-app", ExpectedReplicas: 2},
				{Name: "system-memcache", ExpectedReplicas: 1},
				{Name: "system-sidekiq", ExpectedReplicas: 2},
				{Name: "system-sphinx", ExpectedReplicas: 1},
				{Name: "zync", ExpectedReplicas: 2},
				{Name: "zync-database", ExpectedReplicas: 1},
				{Name: "zync-que", ExpectedReplicas: 2},
			},
		},
		"fuseDeploymentConfig": Namespace{
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
		"solutionExplorerDeploymentConfig": Namespace{
			Name: NamespacePrefix + "solution-explorer",
			Products: []Product{
				{Name: "tutorial-web-app", ExpectedReplicas: 1},
			},
		},
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

func TestDeploymentExpectedReplicas(t *testing.T, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	deployments := getDeployments(rhmi.Spec.Type)
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

			deployment, err := ctx.KubeClient.AppsV1().Deployments(namespace.Name).Get(product.Name, v1.GetOptions{})
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

func getDeployments(installType string) []Namespace {
	var rhmi2Deployments []Namespace
	var commonApiDeployments []Namespace
	var managedApiDeployments []Namespace

	for _, deployment := range rhmi2DeploymentsList {
		rhmi2Deployments = append(rhmi2Deployments, getDeploymentConfiguration(deployment))
	}
	for _, deployment := range commonApiDeploymentsList {
		commonApiDeployments = append(commonApiDeployments, getDeploymentConfiguration(deployment))
	}
	for _, deployment := range managedApiDeploymentsList {
		managedApiDeployments = append(managedApiDeployments, getDeploymentConfiguration(deployment))
	}

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return append(append(commonApiDeployments, []Namespace{getDeploymentConfiguration("rhmiOperatorDeploymentForManagedApi")}...), managedApiDeployments...)
	} else {
		return append(append(commonApiDeployments, rhmi2Deployments...), []Namespace{getDeploymentConfiguration("rhmiOperatorDeploymentForRhmi2")}...)
	}
}

func TestDeploymentConfigExpectedReplicas(t *testing.T, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	deploymentConfigs := getDeploymentConfigs(rhmi.Spec.Type)

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

			if deploymentConfig.Status.AvailableReplicas < product.ExpectedReplicas {
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

func getDeploymentConfigs(installType string) []Namespace {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return []Namespace{Namespace{
			Name: NamespacePrefix + "3scale",
			Products: []Product{
				Product{Name: "apicast-production", ExpectedReplicas: 2},
				Product{Name: "apicast-staging", ExpectedReplicas: 2},
				Product{Name: "backend-cron", ExpectedReplicas: 2},
				Product{Name: "backend-listener", ExpectedReplicas: 2},
				Product{Name: "backend-worker", ExpectedReplicas: 2},
				Product{Name: "system-app", ExpectedReplicas: 2},
				Product{Name: "system-memcache", ExpectedReplicas: 1},
				Product{Name: "system-sidekiq", ExpectedReplicas: 2},
				Product{Name: "system-sphinx", ExpectedReplicas: 1},
				Product{Name: "zync", ExpectedReplicas: 2},
				Product{Name: "zync-database", ExpectedReplicas: 1},
				Product{Name: "zync-que", ExpectedReplicas: 2},
			},
		}}
	} else {
		return []Namespace{
			getDeploymentConfiguration("threeScaleDeploymentConfig"),
			getDeploymentConfiguration("fuseDeploymentConfig"),
			getDeploymentConfiguration("solutionExplorerDeploymentConfig"),
		}
	}
}

func TestStatefulSetsExpectedReplicas(t *testing.T, ctx *TestingContext) {
	statefulSets := []Namespace{
		{
			Name: MonitoringOperatorNamespace,
			Products: []Product{
				Product{Name: "alertmanager-application-monitoring", ExpectedReplicas: 1},
				Product{Name: "prometheus-application-monitoring", ExpectedReplicas: 1},
			},
		},
		{
			Name: NamespacePrefix + "rhsso",
			Products: []Product{
				Product{Name: "keycloak", ExpectedReplicas: 2},
			},
		},
		{
			Name: NamespacePrefix + "user-sso",
			Products: []Product{
				Product{Name: "keycloak", ExpectedReplicas: 2},
			},
		},
	}

	for _, namespace := range statefulSets {
		for _, product := range namespace.Products {
			statefulSet, err := ctx.KubeClient.AppsV1().StatefulSets(namespace.Name).Get(product.Name, v1.GetOptions{})
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
