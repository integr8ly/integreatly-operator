package common

import (
	goctx "context"
	"fmt"

	"time"

	threescaleBv1 "github.com/3scale/3scale-operator/pkg/apis/capabilities/v1beta1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	backendName           = "backend-sample"
	systemBackendName     = "backend1"
	backendPrivateBaseURL = "https://api.example.com"
	projectNamespace3     = "test-project3"
)

func Test3scaleBackendViaCR(t TestingTB, ctx *TestingContext) {
	// make project
	project, err := makeProject(ctx, projectNamespace3)
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

	// create secret to be used when creating backend
	secret, err := genSecret(ctx, map[string][]byte{
		"adminURL": []byte(adminURL),
		"token":    []byte(*accessToken),
	}, projectNamespace3)
	if err != nil {
		t.Fatalf("failed to create secret %v", err)
	}

	//create backend cr
	backendCR := &threescaleBv1.Backend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendName,
			Namespace: projectNamespace3,
		},
		Spec: threescaleBv1.BackendSpec{
			Name:           backendName,
			SystemName:     systemBackendName,
			PrivateBaseURL: backendPrivateBaseURL,
			ProviderAccountRef: &corev1.LocalObjectReference{
				Name: projectAdminSecret,
			},
		},
	}
	if err := ctx.Client.Create(goctx.TODO(), backendCR); err != nil {
		t.Fatalf("failed to create backen cr with error: %v", err)
	}

	// setup portaClient
	threescaleClient, err := setupPortaClient(accessToken, route.Spec.Host)
	if err != nil {
		t.Fatalf("failed to setup portaClient: %v", err)
	}

	// verify that product has been created
	err = wait.Poll(time.Second*5, time.Minute*2, func() (done bool, err error) {
		backendList, err := threescaleClient.ListBackendApis()
		if err != nil {
			return false, nil
		}
		for _, backend := range backendList.Backends {
			if backend.Element.Name == backendName {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		t.Fatalf("failed to get backend api: %v", err)
	}

	// delete backend cr
	if err := ctx.Client.Delete(goctx.TODO(), backendCR); err != nil {
		t.Fatalf("failed to delete product cr with error: %v", err)
	}

	//delete secret
	if err := ctx.Client.Delete(goctx.TODO(), secret); err != nil {
		t.Fatalf("Failed to delete secret with error: %v", err)
	}

	// delete project
	if err := ctx.Client.Delete(goctx.TODO(), project); err != nil {
		t.Fatalf("Failed to delete testing namespace with error: %v", err)
	}
}
