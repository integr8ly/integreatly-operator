package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func NewTestClient(scheme *runtime.Scheme, initObj ...runtime.Object) k8sclient.Client {
	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(initObj...).Build()
}

func NewSubResourceWriterMock(wantErr bool) k8sclient.SubResourceWriter {
	return &SubResourceWriterMock{wantErr: wantErr}
}

type SubResourceWriterMock struct {
	wantErr bool
}

func (s *SubResourceWriterMock) Create(_ context.Context, _ k8sclient.Object, _ k8sclient.Object, _ ...k8sclient.SubResourceCreateOption) error {
	if s.wantErr {
		return fmt.Errorf("error")
	}
	return nil
}
func (s *SubResourceWriterMock) Update(_ context.Context, _ k8sclient.Object, _ ...k8sclient.SubResourceUpdateOption) error {
	if s.wantErr {
		return fmt.Errorf("error")
	}
	return nil
}
func (s *SubResourceWriterMock) Patch(_ context.Context, _ k8sclient.Object, _ k8sclient.Patch, _ ...k8sclient.SubResourcePatchOption) error {
	if s.wantErr {
		return fmt.Errorf("error")
	}
	return nil
}
