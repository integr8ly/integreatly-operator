package marketplace

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	SubscriptionName,
	Package,
	Channel string
}

func (m *Manager) InstallOperator(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error {
	sub := &coreosv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      t.SubscriptionName,
		},
	}

	res, err := catalogSourceReconciler.Reconcile(ctx, t.SubscriptionName)
	if res.Requeue {
		return fmt.Errorf("Requeue")
	}
	if err != nil {
		return err
	}

	logrus.Infof("Catalog Source Name: " + catalogSourceReconciler.CatalogSourceName())
	logrus.Infof("Catalog Source NS: " + catalogSourceReconciler.CatalogSourceNamespace())
	logrus.Infof("Package: " + t.Package)
	logrus.Infof("Target %v", t)

	mutateSub := func() error {

		sub.Spec = &coreosv1alpha1.SubscriptionSpec{
			InstallPlanApproval:    approvalStrategy,
			Channel:                t.Channel,
			Package:                t.Package,
			CatalogSource:          catalogSourceReconciler.CatalogSourceName(),
			CatalogSourceNamespace: catalogSourceReconciler.CatalogSourceNamespace(),
		}

		logrus.Infof("Catalog Source Name 2: " + catalogSourceReconciler.CatalogSourceName())
		logrus.Infof("Catalog Source NS 2 : " + catalogSourceReconciler.CatalogSourceNamespace())
		logrus.Infof("Package 2: " + t.Package)
		logrus.Infof("Channel 2: " + t.Channel)

		return nil
	}

	//catalog source is ready create the other stuff
	og := &v1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      OperatorGroupName,
			Labels:    map[string]string{"integreatly": t.SubscriptionName},
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
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, sub, mutateSub)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error("error creating sub", err)
		logrus.Errorf("sub:: %v", sub)
		return err
	}

	if (k8serr.IsAlreadyExists(err)) {
		log.Info("Sub already exist")
	}


	logrus.Infof("sub.Spec.Package: " + sub.Spec.Package)
	logrus.Infof("sub.Spec.Channel: " + sub.Spec.Channel)
	logrus.Infof("sub.Spec.CatalogSource: " + sub.Spec.CatalogSource)
	logrus.Infof("sub.Spec.CatalogSourceNamespace: " + sub.Spec.CatalogSourceNamespace)
	if (sub.Status.InstallPlanRef != nil) {
		logrus.Infof("sub.Status.InstallPlanRef.Name: " + sub.Status.InstallPlanRef.Name)
	}


	logrus.Infof("Sub 2 %v", sub)


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

	logrus.Infof("Sub name and ns: " +  "  " +  subName +"  " + ns)

	sub, err := m.getSubscription(ctx, serverClient, subName, ns)
	if err != nil {
		logrus.Error("Error getting install plan %v", err)
		return nil, nil, fmt.Errorf("GetSubscriptionInstallPlan: %w", err)
	}

	logrus.Infof("Sub 1: %v", sub)

	if sub.Status.Install == nil || sub.Status.InstallPlanRef == nil {
		logrus.Error("Error getting install plan, 2 %v", err)
		return nil, sub, k8serr.NewNotFound(coreosv1alpha1.Resource("installplan"), "")
	}

	if (sub.Status.InstallPlanRef != nil) {
		logrus.Info("Getting plan " +  sub.Status.InstallPlanRef.Name + "   " +  sub.Status.InstallPlanRef.Namespace)
	}

	ip := &coreosv1alpha1.InstallPlan{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      sub.Status.InstallPlanRef.Name,
		Namespace: sub.Status.InstallPlanRef.Namespace,
	}, ip); err != nil {
		logrus.Error("Error getting install plan, 3 %v", err)
		if k8serr.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	return ip, sub, err
}
