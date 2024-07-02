package common

import (
	goctx "context"
	"fmt"
	"time"

	threescaleBv1 "github.com/3scale/3scale-operator/apis/capabilities/v1beta1"
	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	productName = "product-sample"
	adminRoute  = "3scale-admin.apps"
)

func Test3scaleProductViaCR(t TestingTB, ctx *TestingContext) {

	// poll to make sure the project is deleted from H30 and H29 before attempting to create again
	project := &projectv1.Project{}
	err := wait.PollUntilContextTimeout(goctx.TODO(), tenantCreatedLoopTimeout, tenantCreatedTimeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: projectNamespace, Namespace: projectNamespace}, project)
		if err != nil {
			return true, nil
		}
		return false, err

	})
	if err != nil {
		t.Logf("failed to check project %v", err)
	}
	// make project
	project, err = makeProject(ctx, projectNamespace)
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	// get admin token from system seed
	accessToken, err := getAdminToken(ctx, ThreeScaleProductNamespace)
	if err != nil {
		t.Fatalf("failed to get admin token:%v", err)
	}

	// get admin url
	route, err := getRoutes(ctx, adminRoute, ThreeScaleProductNamespace)
	if err != nil {
		t.Fatalf("failed to get route %v", err)
	}
	adminURL := fmt.Sprintf("https://%v", route.Spec.Host)

	//create secret to be used when creating product
	secret, err := createSecret(ctx, map[string][]byte{
		"adminURL": []byte(adminURL),
		"token":    []byte(*accessToken),
	}, projectAdminSecret, projectNamespace)
	if err != nil {
		t.Fatalf("failed to create secret %v", err)
	}

	//create product cr
	productCR := &threescaleBv1.Product{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectName,
			Namespace: projectNamespace,
			Annotations: map[string]string{
				"insecure_skip_verify": "true",
			},
		},
		Spec: threescaleBv1.ProductSpec{
			Name: productName,
			ProviderAccountRef: &corev1.LocalObjectReference{
				Name: projectAdminSecret,
			},
		},
	}
	if err := ctx.Client.Create(goctx.TODO(), productCR); err != nil {
		t.Fatalf("failed to create product cr with error: %v", err)
	}

	// setup portaClient
	threescaleClient, err := setupPortaClient(accessToken, route.Spec.Host)
	if err != nil {
		t.Fatalf("failed to setup portaClient: %v", err)
	}

	// verify that product has been created
	err = wait.PollUntilContextTimeout(goctx.TODO(), time.Second*5, time.Minute*2, false, func(ctx2 goctx.Context) (done bool, err error) {
		productList, err := threescaleClient.ListProducts()
		if err != nil {
			return false, nil
		}
		for _, product := range productList.Products {
			if product.Element.Name == productName {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		t.Fatalf("failed to get product name: %v", err)
	}

	// //delete product cr
	if err := ctx.Client.Delete(goctx.TODO(), productCR); err != nil {
		t.Fatalf("failed to delete product cr with error: %v", err)
	}

	//delete secret
	if err := ctx.Client.Delete(goctx.TODO(), secret); err != nil {
		t.Fatalf("failed to delete secret with error: %v", err)
	}

	//delete project
	if err := ctx.Client.Delete(goctx.TODO(), project); err != nil {
		t.Fatalf("failed to delete testing namespace with error: %v", err)
	}
}
