package datasync

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	templatev1 "github.com/openshift/api/template/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	OperatorNamespace = "integreatly-operator"
)

type DataSyncScenario struct {
	Name           string
	ExpectError    bool
	ExpectedError  string
	ExpectedStatus integreatlyv1alpha1.StatusPhase
	FakeConfig     *config.ConfigReadWriterMock
	FakeClient     k8sclient.Client
	FakeMPM        *marketplace.MarketplaceInterfaceMock
	Installation   *integreatlyv1alpha1.RHMI
	Product        *integreatlyv1alpha1.RHMIProductStatus
	Recorder       record.EventRecorder
}

func getFakeConfig() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return OperatorNamespace
		},
		ReadDataSyncFunc: func() (ready *config.DataSync, e error) {
			return config.NewDataSync(config.ProductConfig{}), nil
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestDataSync(t *testing.T) {
	// Initialize scheme so that types required by the scenarios are available
	scheme := scheme.Scheme
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to initialize scheme: %s", err)
	}

	datasyncServerAppTemplate := &templatev1.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "template.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "datasync-server-app",
			Namespace: datasyncNs,
		},
	}

	cases := []DataSyncScenario{
		{
			Name:           "test error on failed config read",
			ExpectError:    true,
			ExpectedError:  fmt.Sprintf("could not retrieve %[1]s config: could not read %[1]s config", integreatlyv1alpha1.ProductDataSync),
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadDataSyncFunc: func() (ready *config.DataSync, e error) {
					return nil, fmt.Errorf("could not read %s config", integreatlyv1alpha1.ProductDataSync)
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
		},
		{
			Name:           "test successful reconcile when resource already exists",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(datasyncServerAppTemplate),
			FakeConfig:     getFakeConfig(),
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
		},
		{
			Name:           "test successful reconcile",
			ExpectError:    false,
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig:     getFakeConfig(),
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
				tc.Recorder,
				getLogger(),
			)

			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			if err == nil && tc.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", tc.ExpectedError)
			}

			// if we expect errors creating the reconciler, don't try to use it
			if tc.ExpectedError != "" {
				return
			}

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected error but got one: %v", err)
			}

			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductDataSync})
}
