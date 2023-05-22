package resources

import (
	"context"
	"errors"

	"github.com/integr8ly/integreatly-operator/utils"

	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultOperatorNamespace = "test"
	defaultRHSSONamespace    = "test"
)

func TestReconcileRHSSOPostgresCredentials(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
	}
	secretRef := &croTypes.SecretRef{
		Name:      "test",
		Namespace: defaultOperatorNamespace,
	}
	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
		Status: croTypes.ResourceTypeStatus{
			Phase:     croTypes.PhaseComplete,
			SecretRef: secretRef,
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
		want         *crov1.Postgres
		wantErr      bool
	}{
		{
			name:         "error returned when postgres cannot be provisioned",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres)
				mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
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
				return moqclient.NewSigsClientMoqWithScheme(scheme, pendingPostgres)
			},
			want: nil,
		},
		{
			name:         "error returned when postgres credential secret cannot be found",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres)
				mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return errors.New("test error")
				}
				return mockClient
			},
			wantErr: true,
		},
		{
			name:         "postgres with expected config returned on successful reconcile",
			postgresName: "test",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres, croPostgresSecret)
				return mockClient
			},
			want: &crov1.Postgres{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: defaultOperatorNamespace,
				},
				Spec: croTypes.ResourceTypeSpec{
					Tier:      "production",
					SecretRef: secretRef,
				},
				Status: croTypes.ResourceTypeStatus{
					Phase:     croTypes.PhaseComplete,
					SecretRef: secretRef,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReconcileRHSSOPostgresCredentials(context.TODO(), tt.installation, tt.fakeClient(), tt.postgresName, defaultOperatorNamespace, defaultRHSSONamespace, constants.GcpSnapshotFrequency, constants.GcpSnapshotRetention)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileRHSSOPostgresCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("got should be nil, got = %v", got)
				return
			}

			if tt.want != nil && got.Name != tt.want.Name && got.Spec.Tier != "production" {
				t.Errorf("reconcileCloudResources() got = %v, want = %v", got.Name, tt.want.Name)
			}
		})
	}
}
