package k8s

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/test/utils"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
)

func TestPatchIfExists(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		fn           controllerutil.MutateFn
		obj          k8sclient.Object
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
				serverClient: fakeclient.NewFakeClientWithScheme(scheme, &k8sappsv1.Deployment{
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
					GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
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
					GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
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
					GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
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
					GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return nil
					},
					PatchFunc: func(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
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
