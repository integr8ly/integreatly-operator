package common

import (
	"context"
	"fmt"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sku"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSKUValues(t TestingTB, ctx *TestingContext) {

	//verify the config map is in place
	_, err := getSKUConfigMap(ctx.Client)
	if err != nil {
		t.Fatal("failed to get sku config map", err)
		// for now if the skuconfig map is not found we can return as we don't want to exercise the full test
		return
	}

	// if no skuparam is found then set isdefault to true.
	defaultSKU := false
	skuName, found, err := addon.GetStringParameterByInstallType(context.TODO(), ctx.Client, rhmiv1alpha1.InstallationTypeManagedApi, RHMIOperatorNamespace, "sku")
	if !found {
		defaultSKU = true
	}

	installation, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatal("couldn't get RHMI cr for sku test")
	}

	//verify that the TOSKU value is set and that SKU is not set
	//assuming this is run after installation
	if installation.Status.SKU == "" {
		t.Fatal("SKU status not set after installation")
	}
	if installation.Status.ToSKU != "" {
		t.Fatal("toSKU status set after installation")
	}

	//verify the sku value is as expected depending on whether there is a default used or not.
	if !defaultSKU {
		if installation.Status.SKU != skuName {
			t.Fatal(fmt.Sprintf("sku value set as '%s' but doesn't match the expected value: '%s'", installation.Status.SKU, skuName))
		}
	}

}

func getSKUConfigMap(c k8sclient.Client) (*v1.ConfigMap, error) {
	skuConfigMap := &v1.ConfigMap{}
	if err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: sku.ConfigMapName, Namespace: RHMIOperatorNamespace}, skuConfigMap); err != nil {
		return nil, fmt.Errorf("failed to get SKU config map: '%w'", err)
	}
	return skuConfigMap, nil
}
