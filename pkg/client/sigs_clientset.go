package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakesigs "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

//go:generate moq -out sigs_client_moq.go . SigsClientInterface
type SigsClientInterface interface {
	k8sclient.Reader
	k8sclient.Writer
	k8sclient.StatusClient
	k8sclient.WithWatch
	GetSigsClient() k8sclient.Client
	Scheme() *runtime.Scheme
	RESTMapper() meta.RESTMapper
}

func NewSigsClientMoqWithScheme(clientScheme *runtime.Scheme, initObjs ...runtime.Object) *SigsClientInterfaceMock {
	sigsClient := fakesigs.NewClientBuilder().WithRuntimeObjects(initObjs...).WithScheme(clientScheme).Build()
	return &SigsClientInterfaceMock{
		GetSigsClientFunc: func() k8sclient.Client {
			return sigsClient
		},
		GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
			return sigsClient.Get(ctx, key, obj)
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
	}
}
