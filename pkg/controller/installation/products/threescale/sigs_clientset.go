package threescale

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	fakesigs "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

//go:generate moq -out sigs_client_moq.go . SigsClientInterface
type SigsClientInterface interface {
	sigs.Reader
	sigs.Writer
	sigs.StatusClient
	GetSigsClient() sigs.Client
}

func NewSigsClientMoqWithScheme(clientScheme *runtime.Scheme, initObjs ...runtime.Object) *SigsClientInterfaceMock {
	sigsClient := fakesigs.NewFakeClientWithScheme(clientScheme, initObjs...)
	return &SigsClientInterfaceMock{
		GetSigsClientFunc: func() sigs.Client {
			return sigsClient
		},
		GetFunc: func(ctx context.Context, key sigs.ObjectKey, obj runtime.Object) error {
			return sigsClient.Get(ctx, key, obj)
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object) error {
			return sigsClient.Create(ctx, obj)
		},
		UpdateFunc: func(ctx context.Context, obj runtime.Object) error {
			return sigsClient.Update(ctx, obj)
		},
		DeleteFunc: func(ctx context.Context, obj runtime.Object, opts ...sigs.DeleteOptionFunc) error {
			return sigsClient.Delete(ctx, obj)
		},
		ListFunc: func(ctx context.Context, opts *sigs.ListOptions, list runtime.Object) error {
			return sigsClient.List(ctx, opts, list)
		},
		StatusFunc: func() sigs.StatusWriter {
			return sigsClient.Status()
		},
	}
}
