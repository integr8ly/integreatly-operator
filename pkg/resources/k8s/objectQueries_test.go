package k8s

import (
	"context"
	"fmt"
	"testing"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestPatchIfExists(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx          context.Context
		serverClient client.Client
		fn           controllerutil.MutateFn
		obj          client.Object
	}
	tests := []struct {
		name    string
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "success patching an existing cr",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme, &k8sappsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploymentName",
						Namespace: "deploymentNamespace",
					},
				}),
				fn: func() error {
					return nil
				},
				obj: &k8sappsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploymentName",
						Namespace: "deploymentNamespace",
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "failure fetching a cr that needs to be patched",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return fmt.Errorf("generic error")
					},
				},
				fn: func() error {
					return nil
				},
				obj: &k8sappsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploymentName",
						Namespace: "deploymentNamespace",
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "failure fetching a cr that needs to be patched (not found)",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return k8serr.NewNotFound(schema.GroupResource{}, "generic")
					},
				},
				fn: func() error {
					return nil
				},
				obj: &k8sappsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploymentName",
						Namespace: "deploymentNamespace",
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "failure patching an existing cr during mutate function",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return nil
					},
				},
				fn: func() error {
					return fmt.Errorf("generic error")
				},
				obj: &k8sappsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploymentName",
						Namespace: "deploymentNamespace",
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "failure patching an existing cr",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return nil
					},
					PatchFunc: func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return fmt.Errorf("generic error")
					},
				},
				fn: func() error {
					return nil
				},
				obj: &k8sappsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploymentName",
						Namespace: "deploymentNamespace",
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PatchIfExists(tt.args.ctx, tt.args.serverClient, tt.args.fn, tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("PatchIfExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PatchIfExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureObjectDeleted(t *testing.T) {
	type args struct {
		client client.Client
		object client.Object
	}
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	sampleObject := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testName",
			Namespace: "testNs",
		},
	}
	tests := []struct {
		name    string
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "success triggering kubernetes object deletion",
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, sampleObject),
				object: sampleObject,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "success deleting kubernetes object",
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
				object: sampleObject,
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "failure deleting kubernetes object",
			args: args{
				client: func() client.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.DeleteFunc = func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
						return fmt.Errorf("generic error")
					}
					return mockClient
				}(),
				object: sampleObject,
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EnsureObjectDeleted(context.TODO(), tt.args.client, tt.args.object)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureObjectDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EnsureObjectDeleted() got = %v, want %v", got, tt.want)
			}
		})
	}
}
