package osde2e

import (
	"context"
	goctx "context"
	"fmt"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/test/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	OSD_E2E_PRE_TESTS = []common.TestCase{
		{Description: "Integreatly Operator pre-test", Test: PreTest},
	}
	pagerDutySecretName = "pagerduty"
	deadMansSnitchName  = "deadmanssnitch"
	smtpSecretName      = "smtp"
	resourceName        string
)

// PreTest This tests if an installation of Managed-API or RHMI was finished and is successful
func PreTest(t common.TestingTB, ctx *common.TestingContext) {
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*15, time.Minute*70, false, func(ctx2 context.Context) (done bool, err error) {
		rhmi, err := getRHMI(ctx.Client)
		if err != nil {
			t.Fatalf("error getting RHMI CR: %v", err)
		}

		if resourceName == "" {
			resourceName = "rhoam"
		}

		// Patch RHMI CR CR with cluster storage
		if rhmi.Spec.UseClusterStorage == "false" || rhmi.Spec.UseClusterStorage == "" {
			rhmiCR := `{
				"apiVersion": "integreatly.org/v1alpha1",
				"kind": "RHMI",
				"spec": {
					"useClusterStorage" : "true"
				}
			}`

			rhmiCRBytes := []byte(rhmiCR)

			request := ctx.ExtensionClient.RESTClient().Patch(types.MergePatchType).
				Resource("rhmis").
				Name(resourceName).
				Namespace(common.RHOAMOperatorNamespace).
				RequestURI("/apis/integreatly.org/v1alpha1").Body(rhmiCRBytes).Do(context.TODO())
			_, err := request.Raw()

			if err != nil {
				return false, err
			}
		}

		// Get smtp secret - if failed - create SMTP Secret
		_, err = getSecret(ctx.Client, common.NamespacePrefix+smtpSecretName)
		if err != nil {
			smtpSec := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprint(common.NamespacePrefix + smtpSecretName),
					Namespace: common.RHOAMOperatorNamespace,
				},
				Data: map[string][]byte{
					"host":     []byte("test"),
					"password": []byte("test"),
					"port":     []byte("test"),
					"tls":      []byte("test"),
					"username": []byte("test"),
				},
			}
			if err := ctx.Client.Create(goctx.TODO(), smtpSec.DeepCopy()); err != nil {
				t.Fatalf("Failed to create Pager Duty Secret: %v", err)
			}
		}

		// Get pagerduty secret - if failed - create
		_, err = getSecret(ctx.Client, common.NamespacePrefix+pagerDutySecretName)
		if err != nil {

			pagerDuty := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.NamespacePrefix + pagerDutySecretName,
					Namespace: common.RHOAMOperatorNamespace,
				},
				Data: map[string][]byte{
					"serviceKey": []byte("test"),
				},
			}
			if err := ctx.Client.Create(goctx.TODO(), pagerDuty.DeepCopy()); err != nil {
				t.Fatalf("Failed to create Pager Duty Secret: %v", err)
			}
		}

		// Get dms - if failed - create
		_, err = getSecret(ctx.Client, common.NamespacePrefix+deadMansSnitchName)
		if err != nil {

			dms := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.NamespacePrefix + deadMansSnitchName,
					Namespace: common.RHOAMOperatorNamespace,
				},
				Data: map[string][]byte{
					"url": []byte("https://dms.example.com"),
				},
			}
			if err := ctx.Client.Create(goctx.TODO(), dms.DeepCopy()); err != nil {
				t.Fatalf("Failed to create DMS secret: %v", err)
			}
		}

		if rhmi.Status.Stage != "complete" {
			t.Logf("Current stage is %v", rhmi.Status.Stage)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		t.Errorf("Something went wrong ...%v", err)
	}
}

func getRHMI(client dynclient.Client) (*integreatlyv1alpha1.RHMI, error) {
	rhmi := &integreatlyv1alpha1.RHMI{}
	watchNS := common.GetNamespacePrefix()
	nsSegments := strings.Split(watchNS, "-")
	crName := nsSegments[1]
	if err := client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: common.RHOAMOperatorNamespace}, rhmi); err != nil {
		return nil, fmt.Errorf("error getting RHMI CR: %w", err)
	}
	return rhmi, nil
}

func getSecret(client dynclient.Client, secretName string) (*v1.Secret, error) {
	secret := &v1.Secret{}

	if err := client.Get(goctx.TODO(), types.NamespacedName{Name: secretName, Namespace: common.RHOAMOperatorNamespace}, secret); err != nil {
		return nil, fmt.Errorf("Error getting secret")
	}
	return secret, nil
}
