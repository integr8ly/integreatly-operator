package common

import (
	goctx "context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	noobaav1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	noobaaCrName         = "noobaa-test"
	mcgTestNamespace     = "test-mcg"
	mcgOperatorNamespace = "redhat-rhoam-mcg-operator"
	mcgNoobaaMgmtRoute   = "noobaa-mgmt"
	//mcgS3Route         = "s3"
	//mcgStsRoute        = "sts"
	noobaaAdminSecret = "noobaa-admin"
)

func TestNoobaaViaCR(t TestingTB, ctx *TestingContext) {
	// make project
	project, err := makeProject(ctx, mcgTestNamespace)
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	// get admin token
	accessToken, err := getNoobaaAdminToken(ctx, mcgOperatorNamespace)
	if err != nil {
		t.Fatalf("failed to get admin token:%v", err)
	}

	// get admin url
	route, err := getRoutes(ctx, mcgNoobaaMgmtRoute, mcgOperatorNamespace)
	if err != nil {
		t.Fatalf("failed to get route %v", err)
	}
	adminURL := fmt.Sprintf("https://%v", route.Spec.Host)

	//create secret to be used when creating product
	secret, err := genSecret(ctx, map[string][]byte{
		"adminURL": []byte(adminURL),
		"token":    []byte(*accessToken),
	}, noobaaAdminSecret, mcgTestNamespace)
	if err != nil {
		t.Fatalf("failed to create secret %v", err)
	}

	//create noobaas.noobaa.io cr
	noobaaCR := &noobaav1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      noobaaCrName,
			Namespace: mcgTestNamespace,
		},
		Spec: noobaav1.NooBaaSpec{
			DBType: noobaav1.DBTypes(v1alpha1.DBTypePostgres),
			DefaultBackingStoreSpec: &noobaav1.BackingStoreSpec{
				PVPool: (*noobaav1.PVPoolSpec)(&v1alpha1.PVPoolSpec{
					NumVolumes: 1,
					Secret: corev1.SecretReference{
						Name: noobaaAdminSecret,
					},
					VolumeResources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("100m")},
					},
				}),
			},
		},
	}
	if err := ctx.Client.Create(goctx.TODO(), noobaaCR); err != nil {
		t.Fatalf("failed to create product cr with error: %v", err)
	}

	err = wait.Poll(time.Second*5, time.Minute*2, func() (done bool, err error) {
		err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: noobaaCrName, Namespace: mcgTestNamespace}, noobaaCR)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		//if noobaaCR.Status.Phase == "Ready" {  //TODO, add required fields to CR  to have status (?)
		//	return true, nil
		//}
		if noobaaCR.Spec.DefaultBackingStoreSpec.PVPool.NumVolumes == 1 { //TODO need check status, but it's missing
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed to get Noobaa CR state: %v", err)
	}

	time.Sleep(10 * time.Minute) // TODO - delete

	// delete Noobaa cr
	if err := ctx.Client.Delete(goctx.TODO(), noobaaCR); err != nil {
		t.Fatalf("failed to delete product cr with error: %v", err)
	}

	//delete secret
	if err := ctx.Client.Delete(goctx.TODO(), secret); err != nil {
		t.Fatalf("failed to delete secret with error: %v", err)
	}

	// //delete project
	if err := ctx.Client.Delete(goctx.TODO(), project); err != nil {
		t.Fatalf("failed to delete testing namespace with error: %v", err)
	}
}

func getNoobaaAdminToken(ctx *TestingContext, namespace string) (*string, error) {
	return getToken(ctx, namespace, "ADMIN_ACCESS_TOKEN", "noobaa-admin")
}
