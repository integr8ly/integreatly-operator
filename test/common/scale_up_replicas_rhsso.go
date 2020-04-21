package common

import (
	goctx "context"
	"testing"
	"time"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numberOfRhssoReplicas  int = 2
	scaleUpRhssoReplicas   int = 3
	scaleDownRhssoReplicas int = 1
	rhssoName                  = "rhsso"
	rhssoNamespace             = "redhat-rhmi-rhsso"
	userSSOName                = "rhssouser"
	userSSONamespace           = "redhat-rhmi-user-sso"
)

func TestReplicasInRHSSOAndUserSSO(t *testing.T, ctx *TestingContext) {
	checkScalingOfKeycloakReplicas(t, ctx, rhssoName, rhssoNamespace)
	checkScalingOfKeycloakReplicas(t, ctx, userSSOName, userSSONamespace)
}

func checkScalingOfKeycloakReplicas(t *testing.T, ctx *TestingContext, keycloakCRName string, keycloakCRNamespace string) {
	keycloakCR, err := getKeycloakCR(ctx.Client, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("failed to get KeycloakCR : %v", err)
	}

	t.Log("Checking correct number of replicas are set")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, numberOfRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas to start : %v", err)
	}

	keycloakCR, err = updateKeycloakCR(ctx.Client, scaleUpRhssoReplicas, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, scaleUpRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	keycloakCR, err = updateKeycloakCR(ctx.Client, scaleDownRhssoReplicas, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, numberOfRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	keycloakCR, err = updateKeycloakCR(ctx.Client, numberOfRhssoReplicas, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of replicas has been reset")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, numberOfRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}
}

func getKeycloakCR(dynClient k8sclient.Client, keycloakCRName string, keycloakCRNamespace string) (keycloakv1alpha1.Keycloak, error) {
	keycloakCR := &keycloakv1alpha1.Keycloak{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: keycloakCRName, Namespace: keycloakCRNamespace}, keycloakCR); err != nil {
		return *keycloakCR, err
	}

	return *keycloakCR, nil
}

func updateKeycloakCR(dynClient k8sclient.Client, replicas int, keycloakCRName string, keycloakCRNamespace string) (keycloakv1alpha1.Keycloak, error) {

	keycloakCR, err := getKeycloakCR(dynClient, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		return keycloakCR, err
	}
	keycloakCR = keycloakv1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:            keycloakCRName,
			Namespace:       keycloakCRNamespace,
			ResourceVersion: keycloakCR.GetResourceVersion(),
		},
		Spec: keycloakv1alpha1.KeycloakSpec{
			Instances: replicas,
		},
	}

	if err := dynClient.Update(goctx.TODO(), keycloakCR.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return keycloakCR, err
	}

	return keycloakCR, nil
}

func checkNumberOfReplicasAgainstValueRhsso(keycloakCR keycloakv1alpha1.Keycloak, ctx *TestingContext, numberOfRequiredReplicas int, retryInterval, timeout time.Duration, t *testing.T) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		keycloakCR, err = getKeycloakCR(ctx.Client, keycloakCR.Name, keycloakCR.Namespace)
		if err != nil {
			t.Fatalf("failed to get KeycloakCR : %v", err)
		}
		if keycloakCR.Spec.Instances != numberOfRequiredReplicas {
			t.Logf("Number of replicas for keycloakCR.Spec.Instances is not correct : Replicas - %v, Expected - %v", keycloakCR.Spec.Instances, numberOfRequiredReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		return true, nil
	})
}
