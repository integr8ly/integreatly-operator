package osde2e

import (
	goctx "context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/test/common"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	OSD_E2E_PRE_TESTS = []common.TestCase{
		{Description: "Managed-API pre-test", Test: PreTest},
	}
)

//PreTest This tests if an installation of Managed-API was finished and is successful
func PreTest(t *testing.T, ctx *common.TestingContext) {
	err := wait.Poll(time.Second*15, time.Minute*80, func() (done bool, err error) {

		rhmi, err := getRHMI(ctx.Client)
		if err != nil {
			t.Fatalf("error getting RHMI CR: %v", err)
		}

		// Patch Managed-API CR with cluster storage
		if rhmi.Spec.UseClusterStorage != "true" {
			rhmiCR := fmt.Sprintf(`{
					"apiVersion": "integreatly.org/v1alpha1",
					"kind": "RHMI",
					"spec": {
						"useClusterStorage" : "true"
					}
				}`)

			rhmiCRBytes := []byte(rhmiCR)

			request := ctx.ExtensionClient.RESTClient().Patch(types.MergePatchType).
				Resource("rhmis").
				Name("rhoam").
				Namespace(common.RHMIOperatorNamespace).
				RequestURI("/apis/integreatly.org/v1alpha1").Body(rhmiCRBytes).Do()
			_, err := request.Raw()

			if err != nil {
				return false, err
			}
		}

		if rhmi.Status.Stage != "complete" {
			t.Logf("Current stage is %v", rhmi.Status.Stage)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		t.Errorf("Installation was not successful after 80minutes ...%v", err)
	}
}

func getRHMI(client dynclient.Client) (*integreatlyv1alpha1.RHMI, error) {
	rhmi := &integreatlyv1alpha1.RHMI{}
	watchNS, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "could not get watch namespace from getRHMI function")
	}
	nsSegments := strings.Split(watchNS, "-")
	crName := nsSegments[1]
	if err := client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: common.RHMIOperatorNamespace}, rhmi); err != nil {
		return nil, fmt.Errorf("error getting RHMI CR: %w", err)
	}
	return rhmi, nil
}
