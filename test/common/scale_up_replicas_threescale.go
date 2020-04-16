package common

import (
	goctx "context"
	"testing"
	"time"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numberOfReplicas  int64 = 2
	scaleUpReplicas   int64 = 3
	scaleDownReplicas int64 = 1
	name                    = "3scale"
	namespace               = "redhat-rhmi-3scale"
	retryInterval           = time.Second * 20
	timeout                 = time.Minute * 7
)

func TestReplicasInThreescale(t *testing.T, ctx *TestingContext) {

	apim, err := getAPIManager(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get APIManager : %v", err)
	}

	t.Log("Checking correct number of replicas are set")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, numberOfReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas to start : %v", err)
	}

	apim, err = updateAPIManager(ctx.Client, scaleUpReplicas)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, scaleUpReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	apim, err = updateAPIManager(ctx.Client, scaleDownReplicas)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, numberOfReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	apim, err = updateAPIManager(ctx.Client, numberOfReplicas)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of replicas has been reset")
	if err := checkNumberOfReplicasAgainstValue(apim, ctx, numberOfReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}
}

func getAPIManager(dynClient k8sclient.Client) (threescalev1.APIManager, error) {
	apim := &threescalev1.APIManager{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, apim); err != nil {
		return *apim, err
	}

	return *apim, nil
}

func updateAPIManager(dynClient k8sclient.Client, replicas int64) (threescalev1.APIManager, error) {

	apim, err := getAPIManager(dynClient)
	if err != nil {
		return apim, err
	}
	resourceRequirements := true
	apim = threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: apim.GetResourceVersion(),
		},
		Spec: threescalev1.APIManagerSpec{
			HighAvailability: &threescalev1.HighAvailabilitySpec{},
			APIManagerCommonSpec: threescalev1.APIManagerCommonSpec{
				ResourceRequirementsEnabled: &resourceRequirements,
			},
			System: &threescalev1.SystemSpec{
				DatabaseSpec: &threescalev1.SystemDatabaseSpec{
					PostgreSQL: &threescalev1.SystemPostgreSQLSpec{},
				},
				FileStorageSpec: &threescalev1.SystemFileStorageSpec{
					S3: &threescalev1.SystemS3Spec{},
				},
				AppSpec:     &threescalev1.SystemAppSpec{Replicas: &[]int64{replicas}[0]},
				SidekiqSpec: &threescalev1.SystemSidekiqSpec{Replicas: &[]int64{replicas}[0]},
			},
			Apicast: &threescalev1.ApicastSpec{
				ProductionSpec: &threescalev1.ApicastProductionSpec{Replicas: &[]int64{replicas}[0]},
				StagingSpec:    &threescalev1.ApicastStagingSpec{Replicas: &[]int64{replicas}[0]},
			},
			Backend: &threescalev1.BackendSpec{
				ListenerSpec: &threescalev1.BackendListenerSpec{Replicas: &[]int64{replicas}[0]},
				WorkerSpec:   &threescalev1.BackendWorkerSpec{Replicas: &[]int64{replicas}[0]},
				CronSpec:     &threescalev1.BackendCronSpec{Replicas: &[]int64{replicas}[0]},
			},
			Zync: &threescalev1.ZyncSpec{
				AppSpec: &threescalev1.ZyncAppSpec{Replicas: &[]int64{replicas}[0]},
				QueSpec: &threescalev1.ZyncQueSpec{Replicas: &[]int64{replicas}[0]},
			},
		},
	}

	if err := dynClient.Update(goctx.TODO(), apim.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return apim, err
	}

	return apim, nil
}

func checkNumberOfReplicasAgainstValue(apim threescalev1.APIManager, ctx *TestingContext, numberOfRequiredReplicas int64, retryInterval, timeout time.Duration, t *testing.T) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		apim, err = getAPIManager(ctx.Client)
		if err != nil {
			t.Fatalf("failed to get APIManager : %v", err)
		}
		if *apim.Spec.System.AppSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.System.AppSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.System.AppSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.System.SidekiqSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.System.SidekiqSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.System.SidekiqSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Apicast.ProductionSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Apicast.ProductionSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Apicast.ProductionSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Apicast.StagingSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Apicast.StagingSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Apicast.StagingSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Backend.ListenerSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Backend.ListenerSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Backend.ListenerSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Backend.WorkerSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Backend.WorkerSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Backend.WorkerSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Backend.CronSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Backend.CronSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Backend.CronSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Zync.AppSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Zync.AppSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Zync.AppSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		if *apim.Spec.Zync.QueSpec.Replicas != numberOfRequiredReplicas {
			t.Logf("Number of replicas for apim.Spec.Zync.QueSpec is not correct : Replicas - %v, Expected - %v", *apim.Spec.Zync.QueSpec.Replicas, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		return true, nil
	})
}
