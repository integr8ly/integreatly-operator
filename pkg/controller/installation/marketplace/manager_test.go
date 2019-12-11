package marketplace

import (
	"context"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	coreosv1alpha1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func TestReconcileCatalogSource(t *testing.T) {

	testNameSpace := "test-namespace"

	scenarios := []struct {
		Name       string
		FakeClient client.Client
		Verify     func(csName string, err error, c client.Client)
	}{
		{
			Name:       "Test catalog source created successfully",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme()),
			Verify: func(csName string, err error, c client.Client) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				catalogSource := &coreosv1alpha1.CatalogSource{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: csName, Namespace: testNameSpace}, catalogSource)

				if err != nil && catalogSource.Spec.ConfigMap != csName {
					t.Fatalf("Expected catalog source to be created but wasn't: %v", err)
				}

			},
		},
		{
			Name: "Test catalog source updated successfully",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme(), &coreosv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cs-" + testNameSpace,
					Namespace: testNameSpace,
				},
				Spec: coreosv1alpha1.CatalogSourceSpec{
					ConfigMap: "randomConfigMap",
				},
			}),
			Verify: func(csName string, err error, c client.Client) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				catalogSource := &coreosv1alpha1.CatalogSource{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: csName, Namespace: testNameSpace}, catalogSource)

				if err != nil && catalogSource.Spec.ConfigMap != csName && catalogSource.Spec.ConfigMap != "randomConfigMap" {
					t.Fatalf("Expected catalog source to be updated but wasn't: %v", err)
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
			Verify: func(csName string, err error, c client.Client) {
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
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			Verify: func(csName string, err error, c client.Client) {
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
				UpdateFunc: func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
			Verify: func(csName string, err error, c client.Client) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			catalogSourceName, err := NewManager().reconcileCatalogSource(context.TODO(), scenario.FakeClient, testNameSpace, "testCm")
			scenario.Verify(catalogSourceName, err, scenario.FakeClient)
		})
	}
}

func TestReconcileRegistryConfigMap(t *testing.T) {

	testNameSpace := "test-namespace"

	scenarios := []struct {
		Name        string
		FakeClient  client.Client
		FakeMapData map[string]string
		Verify      func(cmName string, err error, c client.Client, configMapData map[string]string)
	}{
		{
			Name:        "Test registry config map created successfully",
			FakeClient:  fake.NewFakeClientWithScheme(buildScheme()),
			FakeMapData: map[string]string{"test": "someData"},
			Verify: func(cmName string, err error, c client.Client, configMapData map[string]string) {
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				configMap := &corev1.ConfigMap{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: cmName, Namespace: testNameSpace}, configMap)

				if err != nil && !reflect.DeepEqual(configMap.Data, configMapData) {
					t.Fatalf("Expected registry config map to be created with data but wasn't: %v", err)
				}
			},
		},
		{
			Name: "Test registry config map gets updated successfully",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cm-" + testNameSpace,
					Namespace: testNameSpace,
				},
				Data: map[string]string{"test": "outDatedData"},
			}),
			FakeMapData: map[string]string{"test": "someNewData"},
			Verify: func(cmName string, err error, c client.Client, configMapData map[string]string) {
				configMap := &corev1.ConfigMap{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: cmName, Namespace: testNameSpace}, configMap)

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
			Verify: func(cmName string, err error, c client.Client, configMapData map[string]string) {
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
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			Verify: func(cmName string, err error, c client.Client, configMapData map[string]string) {
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
				UpdateFunc: func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
			FakeMapData: map[string]string{"test": "someData"},
			Verify: func(cmName string, err error, c client.Client, configMapData map[string]string) {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			configMapName, err := NewManager().reconcileRegistryConfigMap(context.TODO(), scenario.FakeClient, testNameSpace, scenario.FakeMapData)
			scenario.Verify(configMapName, err, scenario.FakeClient, scenario.FakeMapData)
		})
	}
}
