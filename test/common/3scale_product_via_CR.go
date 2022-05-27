package common

import (
	goctx "context"
	"fmt"
	threescaleBv1 "github.com/3scale/3scale-operator/pkg/apis/capabilities/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const (
	productName       = "product-sample"
	adminRoute        = "3scale-admin.apps"
	projectNamespace2 = "test-project2"
)

func Test3scaleProductViaCR(t TestingTB, ctx *TestingContext) {
	// make project
	project, err := makeProject(ctx, projectNamespace2)
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	// get admin token from system seed
	accessToken, err := getAdminToken(ctx, ThreeScaleProductNamespace)
	if err != nil {
		t.Fatalf("failed to get admin token:%v", err)
	}

	// get admin url
	route, err := getRoutes(ctx, adminRoute)
	if err != nil {
		t.Fatalf("failed to get route %v", err)
	}
	adminURL := fmt.Sprintf("https://%v", route.Spec.Host)

	//create secret to be used when creating product
	secret, err := genSecret(ctx, map[string][]byte{
		"adminURL": []byte(adminURL),
		"token":    []byte(*accessToken),
	}, projectNamespace2)
	if err != nil {
		t.Fatalf("failed to create secret %v", err)
	}

	//create product cr
	productCR := &threescaleBv1.Product{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectName,
			Namespace: projectNamespace2,
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
	err = wait.Poll(time.Second*5, time.Minute*2, func() (done bool, err error) {
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

	// //delete project
	if err := ctx.Client.Delete(goctx.TODO(), project); err != nil {
		t.Fatalf("failed to delete testing namespace with error: %v", err)
	}
}
