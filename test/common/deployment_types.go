package common

import (
	goctx "context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	appsv1 "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	threeScaleDeployment = Namespace{
		Name: ThreeScaleOperatorNamespace,
		Products: []Product{
			Product{Name: "3scale-operator", ExpectedReplicas: 1},
		},
	}
	aMQOnlineOperatorDeployment = Namespace{
		Name: AMQOnlineOperatorNamespace,
		Products: []Product{
			Product{Name: "address-space-controller", ExpectedReplicas: 1},
			Product{Name: "console", ExpectedReplicas: 1},
			Product{Name: "enmasse-operator", ExpectedReplicas: 1},
			Product{Name: "none-authservice", ExpectedReplicas: 1},
			Product{Name: "standard-authservice", ExpectedReplicas: 1},
		},
	}
	cloudResourceOperatorDeployment = Namespace{
		Name: CloudResourceOperatorNamespace,
		Products: []Product{
			Product{Name: "cloud-resource-operator", ExpectedReplicas: 1},
		},
	}
	codeReadyOperatorDeployment = Namespace{
		Name: CodeReadyOperatorNamespace,
		Products: []Product{
			Product{Name: "codeready-operator", ExpectedReplicas: 1},
		},
	}
	codereadyWorkspacesDeployment = Namespace{
		Name: "redhat-rhmi-codeready-workspaces",
		Products: []Product{
			Product{Name: "codeready", ExpectedReplicas: 1},
			Product{Name: "devfile-registry", ExpectedReplicas: 1},
			Product{Name: "plugin-registry", ExpectedReplicas: 1},
		},
	}
	fuseOperatorDeployment = Namespace{
		Name: FuseOperatorNamespace,
		Products: []Product{
			Product{Name: "syndesis-operator", ExpectedReplicas: 1},
		},
	}
	monitoringOperatorDeployment = Namespace{
		Name: MonitoringOperatorNamespace,
		Products: []Product{
			Product{Name: "application-monitoring-operator", ExpectedReplicas: 1},
			Product{Name: "grafana-deployment", ExpectedReplicas: 1},
			Product{Name: "grafana-operator", ExpectedReplicas: 1},
			Product{Name: "prometheus-operator", ExpectedReplicas: 1},
		},
	}
	rhmiOperatorDeployment = Namespace{
		Name: RHMIOperatorNamespace,
		Products: []Product{
			Product{Name: "standard-authservice-postgresql", ExpectedReplicas: 1},
		},
	}
	rhssoOperatorDeployment = Namespace{
		Name: RHSSOOperatorNamespace,
		Products: []Product{
			Product{Name: "keycloak-operator", ExpectedReplicas: 1},
		},
	}
	solutionExplorerOperatorDeployment = Namespace{
		Name: SolutionExplorerOperatorNamespace,
		Products: []Product{
			Product{Name: "tutorial-web-app-operator", ExpectedReplicas: 1},
		},
	}
	upsOperatorDeployment = Namespace{
		Name: UPSOperatorNamespace,
		Products: []Product{
			Product{Name: "unifiedpush-operator", ExpectedReplicas: 1},
		},
	}
	upsDeployment = Namespace{
		Name: "redhat-rhmi-ups",
		Products: []Product{
			Product{Name: "ups", ExpectedReplicas: 1},
		},
	}
	rhssoUserOperatorDeployment = Namespace{
		Name: RHSSOUserOperatorNamespace,
		Products: []Product{
			Product{Name: "keycloak-operator", ExpectedReplicas: 1},
		},
	}

	// Applicable to install types used in 2.X
	rhmi2Deployments = []Namespace{
		aMQOnlineOperatorDeployment,
		codeReadyOperatorDeployment,
		codereadyWorkspacesDeployment,
		fuseOperatorDeployment,
		solutionExplorerOperatorDeployment,
		upsOperatorDeployment,
		upsDeployment,
	}

	// Common to all install types including managed api
	commonApiDeployments = []Namespace{
		threeScaleDeployment,
		cloudResourceOperatorDeployment,
		monitoringOperatorDeployment,
		rhmiOperatorDeployment,
		rhssoOperatorDeployment,
		rhssoUserOperatorDeployment,
	}

	rhmi2ClusterStorageDeployments = []Namespace{
		{
			Name: "redhat-rhmi-operator",
			Products: []Product{
				Product{Name: constants.CodeReadyPostgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.ThreeScaleBackendRedisPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.ThreeScalePostgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.ThreeScaleSystemRedisPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.UPSPostgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.RHSSOPostgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.RHSSOUserProstgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.AMQAuthServicePostgres, ExpectedReplicas: 1},
			},
		},
	}
	managedApiClusterStorageDeployments = []Namespace{
		{
			Name: "redhat-rhmi-operator",
			Products: []Product{
				Product{Name: constants.ThreeScaleBackendRedisPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.ThreeScalePostgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.ThreeScaleSystemRedisPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.RHSSOPostgresPrefix + InstallationName, ExpectedReplicas: 1},
				Product{Name: constants.RHSSOUserProstgresPrefix + InstallationName, ExpectedReplicas: 1},
			},
		},
	}

	threeScaleDeploymentConfig = Namespace{
		Name: "redhat-rhmi-3scale",
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
	}

	fuseDeploymentConfig = Namespace{
		Name: "redhat-rhmi-fuse",
		Products: []Product{
			Product{Name: "syndesis-meta", ExpectedReplicas: 1},
			Product{Name: "syndesis-oauthproxy", ExpectedReplicas: 1},
			Product{Name: "syndesis-prometheus", ExpectedReplicas: 1},
			Product{Name: "syndesis-server", ExpectedReplicas: 1},
			Product{Name: "syndesis-ui", ExpectedReplicas: 1},
			Product{Name: "broker-amq", ExpectedReplicas: 1},
		},
	}

	solutionExplorerDeploymentConfig = Namespace{
		Name: "redhat-rhmi-solution-explorer",
		Products: []Product{
			Product{Name: "tutorial-web-app", ExpectedReplicas: 1},
		},
	}

	rhmi2Dcs = []Namespace{threeScaleDeploymentConfig, fuseDeploymentConfig, solutionExplorerDeploymentConfig}

	managedApiDcs = []Namespace{threeScaleDeploymentConfig}

	statefulSets = []Namespace{
		{
			Name: MonitoringOperatorNamespace,
			Products: []Product{
				Product{Name: "alertmanager-application-monitoring", ExpectedReplicas: 1},
				Product{Name: "prometheus-application-monitoring", ExpectedReplicas: 1},
			},
		},
		{
			Name: "redhat-rhmi-rhsso",
			Products: []Product{
				Product{Name: "keycloak", ExpectedReplicas: 2},
			},
		},
		{
			Name: "redhat-rhmi-user-sso",
			Products: []Product{
				Product{Name: "keycloak", ExpectedReplicas: 2},
			},
		},
	}
)

func TestDeploymentExpectedReplicas(t *testing.T, ctx *TestingContext) {

	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	deployments := getDeployments(rhmi.Spec.Type)
	clusterStorageDeployments := getClusterStorageDeployments(rhmi.Spec.Type)

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

			if deployment.Status.Replicas != product.ExpectedReplicas {
				t.Errorf("Deployment %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deployment.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}

			// Verify that the expected replicas are also available, means they are up and running and consumable by users
			if deployment.Status.AvailableReplicas != product.ExpectedReplicas {
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
	if installType == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return commonApiDeployments
	} else {
		return append(commonApiDeployments, rhmi2Deployments...)
	}
}

func getClusterStorageDeployments(installType string) []Namespace {
	if installType == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return managedApiClusterStorageDeployments
	} else {
		return rhmi2ClusterStorageDeployments
	}
}

func TestDeploymentConfigExpectedReplicas(t *testing.T, ctx *TestingContext) {
	rhmi, err := getRHMI(ctx.Client)
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

			if deploymentConfig.Status.Replicas != product.ExpectedReplicas {
				t.Errorf("DeploymentConfig %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deploymentConfig.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}

			if deploymentConfig.Status.AvailableReplicas != product.ExpectedReplicas {
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
	if installType == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return managedApiDcs
	} else {
		return rhmi2Dcs
	}
}

func TestStatefulSetsExpectedReplicas(t *testing.T, ctx *TestingContext) {

	for _, namespace := range statefulSets {
		for _, product := range namespace.Products {
			statefulSet, err := ctx.KubeClient.AppsV1().StatefulSets(namespace.Name).Get(product.Name, v1.GetOptions{})
			if err != nil {
				t.Errorf("Failed to get StatefulSet %s in namespace %s with error: %s", product.Name, namespace.Name, err)
				continue
			}

			if statefulSet.Status.Replicas != product.ExpectedReplicas {
				t.Errorf("StatefulSet %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					statefulSet.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}

			// Verify the number of ReadyReplicas because the SatefulSet doesn't have the concept of AvailableReplicas
			if statefulSet.Status.ReadyReplicas != product.ExpectedReplicas {
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
