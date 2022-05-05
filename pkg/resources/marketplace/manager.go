package marketplace

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
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

	mutateSub := func() error {
		sub.Spec = &coreosv1alpha1.SubscriptionSpec{
			InstallPlanApproval:    approvalStrategy,
			Channel:                t.Channel,
			Package:                t.Package,
			CatalogSource:          catalogSourceReconciler.CatalogSourceName(),
			CatalogSourceNamespace: catalogSourceReconciler.CatalogSourceNamespace(),
		}
		return nil
	}
	// TODO this 'Get' is a one off and should be removed post upgrade to 3scale cluster scoped
	// Get CSV as only want 3scale operatorGroup to update when upgrade is finished
	csv := &coreosv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      "3scale-operator.v0.9.0",
		},
	}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Namespace: csv.Namespace, Name: csv.Name}, csv)
	if err != nil {
		log.Error("marketplace manager failed to get 3scale csv : ", err)
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
	//get the rhmi CR to check if its a sandbox install
	rhmiCR := &integreatlyv1alpha1.RHMI{}

	rhoamWatchNamespace, err := k8s.GetWatchNamespace()
	if err != nil {
		log.Error("marketplace manager failed to get rhoam watch namespace : ", err)
	}
	// Using rhmi.GetRhmiCr to get the rhmi CR surprise, surprise
	rhmiCR, err = rhmi.GetRhmiCr(serverClient, ctx, rhoamWatchNamespace, log)
	if err != nil {
		log.Error("marketplace manager failed to get rhmi CR in namespace rhoam : ", err)
	}

	//TODO can maybe return this to a create only function after all clusters are updated in production
	//err = serverClient.Create(ctx, og)
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, og, func() error {
		og.Namespace = t.Namespace
		og.Name = OperatorGroupName
		og.Labels = map[string]string{"integreatly": t.SubscriptionName}
		// checks the 3scale CSV for a Succeeded and InstallSucceeded
		if csv.Status.Phase == "Succeeded" && csv.Status.Reason == "InstallSucceeded" {
			// checks the if rhoam install not a Multitenant and it will update the TargetNamespaces in OperatorGroup
			if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(rhmiCR.Spec.Type)) {
				og.Spec.TargetNamespaces = operatorGroupNamespaces
			}
		}
		return nil
	})
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error("error creating operator group", err)
		return err
	}

	log.Infof("Creating subscription in ns if it doesn't already exist", l.Fields{"ns": t.Namespace})
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, sub, mutateSub)
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
	log.Infof("Get", l.Fields{"Subscription Name": subName, "ns": ns})

	sub, err := m.getSubscription(ctx, serverClient, subName, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSubscriptionInstallPlan: %w", err)
	}
	if sub.Status.Install == nil || sub.Status.InstallPlanRef == nil {
		err = k8serr.NewNotFound(coreosv1alpha1.Resource("installplan"), "")
		log.Error("Error getting install plan ref on subscription, %v", err)
		return nil, sub, err
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
