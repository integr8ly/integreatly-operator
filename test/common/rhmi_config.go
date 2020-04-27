package common

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	goctx "context"
)

const (
	RHMIConfigCRName  = "rhmi-config"
	OperatorNamespace = "redhat-rhmi-operator"
)

// TestIntegreatlyRoutesExist tests that the routes for all the products are created
func TestRHMIConfigCRs(t *testing.T, ctx *TestingContext) {
	t.Log("Test rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      RHMIConfigCRName,
			Namespace: OperatorNamespace,
		},
		Spec: v1alpha1.RHMIConfigSpec{
			//Upgrade: v1alpha1.Upgrade{
			//	//Contacts: "asdasd",
			//	//ApplyOn:  "asdasd",
			//},
		},
	}

	if err := ctx.Client.Create(goctx.TODO(), rhmiConfig); err != nil {
		if err != nil {
			t.Fatalf("Failed to create RHMI Config resource %v", err)
			return
		}
	}
	verifyCr(t, ctx)
}

func verifyCr(t *testing.T, ctx *TestingContext) {
	t.Log("Verify rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{}
	// get the RHMI custom resource to check what storage type is being used
	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: RHMIConfigCRName, Namespace: OperatorNamespace}, rhmiConfig)
	if err != nil {
		t.Fatalf("Failed to verify RHMI Config resource %v", err)
	}

	// The CR we created had no values set so the upgrade fields should default to false.
	if rhmiConfig.Spec.Upgrade.AlwaysImmediately != false {
		t.Errorf("AlwaysImmediately should be false by default")
	}
	if rhmiConfig.Spec.Upgrade.DuringNextMaintenance != false {
		t.Errorf("DuringNextMaintenance should be set to false")
	}
}
