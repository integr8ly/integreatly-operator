// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package client

import (
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

// Ensure, that SigsClientInterfaceMock does implement SigsClientInterface.
// If this is not the case, regenerate this file with moq.
var _ SigsClientInterface = &SigsClientInterfaceMock{}

// SigsClientInterfaceMock is a mock implementation of SigsClientInterface.
//
//	func TestSomethingThatUsesSigsClientInterface(t *testing.T) {
//
//		// make and configure a mocked SigsClientInterface
//		mockedSigsClientInterface := &SigsClientInterfaceMock{
//			CreateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
//				panic("mock out the Create method")
//			},
//			DeleteFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
//				panic("mock out the Delete method")
//			},
//			DeleteAllOfFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteAllOfOption) error {
//				panic("mock out the DeleteAllOf method")
//			},
//			GetFunc: func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
//				panic("mock out the Get method")
//			},
//			GetSigsClientFunc: func() k8sclient.Client {
//				panic("mock out the GetSigsClient method")
//			},
//			GroupVersionKindForFunc: func(obj runtime.Object) (schema.GroupVersionKind, error) {
//				panic("mock out the GroupVersionKindFor method")
//			},
//			IsObjectNamespacedFunc: func(obj runtime.Object) (bool, error) {
//				panic("mock out the IsObjectNamespaced method")
//			},
//			ListFunc: func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
//				panic("mock out the List method")
//			},
//			PatchFunc: func(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
//				panic("mock out the Patch method")
//			},
//			RESTMapperFunc: func() meta.RESTMapper {
//				panic("mock out the RESTMapper method")
//			},
//			SchemeFunc: func() *runtime.Scheme {
//				panic("mock out the Scheme method")
//			},
//			StatusFunc: func() k8sclient.SubResourceWriter {
//				panic("mock out the Status method")
//			},
//			SubResourceFunc: func(subResource string) k8sclient.SubResourceClient {
//				panic("mock out the SubResource method")
//			},
//			UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
//				panic("mock out the Update method")
//			},
//			WatchFunc: func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) (watch.Interface, error) {
//				panic("mock out the Watch method")
//			},
//		}
//
//		// use mockedSigsClientInterface in code that requires SigsClientInterface
//		// and then make assertions.
//
//	}
type SigsClientInterfaceMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error

	// DeleteAllOfFunc mocks the DeleteAllOf method.
	DeleteAllOfFunc func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteAllOfOption) error

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error

	// GetSigsClientFunc mocks the GetSigsClient method.
	GetSigsClientFunc func() k8sclient.Client

	// GroupVersionKindForFunc mocks the GroupVersionKindFor method.
	GroupVersionKindForFunc func(obj runtime.Object) (schema.GroupVersionKind, error)

	// IsObjectNamespacedFunc mocks the IsObjectNamespaced method.
	IsObjectNamespacedFunc func(obj runtime.Object) (bool, error)

	// ListFunc mocks the List method.
	ListFunc func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error

	// PatchFunc mocks the Patch method.
	PatchFunc func(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error

	// RESTMapperFunc mocks the RESTMapper method.
	RESTMapperFunc func() meta.RESTMapper

	// SchemeFunc mocks the Scheme method.
	SchemeFunc func() *runtime.Scheme

	// StatusFunc mocks the Status method.
	StatusFunc func() k8sclient.SubResourceWriter

	// SubResourceFunc mocks the SubResource method.
	SubResourceFunc func(subResource string) k8sclient.SubResourceClient

	// UpdateFunc mocks the Update method.
	UpdateFunc func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error

	// WatchFunc mocks the Watch method.
	WatchFunc func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) (watch.Interface, error)

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Obj is the obj argument value.
			Obj k8sclient.Object
			// Opts is the opts argument value.
			Opts []k8sclient.CreateOption
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Obj is the obj argument value.
			Obj k8sclient.Object
			// Opts is the opts argument value.
			Opts []k8sclient.DeleteOption
		}
		// DeleteAllOf holds details about calls to the DeleteAllOf method.
		DeleteAllOf []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Obj is the obj argument value.
			Obj k8sclient.Object
			// Opts is the opts argument value.
			Opts []k8sclient.DeleteAllOfOption
		}
		// Get holds details about calls to the Get method.
		Get []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Key is the key argument value.
			Key k8sclient.ObjectKey
			// Obj is the obj argument value.
			Obj k8sclient.Object
			// Opts is the opts argument value.
			Opts []k8sclient.GetOption
		}
		// GetSigsClient holds details about calls to the GetSigsClient method.
		GetSigsClient []struct {
		}
		// GroupVersionKindFor holds details about calls to the GroupVersionKindFor method.
		GroupVersionKindFor []struct {
			// Obj is the obj argument value.
			Obj runtime.Object
		}
		// IsObjectNamespaced holds details about calls to the IsObjectNamespaced method.
		IsObjectNamespaced []struct {
			// Obj is the obj argument value.
			Obj runtime.Object
		}
		// List holds details about calls to the List method.
		List []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// List is the list argument value.
			List k8sclient.ObjectList
			// Opts is the opts argument value.
			Opts []k8sclient.ListOption
		}
		// Patch holds details about calls to the Patch method.
		Patch []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Obj is the obj argument value.
			Obj k8sclient.Object
			// Patch is the patch argument value.
			Patch k8sclient.Patch
			// Opts is the opts argument value.
			Opts []k8sclient.PatchOption
		}
		// RESTMapper holds details about calls to the RESTMapper method.
		RESTMapper []struct {
		}
		// Scheme holds details about calls to the Scheme method.
		Scheme []struct {
		}
		// Status holds details about calls to the Status method.
		Status []struct {
		}
		// SubResource holds details about calls to the SubResource method.
		SubResource []struct {
			// SubResource is the subResource argument value.
			SubResource string
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Obj is the obj argument value.
			Obj k8sclient.Object
			// Opts is the opts argument value.
			Opts []k8sclient.UpdateOption
		}
		// Watch holds details about calls to the Watch method.
		Watch []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Obj is the obj argument value.
			Obj k8sclient.ObjectList
			// Opts is the opts argument value.
			Opts []k8sclient.ListOption
		}
	}
	lockCreate              sync.RWMutex
	lockDelete              sync.RWMutex
	lockDeleteAllOf         sync.RWMutex
	lockGet                 sync.RWMutex
	lockGetSigsClient       sync.RWMutex
	lockGroupVersionKindFor sync.RWMutex
	lockIsObjectNamespaced  sync.RWMutex
	lockList                sync.RWMutex
	lockPatch               sync.RWMutex
	lockRESTMapper          sync.RWMutex
	lockScheme              sync.RWMutex
	lockStatus              sync.RWMutex
	lockSubResource         sync.RWMutex
	lockUpdate              sync.RWMutex
	lockWatch               sync.RWMutex
}

// Create calls CreateFunc.
func (mock *SigsClientInterfaceMock) Create(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
	if mock.CreateFunc == nil {
		panic("SigsClientInterfaceMock.CreateFunc: method is nil but SigsClientInterface.Create was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.CreateOption
	}{
		Ctx:  ctx,
		Obj:  obj,
		Opts: opts,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(ctx, obj, opts...)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//
//	len(mockedSigsClientInterface.CreateCalls())
func (mock *SigsClientInterfaceMock) CreateCalls() []struct {
	Ctx  context.Context
	Obj  k8sclient.Object
	Opts []k8sclient.CreateOption
} {
	var calls []struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.CreateOption
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *SigsClientInterfaceMock) Delete(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
	if mock.DeleteFunc == nil {
		panic("SigsClientInterfaceMock.DeleteFunc: method is nil but SigsClientInterface.Delete was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.DeleteOption
	}{
		Ctx:  ctx,
		Obj:  obj,
		Opts: opts,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(ctx, obj, opts...)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//
//	len(mockedSigsClientInterface.DeleteCalls())
func (mock *SigsClientInterfaceMock) DeleteCalls() []struct {
	Ctx  context.Context
	Obj  k8sclient.Object
	Opts []k8sclient.DeleteOption
} {
	var calls []struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.DeleteOption
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// DeleteAllOf calls DeleteAllOfFunc.
func (mock *SigsClientInterfaceMock) DeleteAllOf(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteAllOfOption) error {
	if mock.DeleteAllOfFunc == nil {
		panic("SigsClientInterfaceMock.DeleteAllOfFunc: method is nil but SigsClientInterface.DeleteAllOf was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.DeleteAllOfOption
	}{
		Ctx:  ctx,
		Obj:  obj,
		Opts: opts,
	}
	mock.lockDeleteAllOf.Lock()
	mock.calls.DeleteAllOf = append(mock.calls.DeleteAllOf, callInfo)
	mock.lockDeleteAllOf.Unlock()
	return mock.DeleteAllOfFunc(ctx, obj, opts...)
}

// DeleteAllOfCalls gets all the calls that were made to DeleteAllOf.
// Check the length with:
//
//	len(mockedSigsClientInterface.DeleteAllOfCalls())
func (mock *SigsClientInterfaceMock) DeleteAllOfCalls() []struct {
	Ctx  context.Context
	Obj  k8sclient.Object
	Opts []k8sclient.DeleteAllOfOption
} {
	var calls []struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.DeleteAllOfOption
	}
	mock.lockDeleteAllOf.RLock()
	calls = mock.calls.DeleteAllOf
	mock.lockDeleteAllOf.RUnlock()
	return calls
}

// Get calls GetFunc.
func (mock *SigsClientInterfaceMock) Get(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
	if mock.GetFunc == nil {
		panic("SigsClientInterfaceMock.GetFunc: method is nil but SigsClientInterface.Get was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Key  k8sclient.ObjectKey
		Obj  k8sclient.Object
		Opts []k8sclient.GetOption
	}{
		Ctx:  ctx,
		Key:  key,
		Obj:  obj,
		Opts: opts,
	}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc(ctx, key, obj, opts...)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//
//	len(mockedSigsClientInterface.GetCalls())
func (mock *SigsClientInterfaceMock) GetCalls() []struct {
	Ctx  context.Context
	Key  k8sclient.ObjectKey
	Obj  k8sclient.Object
	Opts []k8sclient.GetOption
} {
	var calls []struct {
		Ctx  context.Context
		Key  k8sclient.ObjectKey
		Obj  k8sclient.Object
		Opts []k8sclient.GetOption
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}

// GetSigsClient calls GetSigsClientFunc.
func (mock *SigsClientInterfaceMock) GetSigsClient() k8sclient.Client {
	if mock.GetSigsClientFunc == nil {
		panic("SigsClientInterfaceMock.GetSigsClientFunc: method is nil but SigsClientInterface.GetSigsClient was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetSigsClient.Lock()
	mock.calls.GetSigsClient = append(mock.calls.GetSigsClient, callInfo)
	mock.lockGetSigsClient.Unlock()
	return mock.GetSigsClientFunc()
}

// GetSigsClientCalls gets all the calls that were made to GetSigsClient.
// Check the length with:
//
//	len(mockedSigsClientInterface.GetSigsClientCalls())
func (mock *SigsClientInterfaceMock) GetSigsClientCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetSigsClient.RLock()
	calls = mock.calls.GetSigsClient
	mock.lockGetSigsClient.RUnlock()
	return calls
}

// GroupVersionKindFor calls GroupVersionKindForFunc.
func (mock *SigsClientInterfaceMock) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	if mock.GroupVersionKindForFunc == nil {
		panic("SigsClientInterfaceMock.GroupVersionKindForFunc: method is nil but SigsClientInterface.GroupVersionKindFor was just called")
	}
	callInfo := struct {
		Obj runtime.Object
	}{
		Obj: obj,
	}
	mock.lockGroupVersionKindFor.Lock()
	mock.calls.GroupVersionKindFor = append(mock.calls.GroupVersionKindFor, callInfo)
	mock.lockGroupVersionKindFor.Unlock()
	return mock.GroupVersionKindForFunc(obj)
}

// GroupVersionKindForCalls gets all the calls that were made to GroupVersionKindFor.
// Check the length with:
//
//	len(mockedSigsClientInterface.GroupVersionKindForCalls())
func (mock *SigsClientInterfaceMock) GroupVersionKindForCalls() []struct {
	Obj runtime.Object
} {
	var calls []struct {
		Obj runtime.Object
	}
	mock.lockGroupVersionKindFor.RLock()
	calls = mock.calls.GroupVersionKindFor
	mock.lockGroupVersionKindFor.RUnlock()
	return calls
}

// IsObjectNamespaced calls IsObjectNamespacedFunc.
func (mock *SigsClientInterfaceMock) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	if mock.IsObjectNamespacedFunc == nil {
		panic("SigsClientInterfaceMock.IsObjectNamespacedFunc: method is nil but SigsClientInterface.IsObjectNamespaced was just called")
	}
	callInfo := struct {
		Obj runtime.Object
	}{
		Obj: obj,
	}
	mock.lockIsObjectNamespaced.Lock()
	mock.calls.IsObjectNamespaced = append(mock.calls.IsObjectNamespaced, callInfo)
	mock.lockIsObjectNamespaced.Unlock()
	return mock.IsObjectNamespacedFunc(obj)
}

// IsObjectNamespacedCalls gets all the calls that were made to IsObjectNamespaced.
// Check the length with:
//
//	len(mockedSigsClientInterface.IsObjectNamespacedCalls())
func (mock *SigsClientInterfaceMock) IsObjectNamespacedCalls() []struct {
	Obj runtime.Object
} {
	var calls []struct {
		Obj runtime.Object
	}
	mock.lockIsObjectNamespaced.RLock()
	calls = mock.calls.IsObjectNamespaced
	mock.lockIsObjectNamespaced.RUnlock()
	return calls
}

// List calls ListFunc.
func (mock *SigsClientInterfaceMock) List(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
	if mock.ListFunc == nil {
		panic("SigsClientInterfaceMock.ListFunc: method is nil but SigsClientInterface.List was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		List k8sclient.ObjectList
		Opts []k8sclient.ListOption
	}{
		Ctx:  ctx,
		List: list,
		Opts: opts,
	}
	mock.lockList.Lock()
	mock.calls.List = append(mock.calls.List, callInfo)
	mock.lockList.Unlock()
	return mock.ListFunc(ctx, list, opts...)
}

// ListCalls gets all the calls that were made to List.
// Check the length with:
//
//	len(mockedSigsClientInterface.ListCalls())
func (mock *SigsClientInterfaceMock) ListCalls() []struct {
	Ctx  context.Context
	List k8sclient.ObjectList
	Opts []k8sclient.ListOption
} {
	var calls []struct {
		Ctx  context.Context
		List k8sclient.ObjectList
		Opts []k8sclient.ListOption
	}
	mock.lockList.RLock()
	calls = mock.calls.List
	mock.lockList.RUnlock()
	return calls
}

// Patch calls PatchFunc.
func (mock *SigsClientInterfaceMock) Patch(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
	if mock.PatchFunc == nil {
		panic("SigsClientInterfaceMock.PatchFunc: method is nil but SigsClientInterface.Patch was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Obj   k8sclient.Object
		Patch k8sclient.Patch
		Opts  []k8sclient.PatchOption
	}{
		Ctx:   ctx,
		Obj:   obj,
		Patch: patch,
		Opts:  opts,
	}
	mock.lockPatch.Lock()
	mock.calls.Patch = append(mock.calls.Patch, callInfo)
	mock.lockPatch.Unlock()
	return mock.PatchFunc(ctx, obj, patch, opts...)
}

// PatchCalls gets all the calls that were made to Patch.
// Check the length with:
//
//	len(mockedSigsClientInterface.PatchCalls())
func (mock *SigsClientInterfaceMock) PatchCalls() []struct {
	Ctx   context.Context
	Obj   k8sclient.Object
	Patch k8sclient.Patch
	Opts  []k8sclient.PatchOption
} {
	var calls []struct {
		Ctx   context.Context
		Obj   k8sclient.Object
		Patch k8sclient.Patch
		Opts  []k8sclient.PatchOption
	}
	mock.lockPatch.RLock()
	calls = mock.calls.Patch
	mock.lockPatch.RUnlock()
	return calls
}

// RESTMapper calls RESTMapperFunc.
func (mock *SigsClientInterfaceMock) RESTMapper() meta.RESTMapper {
	if mock.RESTMapperFunc == nil {
		panic("SigsClientInterfaceMock.RESTMapperFunc: method is nil but SigsClientInterface.RESTMapper was just called")
	}
	callInfo := struct {
	}{}
	mock.lockRESTMapper.Lock()
	mock.calls.RESTMapper = append(mock.calls.RESTMapper, callInfo)
	mock.lockRESTMapper.Unlock()
	return mock.RESTMapperFunc()
}

// RESTMapperCalls gets all the calls that were made to RESTMapper.
// Check the length with:
//
//	len(mockedSigsClientInterface.RESTMapperCalls())
func (mock *SigsClientInterfaceMock) RESTMapperCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockRESTMapper.RLock()
	calls = mock.calls.RESTMapper
	mock.lockRESTMapper.RUnlock()
	return calls
}

// Scheme calls SchemeFunc.
func (mock *SigsClientInterfaceMock) Scheme() *runtime.Scheme {
	if mock.SchemeFunc == nil {
		panic("SigsClientInterfaceMock.SchemeFunc: method is nil but SigsClientInterface.Scheme was just called")
	}
	callInfo := struct {
	}{}
	mock.lockScheme.Lock()
	mock.calls.Scheme = append(mock.calls.Scheme, callInfo)
	mock.lockScheme.Unlock()
	return mock.SchemeFunc()
}

// SchemeCalls gets all the calls that were made to Scheme.
// Check the length with:
//
//	len(mockedSigsClientInterface.SchemeCalls())
func (mock *SigsClientInterfaceMock) SchemeCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockScheme.RLock()
	calls = mock.calls.Scheme
	mock.lockScheme.RUnlock()
	return calls
}

// Status calls StatusFunc.
func (mock *SigsClientInterfaceMock) Status() k8sclient.SubResourceWriter {
	if mock.StatusFunc == nil {
		panic("SigsClientInterfaceMock.StatusFunc: method is nil but SigsClientInterface.Status was just called")
	}
	callInfo := struct {
	}{}
	mock.lockStatus.Lock()
	mock.calls.Status = append(mock.calls.Status, callInfo)
	mock.lockStatus.Unlock()
	return mock.StatusFunc()
}

// StatusCalls gets all the calls that were made to Status.
// Check the length with:
//
//	len(mockedSigsClientInterface.StatusCalls())
func (mock *SigsClientInterfaceMock) StatusCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockStatus.RLock()
	calls = mock.calls.Status
	mock.lockStatus.RUnlock()
	return calls
}

// SubResource calls SubResourceFunc.
func (mock *SigsClientInterfaceMock) SubResource(subResource string) k8sclient.SubResourceClient {
	if mock.SubResourceFunc == nil {
		panic("SigsClientInterfaceMock.SubResourceFunc: method is nil but SigsClientInterface.SubResource was just called")
	}
	callInfo := struct {
		SubResource string
	}{
		SubResource: subResource,
	}
	mock.lockSubResource.Lock()
	mock.calls.SubResource = append(mock.calls.SubResource, callInfo)
	mock.lockSubResource.Unlock()
	return mock.SubResourceFunc(subResource)
}

// SubResourceCalls gets all the calls that were made to SubResource.
// Check the length with:
//
//	len(mockedSigsClientInterface.SubResourceCalls())
func (mock *SigsClientInterfaceMock) SubResourceCalls() []struct {
	SubResource string
} {
	var calls []struct {
		SubResource string
	}
	mock.lockSubResource.RLock()
	calls = mock.calls.SubResource
	mock.lockSubResource.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *SigsClientInterfaceMock) Update(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
	if mock.UpdateFunc == nil {
		panic("SigsClientInterfaceMock.UpdateFunc: method is nil but SigsClientInterface.Update was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.UpdateOption
	}{
		Ctx:  ctx,
		Obj:  obj,
		Opts: opts,
	}
	mock.lockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	mock.lockUpdate.Unlock()
	return mock.UpdateFunc(ctx, obj, opts...)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//
//	len(mockedSigsClientInterface.UpdateCalls())
func (mock *SigsClientInterfaceMock) UpdateCalls() []struct {
	Ctx  context.Context
	Obj  k8sclient.Object
	Opts []k8sclient.UpdateOption
} {
	var calls []struct {
		Ctx  context.Context
		Obj  k8sclient.Object
		Opts []k8sclient.UpdateOption
	}
	mock.lockUpdate.RLock()
	calls = mock.calls.Update
	mock.lockUpdate.RUnlock()
	return calls
}

// Watch calls WatchFunc.
func (mock *SigsClientInterfaceMock) Watch(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) (watch.Interface, error) {
	if mock.WatchFunc == nil {
		panic("SigsClientInterfaceMock.WatchFunc: method is nil but SigsClientInterface.Watch was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Obj  k8sclient.ObjectList
		Opts []k8sclient.ListOption
	}{
		Ctx:  ctx,
		Obj:  obj,
		Opts: opts,
	}
	mock.lockWatch.Lock()
	mock.calls.Watch = append(mock.calls.Watch, callInfo)
	mock.lockWatch.Unlock()
	return mock.WatchFunc(ctx, obj, opts...)
}

// WatchCalls gets all the calls that were made to Watch.
// Check the length with:
//
//	len(mockedSigsClientInterface.WatchCalls())
func (mock *SigsClientInterfaceMock) WatchCalls() []struct {
	Ctx  context.Context
	Obj  k8sclient.ObjectList
	Opts []k8sclient.ListOption
} {
	var calls []struct {
		Ctx  context.Context
		Obj  k8sclient.ObjectList
		Opts []k8sclient.ListOption
	}
	mock.lockWatch.RLock()
	calls = mock.calls.Watch
	mock.lockWatch.RUnlock()
	return calls
}
