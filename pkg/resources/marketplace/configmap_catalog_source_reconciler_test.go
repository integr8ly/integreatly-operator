package marketplace

import (
	"context"
	"errors"
	"github.com/integr8ly/integreatly-operator/test/utils"
	"reflect"
	"testing"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigMapCatalogSourceReconcilerReconcileCatalogSource(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	testNameSpace := "test-namespace"

	scenarios := []struct {
		Name                     string
		FakeClient               k8sclient.Client
		DesiredConfigMapName     string
		DesiredCatalogSourceName string
		Verify                   func(desiredCSName string, desiredConfigMapName string, res reconcile.Result, err error, c k8sclient.Client)
	}{
		{
			Name:                     "Test catalog source created successfully",
			FakeClient:               fake.NewFakeClientWithScheme(scheme),
			DesiredConfigMapName:     "example-configmap",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredConfigMapName string, res reconcile.Result, err error, c k8sclient.Client) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}
				catalogSource := &coreosv1alpha1.CatalogSource{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: desiredCSName, Namespace: testNameSpace}, catalogSource)
				if err != nil {
					t.Fatalf("Expected catalog source to be created but wasn't: %v", err)
				}
				if catalogSource.Spec.ConfigMap != desiredConfigMapName {
					t.Fatalf("CatalogSource ConfigMap field not reconciled: desired '%s', existing '%s'", desiredConfigMapName, catalogSource.Spec.ConfigMap)
				}

			},
		},
		{
			Name: "Test catalog source updated successfully",
			FakeClient: fake.NewFakeClientWithScheme(scheme, &coreosv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cs-" + testNameSpace,
					Namespace: testNameSpace,
				},
				Spec: coreosv1alpha1.CatalogSourceSpec{
					ConfigMap: "randomConfigMap",
				},
			}),
			DesiredConfigMapName:     "desiredRandomConfigMap",
			DesiredCatalogSourceName: "registry-cs-" + testNameSpace,
			Verify: func(desiredCSName string, desiredConfigMapName string, res reconcile.Result, err error, c k8sclient.Client) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				catalogSource := &coreosv1alpha1.CatalogSource{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: desiredCSName, Namespace: testNameSpace}, catalogSource)

				if err != nil {
					t.Fatalf("Expected catalog source to be updated but wasn't: %v", err)
				}

				if catalogSource.Spec.ConfigMap != desiredConfigMapName {
					t.Fatalf("CatalogSource ConfigMap field not reconciled: desired '%s', existing '%s'", desiredConfigMapName, catalogSource.Spec.ConfigMap)
				}
			},
		},
		{
			Name: "Test catalog source retrieving resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("General error")
				},
			},
			DesiredConfigMapName:     "dummycm",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredConfigMapName string, res reconcile.Result, err error, c k8sclient.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
		{
			Name: "Test catalog source creating resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "catalogsource")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			DesiredConfigMapName:     "dummycm",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredConfigMapName string, res reconcile.Result, err error, c k8sclient.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
		{
			Name: "Test catalog source updating error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return nil
				},
				UpdateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
			DesiredConfigMapName:     "dummycm",
			DesiredCatalogSourceName: "example-catalogsourcename",
			Verify: func(desiredCSName string, desiredConfigMapName string, res reconcile.Result, err error, c k8sclient.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			csReconciler := NewConfigMapCatalogSourceReconciler("example-manifestproduct-dir", scenario.FakeClient, testNameSpace, scenario.DesiredCatalogSourceName)
			res, err := csReconciler.reconcileCatalogSource(context.TODO(), scenario.DesiredConfigMapName)
			scenario.Verify(scenario.DesiredCatalogSourceName, scenario.DesiredConfigMapName, res, err, scenario.FakeClient)
		})
	}
}

func TestConfigMapCatalogSourceReconcilerRegistryConfigMap(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	testNameSpace := "test-namespace"

	scenarios := []struct {
		Name        string
		FakeClient  k8sclient.Client
		FakeMapData map[string]string
		Verify      func(cmName string, err error, c k8sclient.Client, configMapData map[string]string)
	}{
		{
			Name:        "Test registry config map created successfully",
			FakeClient:  fake.NewFakeClientWithScheme(scheme),
			FakeMapData: map[string]string{"test": "someData"},
			Verify: func(cmName string, err error, c k8sclient.Client, configMapData map[string]string) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				configMap := &corev1.ConfigMap{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: cmName, Namespace: testNameSpace}, configMap)

				if err != nil && !reflect.DeepEqual(configMap.Data, configMapData) {
					t.Fatalf("Expected registry config map to be created with data but wasn't: %v", err)
				}
			},
		},
		{
			Name: "Test registry config map gets updated successfully",
			FakeClient: fake.NewFakeClientWithScheme(scheme, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cm-" + testNameSpace,
					Namespace: testNameSpace,
				},
				Data: map[string]string{"test": "outDatedData"},
			}),
			FakeMapData: map[string]string{"test": "someNewData"},
			Verify: func(cmName string, err error, c k8sclient.Client, configMapData map[string]string) {
				configMap := &corev1.ConfigMap{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: cmName, Namespace: testNameSpace}, configMap)

				if err != nil && !reflect.DeepEqual(configMap.Data, configMapData) {
					t.Fatalf("Expected registry config map to be updated with data but wasn't: %v", err)
				}
			},
		},
		{
			Name: "Test registry config map retrieving resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("General error")
				},
			},
			Verify: func(cmName string, err error, c k8sclient.Client, configMapData map[string]string) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
		{
			Name: "Test registry config map creating resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "catalogsource")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			Verify: func(cmName string, err error, c k8sclient.Client, configMapData map[string]string) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
		{
			Name: "Test registry config map updating resource error",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return nil
				},
				UpdateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
			FakeMapData: map[string]string{"test": "someData"},
			Verify: func(cmName string, err error, c k8sclient.Client, configMapData map[string]string) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			csReconciler := NewConfigMapCatalogSourceReconciler("example-manifestproduct-dir", scenario.FakeClient, testNameSpace, "csName")
			desiredConfigMapName, err := csReconciler.reconcileRegistryConfigMap(context.TODO(), scenario.FakeMapData)
			scenario.Verify(desiredConfigMapName, err, scenario.FakeClient, scenario.FakeMapData)
		})
	}
}
