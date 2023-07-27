package common

import (
	"github.com/integr8ly/integreatly-operator/pkg/products/obo"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
)

func TestClusterPackageAvailable(t TestingTB, ctx *TestingContext) {

	pkg, err := obo.GetOboClusterPackage(ctx.Client)
	if err != nil {
		t.Errorf("failed to get ClusterPackage: %w", err)
	}

	if pkg.Status.Phase != packageOperatorv1alpha1.PackagePhaseAvailable {
		t.Errorf("error cluster package state is not phase available, current phase: %s", pkg.Status.Phase)
	}
}
