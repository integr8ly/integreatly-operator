package marketplace

import (
	"context"
	"errors"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/test/utils"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

func TestGRPCImageCatalogSourceReconcilerReconcile(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	testNameSpace := "test-namespace"

	scenarios := []struct {
		Name                     string
		FakeClient               k8sclient.Client
		DesiredGRPCImage         string
		DesiredCatalogSourceName string
		Verify                   func(desiredCSName string, desiredGRPCImage string, res reconcile.Result, err error, c k8sclient.Client)
	}{
		{
			Name:                     "Test catalog source created successfully",
			FakeClient:               utils.NewTestClient(scheme),
			DesiredGRPCImage:         "example-grpcimage",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredGRPCImage string, res reconcile.Result, err error, c k8sclient.Client) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}
				catalogSource := &coreosv1alpha1.CatalogSource{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: desiredCSName, Namespace: testNameSpace}, catalogSource)
				if err != nil {
					t.Fatalf("Expected catalog source to be created but wasn't: %v", err)
				}
				if catalogSource.Spec.Image != desiredGRPCImage {
					t.Fatalf("CatalogSource Image field not reconciled: desired '%s', existing '%s'", desiredGRPCImage, catalogSource.Spec.Image)
				}
				if catalogSource.Spec.SourceType != coreosv1alpha1.SourceTypeGrpc {
					t.Fatalf("CatalogSoure type is not of type '%s'", coreosv1alpha1.SourceTypeGrpc)
				}
				if catalogSource.Spec.Address != "" {
					t.Fatalf("CatalogSoure type 'address' attribute set")
				}
				if catalogSource.Spec.ConfigMap != "" {
					t.Fatalf("Unexpected CatalogSource 'configMap' attribute set")
				}
			},
		},
		{
			Name: "Test catalog source updated successfully",
			FakeClient: utils.NewTestClient(scheme, &coreosv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cs-" + testNameSpace,
					Namespace: testNameSpace,
				},
				Spec: coreosv1alpha1.CatalogSourceSpec{
					Image: "randomGRPCImage",
				},
			}),
			DesiredGRPCImage:         "desiredRandomGRPCImage",
			DesiredCatalogSourceName: "registry-cs-" + testNameSpace,
			Verify: func(desiredCSName string, desiredGRPCImage string, res reconcile.Result, err error, c k8sclient.Client) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				catalogSource := &coreosv1alpha1.CatalogSource{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: desiredCSName, Namespace: testNameSpace}, catalogSource)

				if err != nil {
					t.Fatalf("Expected catalog source to be updated but wasn't: %v", err)
				}

				if catalogSource.Spec.Image != desiredGRPCImage {
					t.Fatalf("CatalogSource Image field not reconciled: desired '%s', existing '%s'", desiredGRPCImage, catalogSource.Spec.Image)
				}
			},
		},
		{
			Name: "Test catalog source retrieving resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object) error {
					return errors.New("General error")
				},
			},
			DesiredGRPCImage:         "dummygrpcimage",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredGRPCImage string, res reconcile.Result, err error, c k8sclient.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
		{
			Name: "Test catalog source creating resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "catalogsource")
				},
				CreateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			DesiredGRPCImage:         "dummygrpcimage",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredGRPCImage string, res reconcile.Result, err error, c k8sclient.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
		{
			Name: "Test catalog source updating error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object) error {
					return nil
				},
				UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
			DesiredGRPCImage:         "dummygrpcimage",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredGRPCImage string, res reconcile.Result, err error, c k8sclient.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			csReconciler := NewGRPCImageCatalogSourceReconciler(scenario.DesiredGRPCImage, scenario.FakeClient, testNameSpace, scenario.DesiredCatalogSourceName, l.NewLogger())
			res, err := csReconciler.Reconcile(context.TODO(), "test")
			scenario.Verify(scenario.DesiredCatalogSourceName, scenario.DesiredGRPCImage, res, err, scenario.FakeClient)
		})
	}
}
