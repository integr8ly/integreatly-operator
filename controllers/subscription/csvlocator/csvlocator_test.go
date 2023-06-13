package csvlocator

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type testScenario struct {
	Name string

	Locator     CSVLocator
	InstallPlan *operatorsv1alpha1.InstallPlan

	InitObjs  []runtime.Object
	Assertion func(t *testing.T, err error, csv *operatorsv1alpha1.ClusterServiceVersion)
}

func embeddedScenario(t *testing.T) *testScenario {
	csvString := `{"metadata":{"name":"test-csv","namespace":"test","creationTimestamp":null},"spec":{"install":{"strategy":""},"version":"1.0.0","customresourcedefinitions":{},"apiservicedefinitions":{},"displayName":"","provider":{}},"status":{"lastUpdateTime":null,"lastTransitionTime":null,"certsLastUpdated":null,"certsRotateAt":null}}`

	installPlan := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip",
			Namespace: "test",
		},
		Status: operatorsv1alpha1.InstallPlanStatus{
			Plan: []*operatorsv1alpha1.Step{
				{
					Resource: operatorsv1alpha1.StepResource{
						Kind:     operatorsv1alpha1.ClusterServiceVersionKind,
						Manifest: csvString,
					},
				},
			},
		},
	}

	return &testScenario{
		Name:        "EmbeddedCSVLocator",
		InstallPlan: installPlan,
		Locator:     &EmbeddedCSVLocator{},
		Assertion:   assertCorrectCSV,
	}
}

func binaryDataConfigMapScenario(t *testing.T) *testScenario {
	csvString := `
apiVersion: v1alpha1
kind: ClusterServiceVersion
metadata:
    creationTimestamp: null
    name: test-csv
    namespace: test
spec:
    apiservicedefinitions: {}
    customresourcedefinitions: {}
    displayName: ""
    install:
        strategy: ""
        provider: {}
    version: 1.0.0
status:
    certsLastUpdated: null
    certsRotateAt: null
    lastTransitionTime: null
    lastUpdateTime: null

`

	// compress and encode data
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write([]byte(csvString))
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	compressed := b.Bytes()
	base64text := make([]byte, base64.StdEncoding.EncodedLen(len(compressed)))
	base64.StdEncoding.Encode(base64text, compressed)

	configMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "test",
		},
		BinaryData: map[string][]byte{
			"managed-api-service.clusterserviceversion.yaml": base64text,
		},
	}

	configMapRef := &unpackedBundleReference{
		Namespace: "test",
		Name:      "test-cm",
	}

	configMapRefJSON, err := json.Marshal(configMapRef)
	if err != nil {
		t.Fatalf("failed to marshal config map reference: %v", err)
	}

	installPlan := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip",
			Namespace: "test",
		},
		Status: operatorsv1alpha1.InstallPlanStatus{
			Plan: []*operatorsv1alpha1.Step{
				{
					Resource: operatorsv1alpha1.StepResource{
						Kind:     operatorsv1alpha1.ClusterServiceVersionKind,
						Manifest: string(configMapRefJSON),
					},
				},
			},
		},
	}

	return &testScenario{
		Name: "ConfigMapCSVLocator",
		InitObjs: []runtime.Object{
			configMap,
		},
		InstallPlan: installPlan,
		Locator:     &ConfigMapCSVLocator{},
		Assertion:   assertCorrectCSV,
	}
}

func TestGetCSV(t *testing.T) {
	scenarios := []*testScenario{
		embeddedScenario(t),
		binaryDataConfigMapScenario(t),
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			initObjs := append([]runtime.Object{scenario.InstallPlan}, scenario.InitObjs...)
			client := utils.NewTestClient(scheme, initObjs...)

			csv, err := scenario.Locator.GetCSV(context.TODO(), client, scenario.InstallPlan)
			scenario.Assertion(t, err, csv)
		})
	}
}

func TestCachedCSVLocator(t *testing.T) {
	mockLocator := &mockCSVLocator{
		CSV: &operatorsv1alpha1.ClusterServiceVersion{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-csv",
				Namespace: "test",
			},
		},
		Counter: 0,
	}

	ip1 := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip-1",
			Namespace: "test",
		},
	}

	cached := NewCachedCSVLocator(mockLocator)

	csv, err := cached.GetCSV(context.TODO(), nil, ip1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if csv.Name != "test-csv" {
		t.Errorf("unexpected name for csv. Expected test-csv, got %s", csv.Name)
	}
	if csv.Namespace != "test" {
		t.Errorf("unexpected namespace for csv. Expected test, got %s", csv.Namespace)
	}
	if mockLocator.Counter != 1 {
		t.Errorf("unexpected value for counter. Expected 1, got %d", mockLocator.Counter)
	}

	// Call GetCSV again, the counter should remain the same as the CSV is cached
	_, err = cached.GetCSV(context.TODO(), nil, ip1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockLocator.Counter != 1 {
		t.Errorf("unexpected value for counter. Expected 1, got %d", mockLocator.Counter)
	}
}

func assertCorrectCSV(t *testing.T, err error, csv *operatorsv1alpha1.ClusterServiceVersion) {
	if err != nil {
		t.Fatalf("no error expected, got %v", err)
	}

	if csv == nil {
		t.Fatal("expected csv to be found, but nil was returned")
	}

	if csv.Name != "test-csv" {
		t.Errorf("expected csv name to be test-csv, but got %s", csv.Name)
	}
	if csv.Namespace != "test" {
		t.Errorf("expected csv namespace to be test, but got %s", csv.Namespace)
	}
	if csv.Spec.Version.String() != "1.0.0" {
		t.Errorf("expected csv version to be 1.0.0, got %s", csv.Spec.Version.String())
	}
}

type mockCSVLocator struct {
	CSV     *operatorsv1alpha1.ClusterServiceVersion
	Counter int
}

func (m *mockCSVLocator) GetCSV(_ context.Context, _ k8sclient.Client, _ *operatorsv1alpha1.InstallPlan) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	m.Counter++
	return m.CSV, nil
}
