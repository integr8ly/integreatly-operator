package common

import (
	goctx "context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// applicable to the rhmi 2 installTypes
func rhmi2PvcNamespaces() []PersistentVolumeClaim {
	return []PersistentVolumeClaim{
		{

			Namespace: NamespacePrefix + "fuse",
			PersistentVolumeClaimNames: []string{
				"syndesis-meta",
				"syndesis-prometheus",
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
}

// common to all installTypes including managed-api
func commonPvcNamespaces() []PersistentVolumeClaim {
	return []PersistentVolumeClaim{
		{

			Namespace: NamespacePrefix + "middleware-monitoring-operator",
			PersistentVolumeClaimNames: []string{
				"prometheus-application-monitoring-db-prometheus-application-monitoring-0",
			},
		},
	}
}

func TestPVClaims(t TestingTB, ctx *TestingContext) {

	//get full list of volume claim items
	pvcs := &corev1.PersistentVolumeClaimList{}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	pvcNamespaces := getPvcNamespaces(rhmi.Spec.Type)

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

func getPvcNamespaces(installType string) []PersistentVolumeClaim {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return commonPvcNamespaces()
	} else {
		return append(commonPvcNamespaces(), rhmi2PvcNamespaces()...)
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
