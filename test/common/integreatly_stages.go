package common

import (
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

var (
	expectedStageProducts = map[string][]string{
		"authentication": {
			"rhsso",
		},

		"bootstrap": {},

		"cloud-resources": {
			"cloud-resources",
		},

		"monitoring": {
			"middleware-monitoring",
		},

		"products": {
			"fuse",
			"rhssouser",
			"datasync",
			"codeready-workspaces",
			"fuse-on-openshift",
			"3scale",
			"amqonline",
			"ups",
			"apicurito",
		},

		"solution-explorer": {
			"solution-explorer",
		},
	}
)

func TestIntegreatlyStagesStatus(t *testing.T, ctx *TestingContext) {

	//get RHMI
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	//iterate stages and check their status
	for stageName, productNames := range expectedStageProducts {
		stage, ok := rhmi.Status.Stages[v1alpha1.StageName(stageName)]
		if !ok {
			t.Errorf("Error checking stage %s. Not found", stageName)
			continue
		}

		if status := checkStageStatus(stage); status != "" {
			t.Errorf("Error: Stage %v not completed. It's current status is %v", stage.Name, status)
		}

		for _, productName := range productNames {
			product, ok := stage.Products[v1alpha1.ProductName(productName)]
			if !ok {
				t.Errorf("Product %s not found in stage %s", productName, stageName)
				continue
			}

			if status := checkProductStatus(product); status != "" {
				t.Errorf("Error: Product %s status not completed. It's current status is %s", productName, status)
			}
		}
	}
}

func checkStageStatus(stage v1alpha1.RHMIStageStatus) string {
	return checkStatus(stage.Phase)
}

func checkProductStatus(product v1alpha1.RHMIProductStatus) string {
	return checkStatus(product.Status)
}

// checkStatus verifies that the status is complete. If it is, returns the empty
// string. If it's not, returns the invalid status as a string
func checkStatus(status v1alpha1.StatusPhase) string {
	//check if status is completed or return and error
	if status == "completed" {
		return ""
	}

	return string(status)
}
