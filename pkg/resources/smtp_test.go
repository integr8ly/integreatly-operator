package resources

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetExistingSMTPFromAddress(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}

	scenarios := []struct {
		Name       string
		FakeClient k8sclient.Client
		WantErr    bool
	}{
		{
			Name: "successfully retrieve existing smtp from address",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte(`
global:
  smtp_from: test
`),
				},
			}, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
			}),
		},
		{
			Name:       "failed to retrieve alert manager config secret",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme),
			WantErr:    true,
		},
		{
			Name: "failed to unmarshal yaml from secret data",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte(`invalid yaml`),
				},
			}, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
			}),
			WantErr: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			_, err := GetExistingSMTPFromAddress(context.TODO(), scenario.FakeClient, "test")
			if !scenario.WantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
