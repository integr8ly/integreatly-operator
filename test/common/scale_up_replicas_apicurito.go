package common

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	apicuritov1alpha1 "github.com/apicurio/apicurio-operators/apicurito/pkg/apis/apicur/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numberOfApicuritoReplicas  = 2 //size in reconciler
	scaleUpApicuritoReplicas   = 3
	scaleDownApicuritoReplicas = 1
	apicuritoName              = "apicurito"
	apicuritoNamespace         = "redhat-rhmi-apicurito"
	retryIntervalApicurito     = time.Second * 20
	timeoutApicurito           = time.Minute * 7
)

func TestReplicasInApicurito(t *testing.T, ctx *TestingContext) {

	apicuritoCR, err := getApicurito(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get Apicurito : %v", err)
	}

	t.Log("Checking correct number of replicas are set")
	if err := checkNumberOfReplicasAgainstValueApicurito(apicuritoCR, ctx, numberOfApicuritoReplicas, retryIntervalApicurito, timeoutApicurito, t); err != nil {
		t.Fatalf("Incorrect number of replicas to start : %v", err)
	}

	apicuritoCR, err = updateApicurito(ctx.Client, scaleUpApicuritoReplicas, t)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueApicurito(apicuritoCR, ctx, scaleUpApicuritoReplicas, retryIntervalApicurito, timeoutApicurito, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	apicuritoCR, err = updateApicurito(ctx.Client, scaleDownApicuritoReplicas, t)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueApicurito(apicuritoCR, ctx, numberOfApicuritoReplicas, retryIntervalApicurito, timeoutApicurito, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

}

func getApicurito(dynClient k8sclient.Client) (apicuritov1alpha1.Apicurito, error) {
	apicuritoCR := &apicuritov1alpha1.Apicurito{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: apicuritoName, Namespace: apicuritoNamespace}, apicuritoCR); err != nil {
		return *apicuritoCR, err
	}

	return *apicuritoCR, nil
}

func updateApicurito(dynClient k8sclient.Client, replicas int32, t *testing.T) (apicuritov1alpha1.Apicurito, error) {

	apicuritoCR, err := getApicurito(dynClient)

	if err != nil {
		return apicuritoCR, err
	}
	apicuritoCR = apicuritov1alpha1.Apicurito{
		ObjectMeta: metav1.ObjectMeta{
			Name:            apicuritoName,
			Namespace:       apicuritoNamespace,
			ResourceVersion: apicuritoCR.GetResourceVersion(),
		},
		Spec: apicuritov1alpha1.ApicuritoSpec{
			Image: apicuritoCR.Spec.Image,
			Size:  replicas,
		},
	}

	if err := dynClient.Update(goctx.TODO(), apicuritoCR.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return apicuritoCR, err
	}

	return apicuritoCR, nil
}

func checkNumberOfReplicasAgainstValueApicurito(apicuritoCR apicuritov1alpha1.Apicurito, ctx *TestingContext, numberOfRequiredReplicas int32, retryInterval, timeout time.Duration, t *testing.T) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		apicuritoCR, err = getApicurito(ctx.Client)
		if err != nil {
			t.Fatalf("failed to get Apicurito : %v", err)
		}
		if *&apicuritoCR.Spec.Size == numberOfRequiredReplicas {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("Number of replicas for apicurito.Spec.Size is not correct : Replicas - %v, Expected - %v", *&apicuritoCR.Spec.Size, numberOfRequiredReplicas)
	}
	return nil
}
