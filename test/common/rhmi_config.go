package common

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	goctx "context"
)

const (
	RHMIConfigCRName = "rhmi-config-test"
)

// TestIntegreatlyRoutesExist tests that the routes for all the products are created
func TestRHMIConfigCRs(t *testing.T, ctx *TestingContext) {
	t.Log("Test rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      RHMIConfigCRName,
			Namespace: RHMIOperatorNamespace,
		},
	}

	if err := ctx.Client.Create(goctx.TODO(), rhmiConfig); err != nil {
		t.Fatalf("Failed to create RHMI Config resource %v", err)
	}
	verifyCr(t, ctx)
}

func verifyCr(t *testing.T, ctx *TestingContext) {
	t.Log("Verify rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{}

	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace}, rhmiConfig)
	if err != nil {
		t.Fatalf("Failed to verify RHMI Config resource %v", err)
	}

	// The upgrade fields should default to false
	if rhmiConfig.Spec.Upgrade.AlwaysImmediately != false {
		t.Errorf("AlwaysImmediately should be false by default")
	}
	if rhmiConfig.Spec.Upgrade.DuringNextMaintenance != false {
		t.Errorf("DuringNextMaintenance should be set to false")
	}

	if err := ctx.Client.Delete(goctx.TODO(), rhmiConfig); err != nil {
		t.Errorf("Failed to delete the rhmi config")
	}
}
