package common

import (
	goctx "context"
	"testing"

	appsv1 "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Namespace struct {
	Name     string
	Products []Product
}

type Product struct {
	Name             string
	ExpectedReplicas int32
}

func TestDeploymentExpectedReplicas(t *testing.T, ctx *TestingContext) {
	deployment := getDeployment()

	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}
	// If the cluster is using in cluster storage instead of AWS resources
	// These deployments will also need to be checked
	if isClusterStorage {
		clusterStorageDeployments := getClusterStorageDeployments()
		for _, clusterStorageDeployment := range clusterStorageDeployments {
			deployment = append(deployment, clusterStorageDeployment)
		}
	}

	for _, namespace := range deployment {
		for _, product := range namespace.Products {
			deployment, err := ctx.KubeClient.AppsV1().Deployments(namespace.Name).Get(product.Name, v1.GetOptions{})
			if err != nil {
				t.Fatalf("Deployment %s not found - %s", product.Name, err)
			}
			if deployment.Status.Replicas != product.ExpectedReplicas {
				t.Fatalf("Deployment %s in namespace %s doens't match the number of expected replicas %s/%s",
					product.Name,
					namespace.Name,
					string(deployment.Status.Replicas),
					string(product.ExpectedReplicas),
				)
			}
		}
	}
}

func TestDeploymentConfigExpectedReplicas(t *testing.T, ctx *TestingContext) {

	deploymentConfigNS := getDeploymentConfig()
	for _, namespace := range deploymentConfigNS {
		for _, product := range namespace.Products {

			dc := &appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      product.Name,
					Namespace: namespace.Name,
				},
			}
			err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: product.Name, Namespace: namespace.Name}, dc)
			if err != nil {
				t.Fatalf("DeploymentConfig %s not found - %s", product.Name, err)
			}

			if dc.Status.Replicas != product.ExpectedReplicas {
				t.Fatalf("DeploymentConfig %s in namespace %s doens't match the number of expected replicas %s/%s",
					product.Name,
					namespace.Name,
					string(dc.Status.Replicas),
					string(product.ExpectedReplicas),
				)
			}

		}
	}
}

func TestStatefulSetsExpectedReplicas(t *testing.T, ctx *TestingContext) {

	statefulSetsNS := getStatefulSets()
	for _, namespace := range statefulSetsNS {
		for _, product := range namespace.Products {

			statefulSet, err := ctx.KubeClient.AppsV1().StatefulSets(namespace.Name).Get(product.Name, v1.GetOptions{})
			if err != nil {
				t.Fatalf("StatefulSet %s not found - %s", product.Name, err)
			}

			if statefulSet.Status.Replicas != product.ExpectedReplicas {
				t.Fatalf("StatefulSet %s in namespace %s doens't match the number of expected replicas %s/%s",
					product.Name,
					namespace.Name,
					string(statefulSet.Status.Replicas),
					string(product.ExpectedReplicas),
				)
			}

		}
	}
}

func getStatefulSets() []Namespace {
	return []Namespace{
		{
			Name: "redhat-rhmi-middleware-monitoring-operator",
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
}

func getDeploymentConfig() []Namespace {
	return []Namespace{
		{
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
		},
		{
			Name: "redhat-rhmi-fuse",
			Products: []Product{
				Product{Name: "broker-amq", ExpectedReplicas: 0},
				Product{Name: "syndesis-db", ExpectedReplicas: 1},
				Product{Name: "syndesis-meta", ExpectedReplicas: 1},
				Product{Name: "syndesis-oauthproxy", ExpectedReplicas: 1},
				Product{Name: "syndesis-prometheus", ExpectedReplicas: 1},
				Product{Name: "syndesis-server", ExpectedReplicas: 1},
				Product{Name: "syndesis-ui", ExpectedReplicas: 1},
			},
		},
		{
			Name: "redhat-rhmi-solution-explorer",
			Products: []Product{
				Product{Name: "tutorial-web-app", ExpectedReplicas: 1},
			},
		},
	}
}

func getDeployment() []Namespace {
	return []Namespace{
		{
			Name: "redhat-rhmi-3scale-operator",
			Products: []Product{
				Product{Name: "3scale-operator", ExpectedReplicas: 1},
			},
		},
		{
			Name: "redhat-rhmi-amq-online",
			Products: []Product{
				Product{Name: "address-space-controller", ExpectedReplicas: 1},
				Product{Name: "api-server", ExpectedReplicas: 1},
				Product{Name: "console", ExpectedReplicas: 1},
				Product{Name: "enmasse-operator", ExpectedReplicas: 1},
				Product{Name: "none-authservice", ExpectedReplicas: 1},
				Product{Name: "standard-authservice", ExpectedReplicas: 1},
				Product{Name: "user-api-server", ExpectedReplicas: 1},
			},
		},
		{
			Name: "redhat-rhmi-cloud-resources-operator",
			Products: []Product{
				Product{Name: "cloud-resource-operator", ExpectedReplicas: 1},
			},
		},
		{
			Name: "redhat-rhmi-codeready-workspaces-operator",
			Products: []Product{
				Product{Name: "codeready-operator", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-codeready-workspaces",
			Products: []Product{
				Product{Name: "codeready", ExpectedReplicas: 1},
				Product{Name: "devfile-registry", ExpectedReplicas: 1},
				Product{Name: "plugin-registry", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-fuse-operator",
			Products: []Product{
				Product{Name: "syndesis-operator", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-middleware-monitoring-operator",
			Products: []Product{
				Product{Name: "application-monitoring-operator", ExpectedReplicas: 1},
				Product{Name: "grafana-deployment", ExpectedReplicas: 1},
				Product{Name: "grafana-operator", ExpectedReplicas: 1},
				Product{Name: "prometheus-operator", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-operator",
			Products: []Product{
				Product{Name: "standard-authservice-postgresql", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-rhsso-operator",
			Products: []Product{
				Product{Name: "keycloak-operator", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-solution-explorer-operator",
			Products: []Product{
				Product{Name: "tutorial-web-app-operator", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-ups-operator",
			Products: []Product{
				Product{Name: "unifiedpush-operator", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-ups",
			Products: []Product{
				Product{Name: "ups", ExpectedReplicas: 1},
			},
		},
		Namespace{
			Name: "redhat-rhmi-user-sso-operator",
			Products: []Product{
				Product{Name: "keycloak-operator", ExpectedReplicas: 1},
			},
		},
	}
}

func getClusterStorageDeployments() []Namespace {
	return []Namespace{
		{
			Name: "redhat-rhmi-operator",
			Products: []Product{
				Product{Name: "codeready-postgres-rhmi", ExpectedReplicas: 1},
				Product{Name: "threescale-backend-redis-rhmi", ExpectedReplicas: 1},
				Product{Name: "threescale-postgres-rhmi", ExpectedReplicas: 1},
				Product{Name: "threescale-redis-rhmi", ExpectedReplicas: 1},
				Product{Name: "ups-postgres-rhmi", ExpectedReplicas: 1},
				Product{Name: "rhsso-postgres-rhmi", ExpectedReplicas: 1},
				Product{Name: "rhssouser-postgres-rhmi", ExpectedReplicas: 1},
				Product{Name: "standard-authservice-postgresql", ExpectedReplicas: 1},
			},
		},
	}
}
