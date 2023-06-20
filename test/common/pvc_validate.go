package common

import (
	goctx "context"
	"fmt"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/resources/cluster"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	noobaaDefaultBackingStorePvc = "noobaa-default-backing-store-noobaa-pvc"
	dbNoobaaDbPgPvc              = "db-noobaa-db-pg-0"
)

// common to all installTypes including managed-api
func commonPvcNamespaces(ctx *TestingContext) []PersistentVolumeClaim {
	var pvc []PersistentVolumeClaim

	if platformType, err := cluster.GetPlatformType(goctx.TODO(), ctx.Client); err != nil && platformType == configv1.GCPPlatformType {
		pvc = append(pvc, []PersistentVolumeClaim{
			{
				Namespace: McgOperatorNamespace,
				PersistentVolumeClaimNames: []string{
					dbNoobaaDbPgPvc,
					noobaaDefaultBackingStorePvc,
				},
			},
		}...)
	}
	return pvc
}

func TestPVClaims(t TestingTB, ctx *TestingContext) {

	//get full list of volume claim items
	pvcs := &corev1.PersistentVolumeClaimList{}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	pvcNamespaces := getPvcNamespaces(ctx, rhmi.Spec.Type)
	if pvcNamespaces == nil {
		t.Skip("No PVC's listed to test for")
	}

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

func getPvcNamespaces(ctx *TestingContext, installType string) []PersistentVolumeClaim {
	return commonPvcNamespaces(ctx)
}

func checkForClaim(claim string, pvcs *corev1.PersistentVolumeClaimList) error {

	//Check claim exists and is bound.
	for _, pvc := range pvcs.Items {
		pvcName := pvc.Name
		if strings.HasPrefix(pvcName, noobaaDefaultBackingStorePvc) {
			pvcName = noobaaDefaultBackingStorePvc
		}
		if claim == pvcName {
			if pvc.Status.Phase == "Bound" {
				return nil
			}
			return fmt.Errorf("not bound")
		}
	}
	return fmt.Errorf("not found")

}
