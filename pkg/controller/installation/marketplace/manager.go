package marketplace

import (
	"context"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type operatorSources struct {
	Redhat marketplacev1.OperatorSource
}

func GetOperatorSources() *operatorSources {
	return &operatorSources{
		Redhat: marketplacev1.OperatorSource{
			Spec: marketplacev1.OperatorSourceSpec{
				DisplayName: "Red Hat Operators",
				Publisher:   "Red Hat",
			},
		},
	}
}

type MarketplaceInterface interface {
	CreateSubscription(os marketplacev1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval) error
}

type MarketplaceManager struct {
	client pkgclient.Client
}

func NewManager(client pkgclient.Client) *MarketplaceManager {
	return &MarketplaceManager{
		client: client,
	}
}

func (m *MarketplaceManager) CreateSubscription(os marketplacev1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval) error {
	sub := &coreosv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      pkg,
		},
	}
	err := m.client.Get(context.TODO(), pkgclient.ObjectKey{Name: sub.Name, Namespace: sub.Namespace}, sub)
	if err == nil {
		return k8serr.NewAlreadyExists(coreosv1alpha1.Resource("subscription"), sub.Name)
	}

	csc := &marketplacev1.CatalogSourceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installed-" + pkg + "-" + ns,
			Namespace: "openshift-marketplace",
		},
		Spec: marketplacev1.CatalogSourceConfigSpec{
			DisplayName:     os.Spec.DisplayName,
			Publisher:       os.Spec.Publisher,
			Packages:        pkg,
			TargetNamespace: ns,
		},
	}
	err = m.client.Create(context.TODO(), csc)
	if err != nil {
		return err
	}

	og := &coreosv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ns,
			GenerateName: ns + "-",
		},
		Spec: coreosv1.OperatorGroupSpec{
			TargetNamespaces: operatorGroupNamespaces,
		},
	}
	err = m.client.Create(context.TODO(), og)
	if err != nil {
		return err
	}

	sub.Spec = &coreosv1alpha1.SubscriptionSpec{
		InstallPlanApproval:    approvalStrategy,
		Channel:                channel,
		Package:                pkg,
		CatalogSource:          csc.Name,
		CatalogSourceNamespace: ns,
	}
	err = m.client.Create(context.TODO(), sub)
	if err != nil {
		return err
	}

	return nil
}
