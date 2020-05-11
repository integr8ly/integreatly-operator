package common

import (
	"fmt"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

func TestIntegreatlyStagesStatus(t *testing.T, ctx *TestingContext) {

	//get RHMI
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	//iterate stages and check their status
	// This does not check the pro
	for _, stage := range rhmi.Status.Stages {
		err := checkStatus(stage)
		if err != nil {
			t.Errorf("Error: Stage %v not completed. It's current status is %v", stage.Name, err)
		}
	}
}

func checkStatus(stage v1alpha1.RHMIStageStatus) error {
	//check if stage is completed or return and error
	if stage.Phase == "completed" {
		return nil
	}
	return fmt.Errorf("%v", stage.Phase)
}
