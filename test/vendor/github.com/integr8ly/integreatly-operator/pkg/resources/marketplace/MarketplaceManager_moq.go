// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package marketplace

import (
	"context"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

// Ensure, that MarketplaceInterfaceMock does implement MarketplaceInterface.
// If this is not the case, regenerate this file with moq.
var _ MarketplaceInterface = &MarketplaceInterfaceMock{}

// MarketplaceInterfaceMock is a mock implementation of MarketplaceInterface.
//
// 	func TestSomethingThatUsesMarketplaceInterface(t *testing.T) {
//
// 		// make and configure a mocked MarketplaceInterface
// 		mockedMarketplaceInterface := &MarketplaceInterfaceMock{
// 			GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (*coreosv1alpha1.InstallPlan, *coreosv1alpha1.Subscription, error) {
// 				panic("mock out the GetSubscriptionInstallPlan method")
// 			},
// 			InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error {
// 				panic("mock out the InstallOperator method")
// 			},
// 		}
//
// 		// use mockedMarketplaceInterface in code that requires MarketplaceInterface
// 		// and then make assertions.
//
// 	}
type MarketplaceInterfaceMock struct {
	// GetSubscriptionInstallPlanFunc mocks the GetSubscriptionInstallPlan method.
	GetSubscriptionInstallPlanFunc func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (*coreosv1alpha1.InstallPlan, *coreosv1alpha1.Subscription, error)

	// InstallOperatorFunc mocks the InstallOperator method.
	InstallOperatorFunc func(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error

	// calls tracks calls to the methods.
	calls struct {
		// GetSubscriptionInstallPlan holds details about calls to the GetSubscriptionInstallPlan method.
		GetSubscriptionInstallPlan []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ServerClient is the serverClient argument value.
			ServerClient k8sclient.Client
			// SubName is the subName argument value.
			SubName string
			// Ns is the ns argument value.
			Ns string
		}
		// InstallOperator holds details about calls to the InstallOperator method.
		InstallOperator []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ServerClient is the serverClient argument value.
			ServerClient k8sclient.Client
			// T is the t argument value.
			T Target
			// OperatorGroupNamespaces is the operatorGroupNamespaces argument value.
			OperatorGroupNamespaces []string
			// ApprovalStrategy is the approvalStrategy argument value.
			ApprovalStrategy coreosv1alpha1.Approval
			// CatalogSourceReconciler is the catalogSourceReconciler argument value.
			CatalogSourceReconciler CatalogSourceReconciler
		}
	}
	lockGetSubscriptionInstallPlan sync.RWMutex
	lockInstallOperator            sync.RWMutex
}

// GetSubscriptionInstallPlan calls GetSubscriptionInstallPlanFunc.
func (mock *MarketplaceInterfaceMock) GetSubscriptionInstallPlan(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (*coreosv1alpha1.InstallPlan, *coreosv1alpha1.Subscription, error) {
	if mock.GetSubscriptionInstallPlanFunc == nil {
		panic("MarketplaceInterfaceMock.GetSubscriptionInstallPlanFunc: method is nil but MarketplaceInterface.GetSubscriptionInstallPlan was just called")
	}
	callInfo := struct {
		Ctx          context.Context
		ServerClient k8sclient.Client
		SubName      string
		Ns           string
	}{
		Ctx:          ctx,
		ServerClient: serverClient,
		SubName:      subName,
		Ns:           ns,
	}
	mock.lockGetSubscriptionInstallPlan.Lock()
	mock.calls.GetSubscriptionInstallPlan = append(mock.calls.GetSubscriptionInstallPlan, callInfo)
	mock.lockGetSubscriptionInstallPlan.Unlock()
	return mock.GetSubscriptionInstallPlanFunc(ctx, serverClient, subName, ns)
}

// GetSubscriptionInstallPlanCalls gets all the calls that were made to GetSubscriptionInstallPlan.
// Check the length with:
//     len(mockedMarketplaceInterface.GetSubscriptionInstallPlanCalls())
func (mock *MarketplaceInterfaceMock) GetSubscriptionInstallPlanCalls() []struct {
	Ctx          context.Context
	ServerClient k8sclient.Client
	SubName      string
	Ns           string
} {
	var calls []struct {
		Ctx          context.Context
		ServerClient k8sclient.Client
		SubName      string
		Ns           string
	}
	mock.lockGetSubscriptionInstallPlan.RLock()
	calls = mock.calls.GetSubscriptionInstallPlan
	mock.lockGetSubscriptionInstallPlan.RUnlock()
	return calls
}

// InstallOperator calls InstallOperatorFunc.
func (mock *MarketplaceInterfaceMock) InstallOperator(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error {
	if mock.InstallOperatorFunc == nil {
		panic("MarketplaceInterfaceMock.InstallOperatorFunc: method is nil but MarketplaceInterface.InstallOperator was just called")
	}
	callInfo := struct {
		Ctx                     context.Context
		ServerClient            k8sclient.Client
		T                       Target
		OperatorGroupNamespaces []string
		ApprovalStrategy        coreosv1alpha1.Approval
		CatalogSourceReconciler CatalogSourceReconciler
	}{
		Ctx:                     ctx,
		ServerClient:            serverClient,
		T:                       t,
		OperatorGroupNamespaces: operatorGroupNamespaces,
		ApprovalStrategy:        approvalStrategy,
		CatalogSourceReconciler: catalogSourceReconciler,
	}
	mock.lockInstallOperator.Lock()
	mock.calls.InstallOperator = append(mock.calls.InstallOperator, callInfo)
	mock.lockInstallOperator.Unlock()
	return mock.InstallOperatorFunc(ctx, serverClient, t, operatorGroupNamespaces, approvalStrategy, catalogSourceReconciler)
}

// InstallOperatorCalls gets all the calls that were made to InstallOperator.
// Check the length with:
//     len(mockedMarketplaceInterface.InstallOperatorCalls())
func (mock *MarketplaceInterfaceMock) InstallOperatorCalls() []struct {
	Ctx                     context.Context
	ServerClient            k8sclient.Client
	T                       Target
	OperatorGroupNamespaces []string
	ApprovalStrategy        coreosv1alpha1.Approval
	CatalogSourceReconciler CatalogSourceReconciler
} {
	var calls []struct {
		Ctx                     context.Context
		ServerClient            k8sclient.Client
		T                       Target
		OperatorGroupNamespaces []string
		ApprovalStrategy        coreosv1alpha1.Approval
		CatalogSourceReconciler CatalogSourceReconciler
	}
	mock.lockInstallOperator.RLock()
	calls = mock.calls.InstallOperator
	mock.lockInstallOperator.RUnlock()
	return calls
}