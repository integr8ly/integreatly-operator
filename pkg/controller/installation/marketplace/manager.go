package marketplace

import (
	"context"
	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	of "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	marketplacev2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	providerLabel      = "opsrc-provider"
	IntegreatlyChannel = "integreatly"
)

type operatorSources struct {
	Integreatly marketplacev1.OperatorSource
}

func GetOperatorSources() *operatorSources {
	return &operatorSources{
		Integreatly: marketplacev1.OperatorSource{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					providerLabel: "integreatly",
				},
			},
			Spec: marketplacev1.OperatorSourceSpec{
				DisplayName: "Integreatly Operators",
				Publisher:   "Integreatly",
			},
		},
	}
}

//go:generate moq -out MarketplaceManager_moq.go . MarketplaceInterface
type MarketplaceInterface interface {
	InstallOperator(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval) error
	GetSubscriptionInstallPlans(ctx context.Context, serverClient pkgclient.Client, subName, ns string) (*coreosv1alpha1.InstallPlanList, *coreosv1alpha1.Subscription, error)
}

type MarketplaceManager struct{}

func NewManager() *MarketplaceManager {
	return &MarketplaceManager{}
}

type Target struct {
	Namespace, Pkg, Channel string
}

func (m *MarketplaceManager) createAndWaitCatalogSource(ctx context.Context, owner ownerutil.Owner, t Target, os marketplacev1.OperatorSource, client pkgclient.Client) (string, error) {

	csc := &marketplacev2.CatalogSourceConfig{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "installed-" + os.Labels[providerLabel] + "-" + t.Namespace + "-",
			Namespace:    "openshift-marketplace",
			Labels:       map[string]string{"integreatly": "true"},
		},
		Spec: marketplacev2.CatalogSourceConfigSpec{
			DisplayName:     os.Spec.DisplayName,
			Publisher:       os.Spec.Publisher,
			Packages:        t.Pkg,
			TargetNamespace: t.Namespace,
			Source:          os.Name,
		},
	}
	csList := &of.CatalogSourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CatalogSourceList",
			APIVersion: of.SchemeGroupVersion.String(),
		},
		ListMeta: metav1.ListMeta{},
	}
	ownerutil.EnsureOwner(csc, owner)

	if err := client.List(ctx, &pkgclient.ListOptions{Namespace: t.Namespace}, csList); err != nil {
		return "", err
	}
	// as each operator is the only that should be installed in that we assume the catalog source is present if more than 0 returned
	if len(csList.Items) == 0 {
		if err := client.Create(ctx, csc); err != nil {
			return "", errors.Wrap(err, "failed to create catalog source config")
		}
	}

	var catalogSourceName string
	return catalogSourceName, wait.Poll(time.Second, time.Minute*5, func() (done bool, err error) {
		err = client.List(ctx, &pkgclient.ListOptions{Namespace: t.Namespace}, csList)
		if err == nil && len(csList.Items) > 0 {
			catalogSourceName = csList.Items[0].Name
			return true, nil
		}
		return false, err
	})

}

func (m *MarketplaceManager) InstallOperator(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval) error {
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
	ownerutil.EnsureOwner(sub, owner)

	csName, err := m.createAndWaitCatalogSource(ctx, owner, t, os, serverClient)
	if err != nil {
		return err
	}
	sub.Spec.CatalogSource = csName

	//catalog source is ready create the other stuff
	og := &v1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      t.Namespace + "-integreatly",
			Labels:    map[string]string{"integreatly": t.Pkg},
		},
		Spec: v1.OperatorGroupSpec{
			TargetNamespaces: operatorGroupNamespaces,
		},
	}
	ownerutil.EnsureOwner(og, owner)
	err = serverClient.Create(ctx, og)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		logrus.Infof("error creating operator group")
		return err
	}

	logrus.Infof("creating subscription in ns if it doesn't already exist: %s", t.Namespace)
	err = serverClient.Create(ctx, sub)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		logrus.Infof("error creating sub")
		return err
	}

	return nil

}

func (m *MarketplaceManager) getSubscription(ctx context.Context, serverClient pkgclient.Client, subName, ns string) (*coreosv1alpha1.Subscription, error) {
	sub := &coreosv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      subName,
		},
	}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: sub.Name, Namespace: sub.Namespace}, sub)
	if err != nil {
		logrus.Infof("Error getting subscription %s in ns: %s", subName, ns)
		return nil, err
	}
	return sub, nil
}

func (m *MarketplaceManager) GetSubscriptionInstallPlans(ctx context.Context, serverClient pkgclient.Client, subName, ns string) (*coreosv1alpha1.InstallPlanList, *coreosv1alpha1.Subscription, error) {
	sub, err := m.getSubscription(ctx, serverClient, subName, ns)
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetSubscriptionInstallPlan")
	}
	if sub.Status.Install == nil {
		return nil, sub, k8serr.NewNotFound(coreosv1alpha1.Resource("installplan"), "")
	}

	ip := &coreosv1alpha1.InstallPlanList{}

	err = serverClient.List(ctx, &pkgclient.ListOptions{Namespace: ns}, ip)
	if err != nil {
		return nil, nil, err
	}

	return ip, sub, err
}
