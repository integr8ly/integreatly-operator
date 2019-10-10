package resources

import (
	"bytes"
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestGetDefaultPullSecret(t *testing.T) {
	defPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultOriginPullSecretName,
			Namespace: DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}

	scenarios := []struct {
		Name       string
		FakeClient client.Client
		Verify     func(secret corev1.Secret, err error, t *testing.T)
	}{
		{
			Name:       "Test Default Pull Secret is successfully retrieved",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, defPullSecret),
			Verify: func(secret corev1.Secret, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if bytes.Compare(secret.Data["test"], defPullSecret.Data["test"]) != 0 {
					t.Fatalf("expected data %v, but got %v", defPullSecret.Data["test"], secret.Data["test"])
				}
			},
		},
		{
			Name:       "Test Get Default Pull Secret error",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme),
			Verify: func(secret corev1.Secret, err error, t *testing.T) {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {

			res, err := GetDefaultPullSecret(scenario.FakeClient, context.TODO())
			scenario.Verify(res, err, t)
		})
	}
}

func TestCopyDefaultPullSecretToNameSpace(t *testing.T) {
	defPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultOriginPullSecretName,
			Namespace: DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}

	scenarios := []struct {
		Name       string
		FakeClient client.Client
		Verify     func(client client.Client, err error, t *testing.T)
	}{
		{
			Name: "Test Default Pull Secret is successfully copied over to target namespace",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, defPullSecret, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-namespace",
					Labels:    map[string]string{"webapp": "true"},
				},
			}),
			Verify: func(c client.Client, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				s := &corev1.Secret{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: "new-name-of-secret", Namespace: "test-namespace"}, s)

				if bytes.Compare(s.Data["test"], defPullSecret.Data["test"]) != 0 {
					t.Fatalf("expected data %v, but got %v", defPullSecret.Data["test"], s.Data["test"])
				}
			},
		},
		{
			Name: "Test Get Default Pull Secret error when trying to copy",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-namespace",
					Labels:    map[string]string{"webapp": "true"},
				},
			}),
			Verify: func(c client.Client, err error, t *testing.T) {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := CopyDefaultPullSecretToNameSpace("test-namespace", "new-name-of-secret", scenario.FakeClient, context.TODO())
			scenario.Verify(scenario.FakeClient, err, t)
		})
	}
}
