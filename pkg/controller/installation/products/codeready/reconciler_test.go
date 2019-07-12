package codeready

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	k8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCodeready(t *testing.T) {
	scenarios := []struct {
		Name                 string
		ExpectedStatus       v1alpha1.StatusPhase
		ExpectedError        string
		ExpectedCreateError  string
		Object               *v1alpha1.Installation
		FakeConfig           *config.ConfigReadWriterMock
		FakeK8sClient        *k8sclient.Clientset
		FakeControllerClient client.Client
	}{
		{
			Name:                 "test no phase without errors",
			ExpectedStatus:       v1alpha1.PhaseCreatingSubscription,
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return config.NewCodeReady(config.ProductConfig{}), nil
				},
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
					}), nil
				},
			},
		},
		{
			Name:                 "test error on bad codeready config",
			ExpectedStatus:       v1alpha1.PhaseNone,
			ExpectedCreateError:  "could not retrieve che config: could not load codeready config",
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return nil, errors.New("could not load codeready config")
				},
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
					}), nil
				},
			},
		},
	}

	for _, scenario := range scenarios {
		logger := logrus.WithFields(logrus.Fields{"product": string("codeready")})
		testReconciler, err := NewReconciler(
			scenario.FakeControllerClient,
			&rest.Config{},
			scenario.FakeK8sClient,
			scenario.FakeConfig,
			scenario.Object,
			logger,
		)
		if err != nil && err.Error() != scenario.ExpectedCreateError {
			t.Fatalf("unexpected error creating reconciler: '%v', expected: '%v'", err, scenario.ExpectedCreateError)
		}

		if err == nil && scenario.ExpectedCreateError != "" {
			t.Fatalf("expected error '%v' and got nil", scenario.ExpectedCreateError)
		}

		// if we expect errors creating the reconciler, don't try to use it
		if scenario.ExpectedCreateError != "" {
			continue
		}

		status, err := testReconciler.Reconcile(scenario.Object)
		if err != nil && err.Error() != scenario.ExpectedError {
			t.Fatalf("unexpected error: %v, expected: %v", err, scenario.ExpectedError)
		}

		if err == nil && scenario.ExpectedError != "" {
			t.Fatalf("expected error '%v' and got nil", scenario.ExpectedError)
		}

		if status != scenario.ExpectedStatus {
			t.Fatalf("Expected status: '%v', got: '%v'", scenario.ExpectedStatus, status)
		}
	}
}
