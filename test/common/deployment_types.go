package common

import (
	goctx "context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/utils"
	"github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	"golang.org/x/net/context"
	k8sappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	commonApiDeploymentsList = []string{
		"threeScaleDeployment",
		"threeScaleProductDeployment",
		"cloudResourceOperatorDeployment",
		"observabilityDeployment",
		"rhssoOperatorDeployment",
	}
	managedApiDeploymentsList = []string{
		"marin3rOperatorDeployment",
		"marin3rDeployment",
		"rhssoUserOperatorDeployment",
	}
)

func getDeploymentConfiguration(deploymentName string, inst *integreatlyv1alpha1.RHMI, t TestingTB, ctx *TestingContext) Namespace {
	threescaleConfig := config.NewThreeScale(map[string]string{})
	replicas := threescaleConfig.GetReplicasConfig(inst)
	deployment := map[string]Namespace{
		"threeScaleDeployment": {
			Name: ThreeScaleOperatorNamespace,
			Products: []Product{
				{Name: "threescale-operator-controller-manager-v2", ExpectedReplicas: 1},
			},
		},
		"cloudResourceOperatorDeployment": {
			Name: CloudResourceOperatorNamespace,
			Products: []Product{
				{Name: "cloud-resource-operator", ExpectedReplicas: 1},
			},
		},
		"rhmiOperatorDeploymentForManagedApi": {
			Name:     RHOAMOperatorNamespace,
			Products: []Product{},
		},
		"rhssoOperatorDeployment": {
			Name: RHSSOOperatorNamespace,
			Products: []Product{
				{Name: "rhsso-operator", ExpectedReplicas: 1},
			},
		},
		"rhssoUserOperatorDeployment": {
			Name: RHSSOUserOperatorNamespace,
			Products: []Product{
				{Name: "rhsso-operator", ExpectedReplicas: 1},
			},
		},
		"marin3rOperatorDeployment": {
			Name: Marin3rOperatorNamespace,
			Products: []Product{
				{Name: "marin3r-controller-manager", ExpectedReplicas: 1},
				{Name: "marin3r-controller-webhook", ExpectedReplicas: 2},
			},
		},
		"threeScaleProductDeployment": {
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
				{Name: "system-searchd", ExpectedReplicas: 1},
				{Name: "zync", ExpectedReplicas: 1},
				{Name: "zync-database", ExpectedReplicas: int32(replicas["zyncDatabase"])},
				{Name: "zync-que", ExpectedReplicas: int32(replicas["zyncQue"])},
			},
		},
	}

	ratelimitCR := &k8sappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quota.RateLimitName,
			Namespace: Marin3rProductNamespace,
		},
	}

	key := k8sclient.ObjectKeyFromObject(ratelimitCR)

	err := ctx.Client.Get(context.TODO(), key, ratelimitCR)
	if err != nil {
		if !k8sError.IsNotFound(err) {
			t.Fatalf("Error obtaining ratelimit CR: %v", err)
		}
	}
	deployment["marin3rDeployment"] = Namespace{
		Name: Marin3rProductNamespace,
		Products: []Product{
			{Name: "ratelimit", ExpectedReplicas: *ratelimitCR.Spec.Replicas},
		},
	}

	return deployment[deploymentName]
}

func getClusterStorageDeployments(ctx *TestingContext, installationName string, installType string) []Namespace {

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
	mtManagedApiClusterStorageDeployments := []Namespace{
		{
			Name: NamespacePrefix + "operator",
			Products: []Product{
				{Name: constants.ThreeScaleBackendRedisPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScalePostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.ThreeScaleSystemRedisPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RHSSOPostgresPrefix + installationName, ExpectedReplicas: 1},
				{Name: constants.RateLimitRedisPrefix + installationName, ExpectedReplicas: 1},
			},
		},
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return mtManagedApiClusterStorageDeployments
	} else {
		return managedApiClusterStorageDeployments
	}
}

func TestDeploymentExpectedReplicas(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	deployments := getDeployments(rhmi, t, ctx)
	clusterStorageDeployments := getClusterStorageDeployments(ctx, rhmi.Name, rhmi.Spec.Type)

	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	// If the cluster is using in cluster storage instead of AWS resources
	// These deployments will also need to be checked
	if isClusterStorage {
		deployments = append(deployments, clusterStorageDeployments...)
	}

	for _, namespace := range deployments {
		for _, product := range namespace.Products {
			deployment, err := ctx.KubeClient.AppsV1().Deployments(namespace.Name).Get(goctx.TODO(), product.Name, metav1.GetOptions{})
			if err != nil {
				// Fail the test without failing immideatlly
				t.Errorf("Failed to get Deployment %s in namespace %s with error: %s", product.Name, namespace.Name, err)
				continue
			}

			if deployment.Status.ReadyReplicas < product.ExpectedReplicas {
				t.Errorf("Deployment %s in namespace %s doesn't match the number of expected replicas. Replicas: %v / Expected Replicas: %v",
					product.Name,
					namespace.Name,
					deployment.Status.Replicas,
					product.ExpectedReplicas,
				)
				continue
			}
		}
	}
}

func getDeployments(inst *integreatlyv1alpha1.RHMI, t TestingTB, ctx *TestingContext) []Namespace {
	var commonApiDeployments []Namespace
	var managedApiDeployments []Namespace

	for _, deployment := range commonApiDeploymentsList {
		commonApiDeployments = append(commonApiDeployments, getDeploymentConfiguration(deployment, inst, t, ctx))
	}
	for _, deployment := range managedApiDeploymentsList {
		managedApiDeployments = append(managedApiDeployments, getDeploymentConfiguration(deployment, inst, t, ctx))
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(inst.Spec.Type)) {
		return append(commonApiDeployments, []Namespace{getDeploymentConfiguration("rhmiOperatorDeploymentForManagedApi", inst, t, ctx)}...)
	} else {
		return append(append(commonApiDeployments, []Namespace{getDeploymentConfiguration("rhmiOperatorDeploymentForManagedApi", inst, t, ctx)}...), managedApiDeployments...)
	}
}

func getThreescaleProductDeployment(inst *integreatlyv1alpha1.RHMI, t TestingTB, ctx *TestingContext) []Namespace {
	return []Namespace{
		getDeploymentConfiguration("threeScaleProductDeployment", inst, t, ctx),
	}
}

func TestStatefulSetsExpectedReplicas(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	var rhssoExpectedReplicas int32 = 2
	var rhssoUserExpectedReplicas int32 = 3
	if utils.RunningInProw(rhmi) {
		rhssoExpectedReplicas = 1
		rhssoUserExpectedReplicas = 1
	}
	if integreatlyv1alpha1.IsRHOAMSingletenant(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		keycloakCR := &v1alpha1.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quota.KeycloakName,
				Namespace: RHSSOUserProductNamespace,
			},
		}
		key := k8sclient.ObjectKeyFromObject(keycloakCR)

		err = ctx.Client.Get(context.TODO(), key, keycloakCR)
		if err != nil {
			t.Fatalf("Error getting Keycloak CR: %v", err)
		}

		rhssoUserExpectedReplicas = int32(keycloakCR.Spec.Instances)
	}
	statefulSets := []Namespace{
		{
			Name: NamespacePrefix + "rhsso",
			Products: []Product{
				{Name: "keycloak", ExpectedReplicas: rhssoExpectedReplicas},
			},
		},
	}

	if integreatlyv1alpha1.IsRHOAMSingletenant(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		statefulSets = append(statefulSets, []Namespace{
			{
				Name: NamespacePrefix + "user-sso",
				Products: []Product{
					{Name: "keycloak", ExpectedReplicas: rhssoUserExpectedReplicas},
				},
			},
		}...)
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
	var err error
	for _, label := range podLabels {
		selector, err = labels.Parse(label)
		if err != nil {
			fmt.Printf("failed to get pods with error %v", err)
			return nil
		}
	}
	return []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
}
