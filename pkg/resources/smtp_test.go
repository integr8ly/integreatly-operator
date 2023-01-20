package resources

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetExistingSMTPFromAddress(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name       string
		FakeClient k8sclient.Client
		WantRes    string
		WantErr    bool
	}{
		{
			Name: "successfully retrieve existing smtp from address",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte("global:\n  smtp_from: noreply-alert@devshift.org"),
				},
			}),
			WantRes: "noreply-alert@devshift.org",
			WantErr: false,
		},
		{
			Name:       "failed to retrieve alert manager config secret",
			FakeClient: utils.NewTestClient(scheme),
			WantRes:    "",
			WantErr:    true,
		},
		{
			Name: "failed to find alertmanager.yaml in alertmanager-application-monitoring secret data",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"fake": []byte("fake:\n test: yes"),
				},
			}),
			WantRes: "",
			WantErr: true,
		},
		{
			Name: "failed to find smtp_from in alert manager config map",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte("global:"),
				},
			}),
			WantRes: "",
			WantErr: true,
		},
		{
			Name: "failed to unmarshal yaml from secret data",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte("invalid yaml"),
				},
			}),
			WantRes: "",
			WantErr: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			smtpFrom, err := GetExistingSMTPFromAddress(context.TODO(), scenario.FakeClient, "test")
			if !scenario.WantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if scenario.WantRes != smtpFrom {
				t.Fatalf("unexpected result from GetExistingSMTPFromAddress(): got %s, want %s", smtpFrom, scenario.WantRes)
			}
		})
	}
}
