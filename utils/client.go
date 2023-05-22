package utils

import (
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func NewTestClient(scheme *runtime.Scheme, initObj ...runtime.Object) k8sclient.Client {
	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(initObj...).Build()
}
