package common

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterPackageAvailable(t TestingTB, ctx *TestingContext) {

	clusterPackage := &packageOperatorv1alpha1.ClusterPackage{
		ObjectMeta: v1.ObjectMeta{
			Name: "rhoam-config",
		},
	}

	err := ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: clusterPackage.Name}, clusterPackage)
	if err != nil {
		t.Fatalf("failed to get cluster package: %v", err)
	}

	if clusterPackage.Status.Phase != packageOperatorv1alpha1.PackagePhaseAvailable {
		t.Fatalf("cluster package status is not available: %v", err)
	}
}
