package common

import (
	goctx "context"
	"fmt"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/config"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	name             = "3scale"
	retryInterval    = time.Second * 20
	timeout          = time.Minute * 7
	requestURL3scale = "/apis/apps.3scale.net/v1alpha1"
	kind             = "APIManagers"
)

var (
	threeScaleDeploymentConfigs = map[string]string{
		"apicast-production": "apicastProd",
		"apicast-staging":    "apicastStage",
		"backend-cron":       "backendCron",
		"backend-listener":   "backendListener",
		"backend-worker":     "backendWorker",
		"system-app":         "systemApp",
		"system-sidekiq":     "systemSidekiq",
		"zync":               "zyncApp",
		"zync-que":           "zyncQue",
	}
)

func TestReplicasInThreescale(t TestingTB, ctx *TestingContext) {

	apim, err := getAPIManager(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get APIManager : %v", err)
	}
	inst, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get RHMI instance %v", err)
	}

	threescaleConfig := config.NewThreeScale(map[string]string{})
	replicas := threescaleConfig.GetReplicasConfig(inst)

	scaleUpReplicas := map[string]int64{}
	for k, v := range replicas {
		scaleUpReplicas[k] = v + 1
	}
	scaleDownReplicas := map[string]int64{}
	for k, v := range replicas {
		scaleDownReplicas[k] = v - 1
	}

	t.Log("Checking correct number of replicas are set")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, replicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas to start : %v", err)
	}

	apim, err = updateAPIManager(ctx, scaleUpReplicas)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, scaleUpReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	apim, err = updateAPIManager(ctx, scaleDownReplicas)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, replicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	apim, err = updateAPIManager(ctx, replicas)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of replicas has been reset")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, replicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	if err := check3ScaleReplicasAreReady(ctx, t, replicas, retryInterval, timeout); err != nil {
		t.Fatalf("Replicas not Ready within timeout: %v", err)
	}

}

func getAPIManager(dynClient k8sclient.Client) (threescalev1.APIManager, error) {
	apim := &threescalev1.APIManager{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: GetPrefixedNamespace("3scale")}, apim); err != nil {
		return *apim, err
	}
	return *apim, nil
}

func updateAPIManager(dynClient *TestingContext, replicas map[string]int64) (threescalev1.APIManager, error) {

	replica := fmt.Sprintf(
		`{
			"apiVersion": "apps.3scale.net/v1alpha1",
			"kind": "APIManager",
			"spec": {
				"system": {
					"appSpec": {
						"replicas": %v
					},
					"sidekiqSpec": {
						"replicas": %v
					}
				},
				"apicast": {
					"productionSpec": {
						"replicas": %v
					},
					"stagingSpec": {
						"replicas": %v
					}
				},
				"backend": {
					"listenerSpec": {
						"replicas": %v
					},
					"cronSpec": {
						"replicas": %v
					},
					"workerSpec": {
						"replicas": %v
					}
				},
				"zync": {
					"appSpec": {
						"replicas": %v
					},
					"queSpec": {
						"replicas": %v
					}
				}
			}
		}`,
		replicas["systemApp"],
		replicas["systemSidekiq"],
		replicas["apicastProd"],
		replicas["apicastStage"],
		replicas["backendListener"],
		replicas["backendCron"],
		replicas["backendWorker"],
		replicas["zyncApp"],
		replicas["zyncQue"],
	)

	replicaBytes := []byte(replica)

	request := dynClient.ExtensionClient.RESTClient().Patch(types.MergePatchType).
		Resource(kind).
		Name(name).
		Namespace(GetPrefixedNamespace("3scale")).
		RequestURI(requestURL3scale).Body(replicaBytes).Do(goctx.TODO())
	_, err := request.Raw()
	if err != nil {
		return threescalev1.APIManager{}, err
	}

	apim, err := getAPIManager(dynClient.Client)
	if err != nil {
		return apim, err
	}

	return apim, nil
}

func checkNumberOfReplicasAgainstValue(apim threescalev1.APIManager, ctx *TestingContext, replicas map[string]int64, retryInterval, timeout time.Duration, t TestingTB) error {
	return wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		apim, err = getAPIManager(ctx.Client)
		if err != nil {
			t.Fatalf("failed to get APIManager : %v", err)
		}

		if *apim.Spec.System.AppSpec.Replicas != replicas["systemApp"] {
			t.Logf("Number of replicas for apim.Spec.System.AppSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.System.AppSpec.Replicas, replicas["systemApp"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.System.SidekiqSpec.Replicas != replicas["systemSidekiq"] {
			t.Logf("Number of replicas for apim.Spec.System.SidekiqSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.System.SidekiqSpec.Replicas, replicas["systemSidekiq"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Apicast.ProductionSpec.Replicas != replicas["apicastProd"] {
			t.Logf("Number of replicas for apim.Spec.Apicast.ProductionSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Apicast.ProductionSpec.Replicas, replicas["apicastProd"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Apicast.StagingSpec.Replicas != replicas["apicastStage"] {
			t.Logf("Number of replicas for apim.Spec.Apicast.StagingSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Apicast.StagingSpec.Replicas, replicas["apicastStage"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Backend.ListenerSpec.Replicas != replicas["backendListener"] {
			t.Logf("Number of replicas for apim.Spec.Backend.ListenerSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Backend.ListenerSpec.Replicas, replicas["backendListener"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Backend.WorkerSpec.Replicas != replicas["backendWorker"] {
			t.Logf("Number of replicas for apim.Spec.Backend.WorkerSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Backend.WorkerSpec.Replicas, replicas["backendWorker"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Backend.CronSpec.Replicas != replicas["backendCron"] {
			t.Logf("Number of replicas for apim.Spec.Backend.CronSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Backend.CronSpec.Replicas, replicas["backendCron"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Zync.AppSpec.Replicas != replicas["zyncApp"] {
			t.Logf("Number of replicas for apim.Spec.Zync.AppSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Zync.AppSpec.Replicas, replicas["zyncApp"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Zync.QueSpec.Replicas != replicas["zyncQue"] {
			t.Logf("Number of replicas for apim.Spec.Zync.QueSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Zync.QueSpec.Replicas, replicas["zyncQue"])
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		return true, nil
	})
}

func check3ScaleReplicasAreReady(ctx *TestingContext, t TestingTB, replicas map[string]int64, retryInterval, timeout time.Duration) error {

	return wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {

		for name, replicasID := range threeScaleDeploymentConfigs {
			deploymentConfig := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: GetPrefixedNamespace("3scale")}}

			err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: name, Namespace: GetPrefixedNamespace("3scale")}, deploymentConfig)
			if err != nil {
				t.Errorf("failed to get DeploymentConfig %s in namespace %s with error: %s", name, GetPrefixedNamespace("3scale"), err)
			}

			if deploymentConfig.Status.ReadyReplicas != int32(replicas[replicasID]) || deploymentConfig.Status.UnavailableReplicas != 0 {
				t.Logf("%s replicas ready %v, expected %v, unavailable replicas %v ", name, deploymentConfig.Status.ReadyReplicas, replicas[replicasID], deploymentConfig.Status.UnavailableReplicas)
				return false, nil
			}
		}

		return true, nil
	})
}
