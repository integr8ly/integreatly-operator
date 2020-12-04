package osde2e

import (
	goctx "context"
	"github.com/integr8ly/integreatly-operator/test/common"
	"k8s.io/apimachinery/pkg/util/wait"
	"testing"
	"time"
)

var (
	UNINSTALL = []common.TestCase{
		{Description: "Managed-API uninstall", Test: Uninstall},
	}
)

//Uninstall stage is triggered at the end of e2e tests, whether the tests were successful or not
func Uninstall(t *testing.T, ctx *common.TestingContext) {
	err := wait.Poll(time.Second*15, time.Minute*40, func() (done bool, err error) {

		// Get RHMI CR - if getRHMI fails, we assume that RHOAM has been deleted
		rhmi, err := getRHMI(ctx.Client)
		if err != nil {
			t.Log("Uninstall Completed")
			return true, nil
		}

		// Apply deletetion timestamp
		if rhmi.DeletionTimestamp == nil {
			err := ctx.Client.Delete(goctx.TODO(), rhmi)
			if err != nil {
				t.Logf("Could not delete RHOAM CR due to %v, retryting...", err)
				return false, nil
			}
			t.Log("Successfully marked RHOAM CR for deletion")
			return false, nil
		}

		t.Logf("Delete of RHOAM in progress - current stage is %v", rhmi.Status.Stage)
		return false, nil
	})

	if err != nil {
		t.Errorf("Could not delete RHOAM CR - Uninstall took longer than 40 minutes: %v", err)
	}
}
