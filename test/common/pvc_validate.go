package common

import (
	goctx "context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//Define list of namespaces and their claims
var (
	pvcNamespaces = []PersistentVolumeClaim{
		{

			Namespace: NamespacePrefix + "fuse",
			PersistentVolumeClaimNames: []string{
				"syndesis-meta",
				"syndesis-prometheus",
			},
		},

		{

			Namespace: NamespacePrefix + "middleware-monitoring-operator",
			PersistentVolumeClaimNames: []string{
				"prometheus-application-monitoring-db-prometheus-application-monitoring-0",
			},
		},

		{

			Namespace: NamespacePrefix + "solution-explorer",
			PersistentVolumeClaimNames: []string{
				"user-walkthroughs",
			},
		},
		{

			Namespace: NamespacePrefix + "operator",
			PersistentVolumeClaimNames: []string{
				"standard-authservice-postgresql",
			},
		},
	}
)

func TestPVClaims(t *testing.T, ctx *TestingContext) {

	//get full list of volume claim items
	pvcs := &corev1.PersistentVolumeClaimList{}

	for _, pvcNamespace := range pvcNamespaces {
		err := ctx.Client.List(goctx.TODO(), pvcs, &k8sclient.ListOptions{Namespace: pvcNamespace.Namespace})
		if err != nil {
			t.Errorf("Error getting PVCs for namespace: %v %w", pvcNamespace.Namespace, err)
			continue
		}
		for _, claim := range pvcNamespace.PersistentVolumeClaimNames {
			//check if claim exists
			err := checkForClaim(claim, pvcs)
			if err != nil {
				t.Errorf("Persistant Volume Claim: %v %v", claim, err)
			}

		}
	}
}

func checkForClaim(claim string, pvcs *corev1.PersistentVolumeClaimList) error {

	//Check claim exists and is bound.
	for _, pvc := range pvcs.Items {
		if claim == pvc.Name {
			if pvc.Status.Phase == "Bound" {
				return nil
			}
			return fmt.Errorf("not bound")
		}
	}
	return fmt.Errorf("not found")

}
