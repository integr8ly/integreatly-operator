package common

import (
	"github.com/integr8ly/integreatly-operator/pkg/products/obo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
)

func TestClusterPackageAvailable(t TestingTB, ctx *TestingContext) {

	pkg, err := obo.GetOboClusterPackage(ctx.Client)
	if err != nil {
		t.Errorf("failed to get ClusterPackage: %w", err)
	}

	isAvailable := false
	for _, condition := range pkg.Status.Conditions {
		if condition.Type == packageOperatorv1alpha1.PackageAvailable && condition.Status == metav1.ConditionTrue {
			isAvailable = true
			break
		}
	}

	if !isAvailable {
		t.Errorf("error cluster package is not available, conditions: %+v", pkg.Status.Conditions)
	}
}
