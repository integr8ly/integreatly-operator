// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package providers

import (
	"context"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"sync"
	"time"
)

// Ensure, that DeploymentDetailsMock does implement DeploymentDetails.
// If this is not the case, regenerate this file with moq.
var _ DeploymentDetails = &DeploymentDetailsMock{}

// DeploymentDetailsMock is a mock implementation of DeploymentDetails.
//
//	func TestSomethingThatUsesDeploymentDetails(t *testing.T) {
//
//		// make and configure a mocked DeploymentDetails
//		mockedDeploymentDetails := &DeploymentDetailsMock{
//			DataFunc: func() map[string][]byte {
//				panic("mock out the Data method")
//			},
//		}
//
//		// use mockedDeploymentDetails in code that requires DeploymentDetails
//		// and then make assertions.
//
//	}
type DeploymentDetailsMock struct {
	// DataFunc mocks the Data method.
	DataFunc func() map[string][]byte

	// calls tracks calls to the methods.
	calls struct {
		// Data holds details about calls to the Data method.
		Data []struct {
		}
	}
	lockData sync.RWMutex
}

// Data calls DataFunc.
func (mock *DeploymentDetailsMock) Data() map[string][]byte {
	if mock.DataFunc == nil {
		panic("DeploymentDetailsMock.DataFunc: method is nil but DeploymentDetails.Data was just called")
	}
	callInfo := struct {
	}{}
	mock.lockData.Lock()
	mock.calls.Data = append(mock.calls.Data, callInfo)
	mock.lockData.Unlock()
	return mock.DataFunc()
}

// DataCalls gets all the calls that were made to Data.
// Check the length with:
//
//	len(mockedDeploymentDetails.DataCalls())
func (mock *DeploymentDetailsMock) DataCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockData.RLock()
	calls = mock.calls.Data
	mock.lockData.RUnlock()
	return calls
}

// Ensure, that BlobStorageProviderMock does implement BlobStorageProvider.
// If this is not the case, regenerate this file with moq.
var _ BlobStorageProvider = &BlobStorageProviderMock{}

// BlobStorageProviderMock is a mock implementation of BlobStorageProvider.
//
//	func TestSomethingThatUsesBlobStorageProvider(t *testing.T) {
//
//		// make and configure a mocked BlobStorageProvider
//		mockedBlobStorageProvider := &BlobStorageProviderMock{
//			CreateStorageFunc: func(ctx context.Context, bs *v1alpha1.BlobStorage) (*BlobStorageInstance, croType.StatusMessage, error) {
//				panic("mock out the CreateStorage method")
//			},
//			DeleteStorageFunc: func(ctx context.Context, bs *v1alpha1.BlobStorage) (croType.StatusMessage, error) {
//				panic("mock out the DeleteStorage method")
//			},
//			GetNameFunc: func() string {
//				panic("mock out the GetName method")
//			},
//			GetReconcileTimeFunc: func(bs *v1alpha1.BlobStorage) time.Duration {
//				panic("mock out the GetReconcileTime method")
//			},
//			SupportsStrategyFunc: func(s string) bool {
//				panic("mock out the SupportsStrategy method")
//			},
//		}
//
//		// use mockedBlobStorageProvider in code that requires BlobStorageProvider
//		// and then make assertions.
//
//	}
type BlobStorageProviderMock struct {
	// CreateStorageFunc mocks the CreateStorage method.
	CreateStorageFunc func(ctx context.Context, bs *v1alpha1.BlobStorage) (*BlobStorageInstance, croType.StatusMessage, error)

	// DeleteStorageFunc mocks the DeleteStorage method.
	DeleteStorageFunc func(ctx context.Context, bs *v1alpha1.BlobStorage) (croType.StatusMessage, error)

	// GetNameFunc mocks the GetName method.
	GetNameFunc func() string

	// GetReconcileTimeFunc mocks the GetReconcileTime method.
	GetReconcileTimeFunc func(bs *v1alpha1.BlobStorage) time.Duration

	// SupportsStrategyFunc mocks the SupportsStrategy method.
	SupportsStrategyFunc func(s string) bool

	// calls tracks calls to the methods.
	calls struct {
		// CreateStorage holds details about calls to the CreateStorage method.
		CreateStorage []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Bs is the bs argument value.
			Bs *v1alpha1.BlobStorage
		}
		// DeleteStorage holds details about calls to the DeleteStorage method.
		DeleteStorage []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Bs is the bs argument value.
			Bs *v1alpha1.BlobStorage
		}
		// GetName holds details about calls to the GetName method.
		GetName []struct {
		}
		// GetReconcileTime holds details about calls to the GetReconcileTime method.
		GetReconcileTime []struct {
			// Bs is the bs argument value.
			Bs *v1alpha1.BlobStorage
		}
		// SupportsStrategy holds details about calls to the SupportsStrategy method.
		SupportsStrategy []struct {
			// S is the s argument value.
			S string
		}
	}
	lockCreateStorage    sync.RWMutex
	lockDeleteStorage    sync.RWMutex
	lockGetName          sync.RWMutex
	lockGetReconcileTime sync.RWMutex
	lockSupportsStrategy sync.RWMutex
}

// CreateStorage calls CreateStorageFunc.
func (mock *BlobStorageProviderMock) CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*BlobStorageInstance, croType.StatusMessage, error) {
	if mock.CreateStorageFunc == nil {
		panic("BlobStorageProviderMock.CreateStorageFunc: method is nil but BlobStorageProvider.CreateStorage was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Bs  *v1alpha1.BlobStorage
	}{
		Ctx: ctx,
		Bs:  bs,
	}
	mock.lockCreateStorage.Lock()
	mock.calls.CreateStorage = append(mock.calls.CreateStorage, callInfo)
	mock.lockCreateStorage.Unlock()
	return mock.CreateStorageFunc(ctx, bs)
}

// CreateStorageCalls gets all the calls that were made to CreateStorage.
// Check the length with:
//
//	len(mockedBlobStorageProvider.CreateStorageCalls())
func (mock *BlobStorageProviderMock) CreateStorageCalls() []struct {
	Ctx context.Context
	Bs  *v1alpha1.BlobStorage
} {
	var calls []struct {
		Ctx context.Context
		Bs  *v1alpha1.BlobStorage
	}
	mock.lockCreateStorage.RLock()
	calls = mock.calls.CreateStorage
	mock.lockCreateStorage.RUnlock()
	return calls
}

// DeleteStorage calls DeleteStorageFunc.
func (mock *BlobStorageProviderMock) DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (croType.StatusMessage, error) {
	if mock.DeleteStorageFunc == nil {
		panic("BlobStorageProviderMock.DeleteStorageFunc: method is nil but BlobStorageProvider.DeleteStorage was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Bs  *v1alpha1.BlobStorage
	}{
		Ctx: ctx,
		Bs:  bs,
	}
	mock.lockDeleteStorage.Lock()
	mock.calls.DeleteStorage = append(mock.calls.DeleteStorage, callInfo)
	mock.lockDeleteStorage.Unlock()
	return mock.DeleteStorageFunc(ctx, bs)
}

// DeleteStorageCalls gets all the calls that were made to DeleteStorage.
// Check the length with:
//
//	len(mockedBlobStorageProvider.DeleteStorageCalls())
func (mock *BlobStorageProviderMock) DeleteStorageCalls() []struct {
	Ctx context.Context
	Bs  *v1alpha1.BlobStorage
} {
	var calls []struct {
		Ctx context.Context
		Bs  *v1alpha1.BlobStorage
	}
	mock.lockDeleteStorage.RLock()
	calls = mock.calls.DeleteStorage
	mock.lockDeleteStorage.RUnlock()
	return calls
}

// GetName calls GetNameFunc.
func (mock *BlobStorageProviderMock) GetName() string {
	if mock.GetNameFunc == nil {
		panic("BlobStorageProviderMock.GetNameFunc: method is nil but BlobStorageProvider.GetName was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetName.Lock()
	mock.calls.GetName = append(mock.calls.GetName, callInfo)
	mock.lockGetName.Unlock()
	return mock.GetNameFunc()
}

// GetNameCalls gets all the calls that were made to GetName.
// Check the length with:
//
//	len(mockedBlobStorageProvider.GetNameCalls())
func (mock *BlobStorageProviderMock) GetNameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetName.RLock()
	calls = mock.calls.GetName
	mock.lockGetName.RUnlock()
	return calls
}

// GetReconcileTime calls GetReconcileTimeFunc.
func (mock *BlobStorageProviderMock) GetReconcileTime(bs *v1alpha1.BlobStorage) time.Duration {
	if mock.GetReconcileTimeFunc == nil {
		panic("BlobStorageProviderMock.GetReconcileTimeFunc: method is nil but BlobStorageProvider.GetReconcileTime was just called")
	}
	callInfo := struct {
		Bs *v1alpha1.BlobStorage
	}{
		Bs: bs,
	}
	mock.lockGetReconcileTime.Lock()
	mock.calls.GetReconcileTime = append(mock.calls.GetReconcileTime, callInfo)
	mock.lockGetReconcileTime.Unlock()
	return mock.GetReconcileTimeFunc(bs)
}

// GetReconcileTimeCalls gets all the calls that were made to GetReconcileTime.
// Check the length with:
//
//	len(mockedBlobStorageProvider.GetReconcileTimeCalls())
func (mock *BlobStorageProviderMock) GetReconcileTimeCalls() []struct {
	Bs *v1alpha1.BlobStorage
} {
	var calls []struct {
		Bs *v1alpha1.BlobStorage
	}
	mock.lockGetReconcileTime.RLock()
	calls = mock.calls.GetReconcileTime
	mock.lockGetReconcileTime.RUnlock()
	return calls
}

// SupportsStrategy calls SupportsStrategyFunc.
func (mock *BlobStorageProviderMock) SupportsStrategy(s string) bool {
	if mock.SupportsStrategyFunc == nil {
		panic("BlobStorageProviderMock.SupportsStrategyFunc: method is nil but BlobStorageProvider.SupportsStrategy was just called")
	}
	callInfo := struct {
		S string
	}{
		S: s,
	}
	mock.lockSupportsStrategy.Lock()
	mock.calls.SupportsStrategy = append(mock.calls.SupportsStrategy, callInfo)
	mock.lockSupportsStrategy.Unlock()
	return mock.SupportsStrategyFunc(s)
}

// SupportsStrategyCalls gets all the calls that were made to SupportsStrategy.
// Check the length with:
//
//	len(mockedBlobStorageProvider.SupportsStrategyCalls())
func (mock *BlobStorageProviderMock) SupportsStrategyCalls() []struct {
	S string
} {
	var calls []struct {
		S string
	}
	mock.lockSupportsStrategy.RLock()
	calls = mock.calls.SupportsStrategy
	mock.lockSupportsStrategy.RUnlock()
	return calls
}