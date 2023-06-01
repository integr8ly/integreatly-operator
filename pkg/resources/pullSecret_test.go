package resources

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"k8s.io/apimachinery/pkg/types"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDestinationNameSpace  = "test-namespace"
	testDestinationSecretName = "new-name-of-secret"
)

func TestCopyDefaultPullSecretToNameSpace(t *testing.T) {
	defPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
			Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name         string
		FakeClient   k8sclient.Client
		Installation *integreatlyv1alpha1.RHMI
		Verify       func(client k8sclient.Client, err error, t *testing.T)
	}{
		{
			Name: "Test Default Pull Secret is successfully copied over to target namespace",
			FakeClient: utils.NewTestClient(scheme, defPullSecret, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-namespace",
					Labels:    map[string]string{"webapp": "true"},
				},
			}),
			Installation: &integreatlyv1alpha1.RHMI{},
			Verify: func(c k8sclient.Client, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				s := &corev1.Secret{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: testDestinationSecretName, Namespace: testDestinationNameSpace}, s)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(s.Data["test"], defPullSecret.Data["test"]) {
					t.Fatalf("expected data %v, but got %v", defPullSecret.Data["test"], s.Data["test"])
				}
			},
		},
		{
			Name: "Test Get Default Pull Secret error when trying to copy",
			FakeClient: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testDestinationNameSpace,
					Name:      testDestinationNameSpace,
				},
			}),
			Installation: &integreatlyv1alpha1.RHMI{},
			Verify: func(c k8sclient.Client, err error, t *testing.T) {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := CopyPullSecretToNameSpace(context.TODO(), scenario.Installation.GetPullSecretSpec(), testDestinationNameSpace, testDestinationSecretName, scenario.FakeClient)
			scenario.Verify(scenario.FakeClient, err, t)
		})
	}
}

func TestCopySecret(t *testing.T) {

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
			Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name                       string
		FakeClient                 k8sclient.Client
		Installation               *integreatlyv1alpha1.RHMI
		DestinationSecretName      string
		DestinationSecretNamespace string
		Verify                     func(client k8sclient.Client, err error, t *testing.T, destinationSecretName string, destinationSecretNamespace string)
	}{
		{
			Name: "Test Secret is successfully copied over to target namespace",
			FakeClient: utils.NewTestClient(scheme, sourceSecret, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testDestinationNameSpace,
					Name:      testDestinationNameSpace,
				},
			}),
			DestinationSecretName:      testDestinationSecretName,
			DestinationSecretNamespace: testDestinationNameSpace,
			Verify: func(c k8sclient.Client, err error, t *testing.T, destinationSecretName string, destinationSecretNamespace string) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				destinationSecret := &corev1.Secret{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: destinationSecretName, Namespace: destinationSecretNamespace}, destinationSecret)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !reflect.DeepEqual(sourceSecret.Data, destinationSecret.Data) {
					t.Fatalf("expected data %v, but got %v", sourceSecret.Data, destinationSecret.Data)
				}
			},
		},
		{
			Name: "Test error when trying to copy secret",
			FakeClient: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testDestinationNameSpace,
					Name:      testDestinationNameSpace,
				},
			}),
			DestinationSecretName:      testDestinationSecretName,
			DestinationSecretNamespace: testDestinationNameSpace,
			Verify: func(c k8sclient.Client, err error, t *testing.T, destinationSecretName string, destinationSecretNamespace string) {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := CopySecret(context.TODO(), tt.FakeClient, sourceSecret.Name, sourceSecret.Namespace, tt.DestinationSecretName, tt.DestinationSecretNamespace)
			tt.Verify(tt.FakeClient, err, t, tt.DestinationSecretName, tt.DestinationSecretNamespace)
		})
	}
}

func TestReconcileSecretToRHMIOperatorNamespace(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	operatorSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-rhsso",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	basicConfig := &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return defaultOperatorNamespace
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}

	tests := []struct {
		Name           string
		FakeClient     k8sclient.Client
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectedError  bool
	}{
		{
			Name:           "Test - Successfully copied secret to operator namespace",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ExpectedError:  false,
			FakeClient:     utils.NewTestClient(scheme, operatorSecret),
		},
		{
			Name:           "Test - Failed copying secret to operator namespace",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectedError:  true,
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return nil
				},
				CreateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
				UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got, err := ReconcileSecretToRHMIOperatorNamespace(context.TODO(), tt.FakeClient, basicConfig, operatorSecret.Name, basicConfig.GetOperatorNamespace())
			if (err != nil) != tt.ExpectedError {
				t.Errorf("ReconcileSecretToRHMIOperatorNamespace() error = %v, wantErr %v", err, tt.ExpectedError)
				return
			}
			if got != tt.ExpectedStatus {
				t.Errorf("ReconcileSecretToRHMIOperatorNamespace() got = %v, want %v", got, tt.ExpectedStatus)
			}
		})
	}
}

func TestReconcileSecretToProductNamespace(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	productSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-rhsso",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	basicConfig := &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return defaultOperatorNamespace
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}

	tests := []struct {
		Name           string
		FakeClient     k8sclient.Client
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectedError  bool
	}{
		{
			Name:           "Test - Successfully copied secret to product namespace",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ExpectedError:  false,
			FakeClient:     utils.NewTestClient(scheme, productSecret),
		},
		{
			Name:           "Test - Phase complete on failure to get secret",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ExpectedError:  false,
			FakeClient:     utils.NewTestClient(scheme),
		},
		{
			Name:           "Test - Failed copied secret to product namespace",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectedError:  true,
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return nil
				},
				CreateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
				UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got, err := ReconcileSecretToProductNamespace(context.TODO(), tt.FakeClient, basicConfig, productSecret.Name, basicConfig.GetOperatorNamespace(), getLogger())
			if (err != nil) != tt.ExpectedError {
				t.Errorf("ReconcileSecretToProductNamespace() error = %v, wantErr %v", err, tt.ExpectedError)
				return
			}
			if got != tt.ExpectedStatus {
				t.Errorf("ReconcileSecretToProductNamespace() got = %v, want %v", got, tt.ExpectedStatus)
			}
		})
	}
}
