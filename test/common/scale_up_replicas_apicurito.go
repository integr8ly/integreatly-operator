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

var (
	numberOfApicuritoReplicas  = 2 //size in reconciler
	scaleUpApicuritoReplicas   = 3
	scaleDownApicuritoReplicas = 1
	apicuritoName              = "apicurito"
	apicuritoNamespace         = NamespacePrefix + "apicurito"
	retryIntervalApicurito     = time.Second * 20
	timeoutApicurito           = time.Minute * 7
	requestURLApicturito       = "/apis/apicur.io/v1alpha1"
	kindApicurito              = "apicuritos"
)

func TestReplicasInApicurito(t *testing.T, ctx *TestingContext) {

	apicuritoCR, err := getApicurito(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get Apicurito : %v", err)
	}

	t.Log("Checking correct number of replicas are set")
	if err := checkNumberOfReplicasAgainstValueApicurito(apicuritoCR, ctx, int32(numberOfApicuritoReplicas), retryIntervalApicurito, timeoutApicurito, t); err != nil {
		t.Fatalf("Incorrect number of replicas to start : %v", err)
	}

	apicuritoCR, err = updateApicurito(ctx, int32(scaleUpApicuritoReplicas), t)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueApicurito(apicuritoCR, ctx, int32(scaleUpApicuritoReplicas), retryIntervalApicurito, timeoutApicurito, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	apicuritoCR, err = updateApicurito(ctx, int32(scaleDownApicuritoReplicas), t)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueApicurito(apicuritoCR, ctx, int32(numberOfApicuritoReplicas), retryIntervalApicurito, timeoutApicurito, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	t.Log("Checking replicas are ready")
	if err := checkReplicasAreReady(ctx, t, int32(numberOfApicuritoReplicas)); err != nil {
		t.Fatalf("Replicas weren't ready within timeout")
	}

}

func checkReplicasAreReady(dynClient *TestingContext, t *testing.T, replicas int32) error {

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {

		deployment, err := dynClient.KubeClient.AppsV1().Deployments(apicuritoNamespace).Get(apicuritoName, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Failed to get Deployment %s in namespace %s with error: %s", apicuritoName, apicuritoNamespace, err)
		}
		if deployment.Status.ReadyReplicas == replicas {
			t.Logf("Replicas Ready %v", deployment.Status.ReadyReplicas)
			return true, nil
		}
		return false, fmt.Errorf("%v", deployment.Status.ReadyReplicas)
	})
	if err != nil {
		return fmt.Errorf("Number of replicas for apicurito.Spec.Size is not correct : Replicas - %v, Expected - %v", err, replicas)
	}
	return nil
}

func getApicurito(dynClient k8sclient.Client) (apicuritov1alpha1.Apicurito, error) {
	apicuritoCR := &apicuritov1alpha1.Apicurito{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: apicuritoName, Namespace: apicuritoNamespace}, apicuritoCR); err != nil {
		return *apicuritoCR, err
	}

	return *apicuritoCR, nil
}

func updateApicurito(ctx *TestingContext, replicas int32, t *testing.T) (apicuritov1alpha1.Apicurito, error) {

	replica := fmt.Sprintf(`{
		"apiVersion": "apicur.io/v1alpha1",
		"kind": "Apicurito",
		"spec": {
			"size": %[1]v
		}
	}`, replicas)

	replicaBytes := []byte(replica)

	request := ctx.ExtensionClient.RESTClient().Patch(types.MergePatchType).
		Resource(kindApicurito).
		Name(apicuritoName).
		Namespace(apicuritoNamespace).
		RequestURI(requestURLApicturito).Body(replicaBytes).Do()
	_, err := request.Raw()

	apicuritoCR, err := getApicurito(ctx.Client)
	if err != nil {
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
