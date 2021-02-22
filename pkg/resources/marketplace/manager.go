package marketplace

import (
	"context"
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IntegreatlyChannel = "rhmi"
	CatalogSourceName  = "rhmi-registry-cs"
	OperatorGroupName  = "rhmi-registry-og"
	Publisher          = "RHMI"
)

var log = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "marketplace"})

//go:generate moq -out MarketplaceManager_moq.go . MarketplaceInterface
type MarketplaceInterface interface {
	InstallOperator(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error
	GetSubscriptionInstallPlan(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*coreosv1alpha1.InstallPlan, *coreosv1alpha1.Subscription, error)
}

type Manager struct{}

var _ MarketplaceInterface = &Manager{}

func NewManager() *Manager {
	return &Manager{}
}

type Target struct {
	Namespace,
	Pkg,
	Channel string
}

func (m *Manager) InstallOperator(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error {
	sub := &coreosv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      t.Pkg,
		},
	}
	sub.Spec = &coreosv1alpha1.SubscriptionSpec{
		InstallPlanApproval:    approvalStrategy,
		Channel:                t.Channel,
		Package:                t.Pkg,
		CatalogSourceNamespace: t.Namespace,
	}

	res, err := catalogSourceReconciler.Reconcile(ctx)
	if res.Requeue {
		return fmt.Errorf("Requeue")
	}
	if err != nil {
		return err
	}
	sub.Spec.CatalogSource = catalogSourceReconciler.CatalogSourceName()

	//catalog source is ready create the other stuff
	og := &v1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      OperatorGroupName,
			Labels:    map[string]string{"integreatly": t.Pkg},
		},
		Spec: v1.OperatorGroupSpec{
			TargetNamespaces: operatorGroupNamespaces,
		},
	}
	err = serverClient.Create(ctx, og)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error("error creating operator group", err)
		return err
	}

	log.Infof("Creating subscription in ns if it doesn't already exist", l.Fields{"ns": t.Namespace})
	err = serverClient.Create(ctx, sub)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error("error creating sub", err)
		return err
	}

	return nil

}

func (m *Manager) getSubscription(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*coreosv1alpha1.Subscription, error) {
	sub := &coreosv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      subName,
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: sub.Name, Namespace: sub.Namespace}, sub)
	if err != nil {
		log.Errorf("Error getting subscription", l.Fields{"name": subName, "ns": ns}, err)
		return nil, err
	}
	return sub, nil
}

func (m *Manager) GetSubscriptionInstallPlan(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*coreosv1alpha1.InstallPlan, *coreosv1alpha1.Subscription, error) {
	sub, err := m.getSubscription(ctx, serverClient, subName, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSubscriptionInstallPlan: %w", err)
	}
	if sub.Status.Install == nil || sub.Status.InstallPlanRef == nil {
		return nil, sub, k8serr.NewNotFound(coreosv1alpha1.Resource("installplan"), "")
	}

	ip := &coreosv1alpha1.InstallPlan{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      sub.Status.InstallPlanRef.Name,
		Namespace: sub.Status.InstallPlanRef.Namespace,
	}, ip); err != nil {
		if k8serr.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	return ip, sub, err
}
