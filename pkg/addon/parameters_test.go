package addon

import (
	"context"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetParameter(t *testing.T) {
	scenarios := []struct {
		Name               string
		ExistingParameters map[string][]byte
		Parameter          string
		ExpectedFound      bool
		ExpectedValue      []byte
	}{
		{
			Name: "Parameter found",
			ExistingParameters: map[string][]byte{
				"test": []byte("foo"),
			},
			ExpectedFound: true,
			ExpectedValue: []byte("foo"),
			Parameter:     "test",
		},
		{
			Name: "Parameter not found: not in secret",
			ExistingParameters: map[string][]byte{
				"test": []byte("foo"),
			},
			ExpectedFound: false,
			Parameter:     "bar",
		},
		{
			Name:               "Parameter not found: secret not defined",
			ExistingParameters: nil,
			ExpectedFound:      false,
			Parameter:          "test",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			corev1.AddToScheme(scheme)
			integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)

			initObjs := []runtime.Object{
				&integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "redhat-test-operator",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
			}

			if scenario.ExistingParameters != nil {
				initObjs = append(initObjs, &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "addon-managed-api-service-parameters",
						Namespace: "redhat-test-operator",
					},
					Data: scenario.ExistingParameters,
				})
			}

			client := fake.NewFakeClientWithScheme(scheme, initObjs...)

			result, ok, err := GetParameter(context.TODO(), client, "redhat-test-operator", scenario.Parameter)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			if ok != scenario.ExpectedFound {
				t.Errorf("unexpected found value. Expected %t, got %t", scenario.ExpectedFound, ok)
				return
			}

			if string(result) != string(scenario.ExpectedValue) {
				t.Errorf("unexpected parameter value. Expected %s, got %s",
					string(scenario.ExpectedValue), string(result))
			}
		})
	}
}
