package client

import (
	"context"
	"github.com/integr8ly/integreatly-operator/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

//go:generate moq -out sigs_client_moq.go . SigsClientInterface
type SigsClientInterface interface {
	k8sclient.Reader
	k8sclient.Writer
	k8sclient.StatusClient
	k8sclient.WithWatch
	k8sclient.SubResourceClientConstructor

	GetSigsClient() k8sclient.Client
	Scheme() *runtime.Scheme
	RESTMapper() meta.RESTMapper
	GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error)
	IsObjectNamespaced(obj runtime.Object) (bool, error)
}

func NewSigsClientMoqWithScheme(clientScheme *runtime.Scheme, initObjs ...runtime.Object) *SigsClientInterfaceMock {
	sigsClient := utils.NewTestClient(clientScheme, initObjs...)
	return NewSigsClientInterfaceMock(sigsClient)
}

func NewSigsClientMoqWithSchemeWithStatusSubresource(clientScheme *runtime.Scheme, initObjs ...runtime.Object) *SigsClientInterfaceMock {
	clientObjs := make([]k8sclient.Object, len(initObjs))
	for i, obj := range initObjs {
		clientObjs[i] = obj.(k8sclient.Object)
	}
	sigsClient := fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(initObjs...).WithStatusSubresource(clientObjs...).Build()
	return NewSigsClientInterfaceMock(sigsClient)
}

func NewSigsClientInterfaceMock(sigsClient k8sclient.Client) *SigsClientInterfaceMock {
	return &SigsClientInterfaceMock{
		GetSigsClientFunc: func() k8sclient.Client {
			return sigsClient
		},
		GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
			return sigsClient.Get(ctx, key, obj, opts...)
		},
		CreateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
			return sigsClient.Create(ctx, obj)
		},
		UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
			return sigsClient.Update(ctx, obj)
		},
		DeleteFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
			return sigsClient.Delete(ctx, obj)
		},
		ListFunc: func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
			return sigsClient.List(ctx, list, opts...)
		},
		StatusFunc: func() k8sclient.StatusWriter {
			return sigsClient.Status()
		},
		PatchFunc: func(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
			return sigsClient.Patch(ctx, obj, patch, opts...)
		},
	}
}
