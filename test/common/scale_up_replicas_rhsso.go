package common

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	numberOfRhssoReplicas  int = 2
	scaleUpRhssoReplicas   int = 3
	scaleDownRhssoReplicas int = 1
	rhssoName                  = "rhsso"
	userSSOName                = "rhssouser"
	requestURLSSO              = "/apis/keycloak.org/v1alpha1"
	kindSSO                    = "Keycloaks"
)

func TestReplicasInRHSSOAndUserSSO(t *testing.T, ctx *TestingContext) {
	checkScalingOfKeycloakReplicas(t, ctx, rhssoName, GetPrefixedNamespace("rhsso"))
	checkScalingOfKeycloakReplicas(t, ctx, userSSOName, GetPrefixedNamespace("user-sso"))
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

	keycloakCR, err = updateKeycloakCR(ctx, scaleUpRhssoReplicas, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, scaleUpRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	keycloakCR, err = updateKeycloakCR(ctx, scaleDownRhssoReplicas, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, numberOfRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	keycloakCR, err = updateKeycloakCR(ctx, numberOfRhssoReplicas, keycloakCRName, keycloakCRNamespace)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of replicas has been reset")
	if err := checkNumberOfReplicasAgainstValueRhsso(keycloakCR, ctx, numberOfRhssoReplicas, retryInterval, timeout, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	t.Log("Checking replicas are ready")
	if err := checkSSOReplicasAreReady(ctx, t, int32(numberOfRhssoReplicas), keycloakCRName, keycloakCRNamespace, retryInterval, timeout); err != nil {
		t.Fatalf("Replicas weren't ready within timeout")
	}
}

func getKeycloakCR(dynClient k8sclient.Client, keycloakCRName string, keycloakCRNamespace string) (keycloakv1alpha1.Keycloak, error) {
	keycloakCR := &keycloakv1alpha1.Keycloak{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: keycloakCRName, Namespace: keycloakCRNamespace}, keycloakCR); err != nil {
		return *keycloakCR, err
	}

	return *keycloakCR, nil
}

func updateKeycloakCR(dynClient *TestingContext, replicas int, keycloakCRName string, keycloakCRNamespace string) (keycloakv1alpha1.Keycloak, error) {

	replica := fmt.Sprintf(`{
		"apiVersion": "keycloak.org/v1alpha1",
		"kind": "Keycloak",
		"spec": {
			"instances": %[1]v
		}		
	}`, replicas)

	replicaBytes := []byte(replica)

	request := dynClient.ExtensionClient.RESTClient().Patch(types.MergePatchType).
		Resource(kindSSO).
		Name(keycloakCRName).
		Namespace(keycloakCRNamespace).
		RequestURI(requestURLSSO).Body(replicaBytes).Do()
	_, err := request.Raw()

	keycloakCR, err := getKeycloakCR(dynClient.Client, keycloakCRName, keycloakCRNamespace)
	if err != nil {
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

func checkSSOReplicasAreReady(dynClient *TestingContext, t *testing.T, replicas int32, keycloakCRName string, keycloakCRNamespace string, retryInterval, timeout time.Duration) error {

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {

		statefulset, err := dynClient.KubeClient.AppsV1().StatefulSets(keycloakCRNamespace).Get("keycloak", metav1.GetOptions{})
		if err != nil {
			t.Errorf("Failed to get Statefulset %s in namespace %s with error: %s", "Keycloak", keycloakCRNamespace, err)
		}
		if statefulset.Status.ReadyReplicas != replicas {
			t.Logf("Replicas Ready %v", statefulset.Status.ReadyReplicas)
			t.Logf("retrying in : %v seconds", retryInterval)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("Number of replicas for sso.Spec.replicas is not correct : Replicas - %v, Expected - %v", err, replicas)
	}
	return nil
}
