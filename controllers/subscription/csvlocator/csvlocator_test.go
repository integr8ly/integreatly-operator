package csvlocator

import (
	"context"
	"encoding/json"
	"testing"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type testScenario struct {
	Name string

	Locator     CSVLocator
	InstallPlan *olmv1alpha1.InstallPlan

	InitObjs  []runtime.Object
	Assertion func(t *testing.T, err error, csv *olmv1alpha1.ClusterServiceVersion)
}

func embeddedScenario(t *testing.T) *testScenario {
	csvString := `{"metadata":{"name":"test-csv","namespace":"test","creationTimestamp":null},"spec":{"install":{"strategy":""},"version":"1.0.0","customresourcedefinitions":{},"apiservicedefinitions":{},"displayName":"","provider":{}},"status":{"lastUpdateTime":null,"lastTransitionTime":null,"certsLastUpdated":null,"certsRotateAt":null}}`

	installPlan := &olmv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip",
			Namespace: "test",
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Plan: []*olmv1alpha1.Step{
				{
					Resource: olmv1alpha1.StepResource{
						Kind:     olmv1alpha1.ClusterServiceVersionKind,
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

func configMapScenario(t *testing.T) *testScenario {
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
    lastUpdateTime: null`

	configMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "test",
		},
		Data: map[string]string{
			"csv.yaml": csvString,
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

	installPlan := &olmv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip",
			Namespace: "test",
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Plan: []*olmv1alpha1.Step{
				{
					Resource: olmv1alpha1.StepResource{
						Kind:     olmv1alpha1.ClusterServiceVersionKind,
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
		configMapScenario(t),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			initObjs := append([]runtime.Object{scenario.InstallPlan}, scenario.InitObjs...)
			client := fake.NewFakeClientWithScheme(buildScheme(), initObjs...)

			csv, err := scenario.Locator.GetCSV(context.TODO(), client, scenario.InstallPlan)
			scenario.Assertion(t, err, csv)
		})
	}
}

func TestCachedCSVLocator(t *testing.T) {
	mockLocator := &mockCSVLocator{
		CSV: &olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-csv",
				Namespace: "test",
			},
		},
		Counter: 0,
	}

	ip1 := &olmv1alpha1.InstallPlan{
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
	csv, err = cached.GetCSV(context.TODO(), nil, ip1)

	if mockLocator.Counter != 1 {
		t.Errorf("unexpected value for counter. Expected 1, got %d", mockLocator.Counter)
	}
}

func TestConditionalCSVLocator(t *testing.T) {
	csv1 := `{"metadata":{"name":"test-csv-1","namespace":"test","creationTimestamp":null},"spec":{"install":{"strategy":""},"version":"1.0.0","customresourcedefinitions":{},"apiservicedefinitions":{},"displayName":"","provider":{}},"status":{"lastUpdateTime":null,"lastTransitionTime":null,"certsLastUpdated":null,"certsRotateAt":null}}`
	csv2 := `
apiVersion: v1alpha1
kind: ClusterServiceVersion
metadata:
    creationTimestamp: null
    name: test-csv-2
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
    lastUpdateTime: null`

	configMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "test",
		},
		Data: map[string]string{
			"csv.yaml": csv2,
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

	installPlan1 := &olmv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip",
			Namespace: "test",
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Plan: []*olmv1alpha1.Step{
				{
					Resource: olmv1alpha1.StepResource{
						Kind:     olmv1alpha1.ClusterServiceVersionKind,
						Manifest: csv1,
					},
				},
			},
		},
	}

	installPlan2 := &olmv1alpha1.InstallPlan{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-ip",
			Namespace: "test",
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Plan: []*olmv1alpha1.Step{
				{
					Resource: olmv1alpha1.StepResource{
						Kind:     olmv1alpha1.ClusterServiceVersionKind,
						Manifest: string(configMapRefJSON),
					},
				},
			},
		},
	}

	client := fake.NewFakeClientWithScheme(buildScheme(), configMap)

	locator := NewConditionalCSVLocator(
		SwitchLocators(
			ForReference,
			ForEmbedded,
		),
	)

	resultCSV1, err1 := locator.GetCSV(context.TODO(), client, installPlan1)
	resultCSV2, err2 := locator.GetCSV(context.TODO(), client, installPlan2)

	if err1 != nil {
		t.Errorf("unexpected error when retrieving CSV 1: %v", err1)
	}
	if err2 != nil {
		t.Errorf("unexpected error when retrieving CSV 2: %v", err2)
	}

	if resultCSV1.Name != "test-csv-1" {
		t.Errorf("unexpected name for csv 1. Expected test-csv-1, got %s", resultCSV1.Name)
	}
	if resultCSV2.Name != "test-csv-2" {
		t.Errorf("unexpected name for csv 2. Expected test-csv-2, got %s", resultCSV2.Name)
	}
}

func assertCorrectCSV(t *testing.T, err error, csv *olmv1alpha1.ClusterServiceVersion) {
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

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	olmv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	return scheme
}

type mockCSVLocator struct {
	CSV     *olmv1alpha1.ClusterServiceVersion
	Counter int
}

func (m *mockCSVLocator) GetCSV(_ context.Context, _ k8sclient.Client, _ *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error) {
	m.Counter++
	return m.CSV, nil
}
