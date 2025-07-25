// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package catalogsource

import (
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

// Ensure, that CatalogSourceClientInterfaceMock does implement CatalogSourceClientInterface.
// If this is not the case, regenerate this file with moq.
var _ CatalogSourceClientInterface = &CatalogSourceClientInterfaceMock{}

// CatalogSourceClientInterfaceMock is a mock implementation of CatalogSourceClientInterface.
//
//	func TestSomethingThatUsesCatalogSourceClientInterface(t *testing.T) {
//
//		// make and configure a mocked CatalogSourceClientInterface
//		mockedCatalogSourceClientInterface := &CatalogSourceClientInterfaceMock{
//			GetLatestCSVFunc: func(catalogSourceKey k8sclient.ObjectKey, packageName string, channelName string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
//				panic("mock out the GetLatestCSV method")
//			},
//		}
//
//		// use mockedCatalogSourceClientInterface in code that requires CatalogSourceClientInterface
//		// and then make assertions.
//
//	}
type CatalogSourceClientInterfaceMock struct {
	// GetLatestCSVFunc mocks the GetLatestCSV method.
	GetLatestCSVFunc func(catalogSourceKey k8sclient.ObjectKey, packageName string, channelName string) (*operatorsv1alpha1.ClusterServiceVersion, error)

	// calls tracks calls to the methods.
	calls struct {
		// GetLatestCSV holds details about calls to the GetLatestCSV method.
		GetLatestCSV []struct {
			// CatalogSourceKey is the catalogSourceKey argument value.
			CatalogSourceKey k8sclient.ObjectKey
			// PackageName is the packageName argument value.
			PackageName string
			// ChannelName is the channelName argument value.
			ChannelName string
		}
	}
	lockGetLatestCSV sync.RWMutex
}

// GetLatestCSV calls GetLatestCSVFunc.
func (mock *CatalogSourceClientInterfaceMock) GetLatestCSV(catalogSourceKey k8sclient.ObjectKey, packageName string, channelName string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	if mock.GetLatestCSVFunc == nil {
		panic("CatalogSourceClientInterfaceMock.GetLatestCSVFunc: method is nil but CatalogSourceClientInterface.GetLatestCSV was just called")
	}
	callInfo := struct {
		CatalogSourceKey k8sclient.ObjectKey
		PackageName      string
		ChannelName      string
	}{
		CatalogSourceKey: catalogSourceKey,
		PackageName:      packageName,
		ChannelName:      channelName,
	}
	mock.lockGetLatestCSV.Lock()
	mock.calls.GetLatestCSV = append(mock.calls.GetLatestCSV, callInfo)
	mock.lockGetLatestCSV.Unlock()
	return mock.GetLatestCSVFunc(catalogSourceKey, packageName, channelName)
}

// GetLatestCSVCalls gets all the calls that were made to GetLatestCSV.
// Check the length with:
//
//	len(mockedCatalogSourceClientInterface.GetLatestCSVCalls())
func (mock *CatalogSourceClientInterfaceMock) GetLatestCSVCalls() []struct {
	CatalogSourceKey k8sclient.ObjectKey
	PackageName      string
	ChannelName      string
} {
	var calls []struct {
		CatalogSourceKey k8sclient.ObjectKey
		PackageName      string
		ChannelName      string
	}
	mock.lockGetLatestCSV.RLock()
	calls = mock.calls.GetLatestCSV
	mock.lockGetLatestCSV.RUnlock()
	return calls
}
