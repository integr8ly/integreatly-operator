package marketplace

import (
	"context"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IntegreatlyChannel = "integreatly"
)

//go:generate moq -out MarketplaceManager_moq.go . MarketplaceInterface
type MarketplaceInterface interface {
	InstallOperator(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval) error
	GetSubscriptionInstallPlans(ctx context.Context, serverClient pkgclient.Client, subName, ns string) (*coreosv1alpha1.InstallPlanList, *coreosv1alpha1.Subscription, error)
}

type MarketplaceManager struct{}

func NewManager() *MarketplaceManager {
	return &MarketplaceManager{}
}

type Target struct {
	Namespace,
	Pkg,
	Channel string
	ManifestPackage string
}

func (m *MarketplaceManager) InstallOperator(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, t Target, operatorGroupNamespaces []string, approvalStrategy coreosv1alpha1.Approval) error {
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

	csName, err := m.createAndWaitCatalogSource(ctx, owner, t, serverClient)
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

func (m *MarketplaceManager) createAndWaitCatalogSource(ctx context.Context, owner ownerutil.Owner, t Target, client pkgclient.Client) (string, error) {

	configMapData, err := GenerateRegistryConfigMapFromManifest(t.ManifestPackage)
	if err != nil {
		return "", fmt.Errorf("Failed to generated config map data from manifest: %w", err)
	}

	configMapName, err := m.reconcileRegistryConfigMap(ctx, client, t.Namespace, configMapData)
	if err != nil {
		return "", fmt.Errorf("Failed to reconcile config map for registry: %w", err)
	}

	csSourceName, err := m.reconcileCatalogSource(ctx, client, t.Namespace, configMapName)
	if err != nil {
		return "", fmt.Errorf("Failed to reconcile catalog source for registry: %w", err)
	}

	return csSourceName, nil
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
		return nil, nil, fmt.Errorf("GetSubscriptionInstallPlan: %w", err)
	}
	if sub.Status.Install == nil {
		return nil, sub, k8serr.NewNotFound(coreosv1alpha1.Resource("installplan"), "")
	}

	ip := &coreosv1alpha1.InstallPlanList{}

	ipListOpts := []pkgclient.ListOption{
		pkgclient.InNamespace(ns),
	}
	err = serverClient.List(ctx, ip, ipListOpts...)
	if err != nil {
		return nil, nil, err
	}

	return ip, sub, err
}

func (m *MarketplaceManager) reconcileRegistryConfigMap(ctx context.Context, client pkgclient.Client, namespace string, configMapData map[string]string) (string, error) {
	logrus.Infof("Reconciling registry config map for namespace %s", namespace)

	configMapName := "registry-cm-" + namespace
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
		},
	}

	err := client.Get(ctx, pkgclient.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, configMap)

	if err != nil && !k8serr.IsNotFound(err) {
		return "", fmt.Errorf("Failed to get config map %s from %s namespace: %w", configMap.Name, configMap.Namespace, err)
	} else if k8serr.IsNotFound(err) {
		configMap.Data = configMapData
		if err := client.Create(ctx, configMap); err != nil {
			return "", fmt.Errorf("Failed to create configmap %s in %s namespace: %w", configMap.Name, configMap.Namespace, err)
		}

		logrus.Infof("Created registry config map for namepsace %s", namespace)
	} else {
		if !reflect.DeepEqual(configMap.Data, configMapData) {
			configMap.Data = configMapData
			if err := client.Update(ctx, configMap); err != nil {
				return "", fmt.Errorf("Failed to update configmap %s in %s namespace: %w", configMap.Name, configMap.Namespace, err)
			}

			logrus.Infof("Updated config map %s in namspace %s", configMapName, namespace)
		}
	}

	logrus.Infof("Successfully reconciled registry config map for namespace %s", namespace)

	return configMapName, nil
}

func (m *MarketplaceManager) reconcileCatalogSource(ctx context.Context, client pkgclient.Client, namespace string, configMapName string) (string, error) {

	logrus.Infof("Reconciling registry catalog source for namespace %s", namespace)

	catalogSourceName := "registry-cs-" + namespace

	catalogSource := &coreosv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalogSourceName,
			Namespace: namespace,
		},
	}

	catalogSourceSpec := coreosv1alpha1.CatalogSourceSpec{
		SourceType:  coreosv1alpha1.SourceTypeConfigmap,
		ConfigMap:   configMapName,
		DisplayName: catalogSourceName,
		Publisher:   "Integreatly",
	}

	err := client.Get(ctx, pkgclient.ObjectKey{Name: catalogSource.Name, Namespace: catalogSource.Namespace}, catalogSource)

	if err != nil && !k8serr.IsNotFound(err) {
		return "", fmt.Errorf("Failed to get catalog source %s from %s namespace: %w", catalogSource.Name, catalogSource.Namespace, err)
	} else if k8serr.IsNotFound(err) {
		catalogSource.Spec = catalogSourceSpec
		if err := client.Create(ctx, catalogSource); err != nil {
			return "", fmt.Errorf("Failed to create catalog source %s in %s namespace: %w", catalogSource.Name, catalogSource.Namespace, err)
		}

		logrus.Infof("Created registry catalog source for namespace %s", namespace)
	} else {
		if catalogSource.Spec.ConfigMap != catalogSourceSpec.ConfigMap {
			catalogSource.Spec.ConfigMap = catalogSourceSpec.ConfigMap
			if err := client.Update(ctx, catalogSource); err != nil {
				return "", fmt.Errorf("Failed to update catalog source %s in %s namespace: %w", catalogSource.Name, catalogSource.Namespace, err)
			}
			logrus.Infof("Updated registry catalog source for namespace %s", namespace)
		}
	}

	logrus.Infof("Successfully reconciled registry catalog source for namespace %s", namespace)

	return catalogSourceName, nil
}
