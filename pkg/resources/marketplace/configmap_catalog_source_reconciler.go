package marketplace

import (
	"context"
	"fmt"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ConfigMapCatalogSourceReconciler struct {
	ManifestsProductDirectory string
	Client                    k8sclient.Client
	Namespace                 string
	CSName                    string
}

var _ CatalogSourceReconciler = &ConfigMapCatalogSourceReconciler{}

func NewConfigMapCatalogSourceReconciler(manifestsProductDirectory string, client client.Client, namespace string, catalogSourceName string) *ConfigMapCatalogSourceReconciler {
	return &ConfigMapCatalogSourceReconciler{
		ManifestsProductDirectory: manifestsProductDirectory,
		Client:                    client,
		Namespace:                 namespace,
		CSName:                    catalogSourceName,
	}
}

func (r *ConfigMapCatalogSourceReconciler) CatalogSourceName() string {
	return r.CSName
}

func (r *ConfigMapCatalogSourceReconciler) Reconcile(ctx context.Context) (reconcile.Result, error) {
	configMapData, err := GenerateRegistryConfigMapFromManifest(r.ManifestsProductDirectory)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Failed to generated config map data from manifest: %w", err)
	}

	configMapName, err := r.reconcileRegistryConfigMap(ctx, configMapData)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Failed to reconcile config map for registry: %w", err)
	}

	res, err := r.reconcileCatalogSource(ctx, configMapName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Failed to reconcile catalog source for registry: %w", err)
	}

	return res, nil
}

func (r *ConfigMapCatalogSourceReconciler) reconcileRegistryConfigMap(ctx context.Context, configMapData map[string]string) (string, error) {
	logrus.Infof("Reconciling registry config map for namespace %s", r.Namespace)

	configMapName := "registry-cm-" + r.Namespace
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Namespace,
			Name:      configMapName,
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Data = configMapData
		return nil
	})
	if err != nil {
		return configMapName, fmt.Errorf("Failed to create/update configmap %s in %s namespace: %w", configMap.Name, configMap.Namespace, err)
	}

	switch or {
	case controllerutil.OperationResultCreated:
		logrus.Infof("Created registry config map for namepsace %s", r.Namespace)
	case controllerutil.OperationResultUpdated:
		logrus.Infof("Updated config map %s in namspace %s", configMapName, r.Namespace)
	case controllerutil.OperationResultNone:
		break
	default:
		return configMapName, fmt.Errorf("Unknown controllerutil.OperationResult '%v'", or)
	}

	logrus.Infof("Successfully reconciled registry config map for namespace %s", r.Namespace)

	return configMapName, nil
}

func (r *ConfigMapCatalogSourceReconciler) reconcileCatalogSource(ctx context.Context, configMapName string) (reconcile.Result, error) {
	logrus.Infof("Reconciling registry catalog source for namespace %s", r.Namespace)

	catalogSource := &coreosv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.CatalogSourceName(),
			Namespace: r.Namespace,
		},
	}

	catalogSourceSpec := coreosv1alpha1.CatalogSourceSpec{
		SourceType:  coreosv1alpha1.SourceTypeConfigmap,
		ConfigMap:   configMapName,
		DisplayName: r.CatalogSourceName(),
		Publisher:   Publisher,
	}

	or, err := controllerutil.CreateOrUpdate(ctx, r.Client, catalogSource, func() error {
		catalogSource.Spec = catalogSourceSpec
		return nil
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create/update registry catalog source for namespace '%s': %w", r.Namespace, err)
	}

	switch or {
	case controllerutil.OperationResultCreated:
		logrus.Infof("Created registry catalog source for namespace %s", r.Namespace)
	case controllerutil.OperationResultUpdated:
		logrus.Infof("Updated registry catalog source for namespace %s", r.Namespace)
	case controllerutil.OperationResultNone:
		break
	default:
		return reconcile.Result{}, fmt.Errorf("Unknown controllerutil.OperationResult '%v'", or)
	}

	logrus.Infof("Successfully reconciled registry catalog source for namespace %s", r.Namespace)

	return reconcile.Result{}, nil
}
