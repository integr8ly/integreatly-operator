package resources

import (
	"bytes"
	"context"
	"errors"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultOperatorNamespace = "test"
)

func TestReconcileRHSSOPostgresCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := crov1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
		Status: crov1.PostgresStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultOperatorNamespace,
			},
		},
	}

	testSecretValDatabase := "testDatabase"
	testSecretValExtPort := "5432"
	testSecretValExtHost := "testExtHost"
	testSecretValPassword := "testPassword"
	testSecretValUsername := "testUsername"
	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"database": []byte(testSecretValDatabase),
			"port":     []byte(testSecretValExtPort),
			"host":     []byte(testSecretValExtHost),
			"password": []byte(testSecretValPassword),
			"username": []byte(testSecretValUsername),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		postgresName string
		installation *integreatlyv1alpha1.RHMI
		fakeClient   func() k8sclient.Client
		want         *corev1.Secret
		wantErr      bool
	}{
		{
			name:         "error returned when postgres cannot be provisioned",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres, croPostgresSecret)
				mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("test error")
				}
				return mockClient
			},
			wantErr: true,
		},
		{
			name:         "nil returned when postgres phase is not complete",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				pendingPostgres := croPostgres.DeepCopy()
				pendingPostgres.Status.Phase = croTypes.PhaseInProgress
				return moqclient.NewSigsClientMoqWithScheme(scheme, pendingPostgres, croPostgresSecret)
			},
			want: nil,
		},
		{
			name:         "error returned when postgres credential secret cannot be found",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				return moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres)
			},
			wantErr: true,
		},
		{
			name:         "secret returned on successful reconcile",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				return fake.NewFakeClientWithScheme(scheme, croPostgres, croPostgresSecret)
			},
			want: &corev1.Secret{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      databaseSecretName,
					Namespace: defaultOperatorNamespace,
				},
				Data: map[string][]byte{
					databaseSecretKeyDatabase:  []byte(testSecretValDatabase),
					databaseSecretKeyExtPort:   []byte(testSecretValExtPort),
					databaseSecretKeyExtHost:   []byte(testSecretValExtHost),
					databaseSecretKeyPassword:  []byte(testSecretValPassword),
					databaseSecretKeyUsername:  []byte(testSecretValUsername),
					databaseSecretKeySuperuser: []byte("false"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReconcileRHSSOPostgresCredentials(context.TODO(), tt.installation, tt.fakeClient(), tt.postgresName, defaultOperatorNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileRHSSOPostgresCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil {
				if got != nil {
					t.Errorf("got should be nil, got = %s", got)
				}
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("secret names do not match, got = %s, want %s", got.Name, tt.want.Name)
			}
			for key, val := range tt.want.Data {
				if !bytes.Equal(val, got.Data[key]) {
					t.Errorf("ReconcileRHSSOPostgresCredentials() got = %v, want %v", got.Data, tt.want.Data)
				}
			}
		})
	}
}
