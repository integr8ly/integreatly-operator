package addon

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	k8sresources "github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RhoamAddonInstallManagedLabel = "rhoam.addon.install/managed"
)

// OperatorRunType is used to indicate how the operator is being currently
// run
type OperatorRunType string

// AddonRunType represents when the operator is run by an add-on installation
var AddonRunType OperatorRunType = "Addon"

// OLMRunType represents when the operator is run by a non-add-on OLM installation
var OLMRunType OperatorRunType = "OLM"

// ClusterRunType represents when the operator is run by a non-OLM in-cluster
// deployment
var ClusterRunType OperatorRunType = "Cluster"

// LocalRunType represents when the operator is run locally
var LocalRunType OperatorRunType = "Local"

var runTypesBySubscription = map[string]OperatorRunType{
	// RHOAM - Add-on
	addonPrefixed(ManagedAPIService): AddonRunType,
	// RHOAM - OLM
	ManagedAPIService: OLMRunType,
	"integreatly":     OLMRunType,
}

// InferOperatorRunType infers how the operator is being run
func InferOperatorRunType(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (OperatorRunType, error) {
	subscription, err := GetSubscription(ctx, client, installation)
	if err != nil {
		return "", err
	}

	if subscription != nil {
		return runTypesBySubscription[subscription.Name], nil
	}

	isCluster, err := IsClusterRunType(ctx, client, installation)
	if err != nil {
		return "", err
	}

	if isCluster {
		return ClusterRunType, nil
	}

	return LocalRunType, nil
}

// GetRhoamOperatorSubscription looks for the managed-api-service operator subscription
// by checking for the standard OLM-generated label on subscriptions
func GetRhoamOperatorSubscription(ctx context.Context, client k8sclient.Client, ns string) (*operatorsv1alpha1.Subscription, error) {
	logrus.Info("Looking for managed-api-service operator subscription")

	// Get all Subscriptions in the operator namespace and check for the OLM-generated label
	subs := &operatorsv1alpha1.SubscriptionList{}
	opts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
	}
	err := client.List(ctx, subs, opts...)
	if err != nil {
		logrus.Errorf("Error getting list of subscriptions %v", err)
		return nil, err
	} else {
		for _, sub := range subs.Items {
			logrus.Infof("Found subscription %s", sub.Name)
			labels := sub.GetLabels()
			// Looking for the OLM-generated label that should be on the managed-api-service subscription
			_, ok := labels["operators.coreos.com/managed-api-service.redhat-rhoam-operator"]
			if ok {
				logrus.Info("Found managed-api-service operator subscription")
				return &sub, nil
			}
		}
	}

	logrus.Info("Did not find managed-api-service operator subscription")
	return nil, nil
}

// OperatorInstalledViaOLM checks if the operator was installed through OLM
func OperatorInstalledViaOLM(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (bool, error) {
	runType, err := InferOperatorRunType(ctx, client, installation)
	if err != nil {
		return false, err
	}

	return runType == OLMRunType, nil
}

// Operator is managed by Hive
func OperatorIsHiveManaged(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (bool, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: installation.Namespace,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		return false, fmt.Errorf("could not retrieve %s namespace: %w", installation.Namespace, err)
	}

	labels := ns.GetLabels()
	value, ok := labels[RhoamAddonInstallManagedLabel]
	if ok {
		if value == "true" {
			logrus.Info("operator is hive managed")
			return true, nil
		}
	}

	return false, nil
}

// IsClusterRunType checks if the operator is run on a cluster
func IsClusterRunType(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (bool, error) {
	deploymentPrefix := "rhoam"

	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-operator", deploymentPrefix),
			Namespace: installation.Namespace,
		},
	}

	return k8sresources.Exists(ctx, client, deployment)
}

// GetSubscription attempts to find the subscription that installed the operator.
// If the subscription is not found, `nil` is returned
func GetSubscription(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (*operatorsv1alpha1.Subscription, error) {
	for subscriptionName := range runTypesBySubscription {
		subscription := &operatorsv1alpha1.Subscription{}
		err := client.Get(ctx, k8sclient.ObjectKey{
			Name:      subscriptionName,
			Namespace: installation.Namespace,
		}, subscription)
		if err != nil && errors.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		return subscription, nil
	}

	// If subscription is not found by known names, try to find it by OLM-generated label
	// This handles cases where the subscription might have a non-deterministic name
	subscription, err := GetRhoamOperatorSubscription(ctx, client, installation.Namespace)
	if err != nil {
		return nil, err
	}
	if subscription != nil {
		return subscription, nil
	}

	return nil, nil
}

// GetCatalogSource attempts to find the CatalogSource that provided the operator
// If the CatalogSource is not found, `nil` is returned
func GetCatalogSource(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (*operatorsv1alpha1.CatalogSource, error) {
	// Attempt to find the Subscription
	subscription, err := GetSubscription(ctx, client, installation)
	if err != nil {
		return nil, err
	}

	// Subscription was not found. There might still be a CatalogSource,
	// but this is not a normal scenario, return `nil`
	if subscription == nil {
		return nil, nil
	}

	// Retrieve the CatalogSource from the Subscription reference
	catalogSource := &operatorsv1alpha1.CatalogSource{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      subscription.Spec.CatalogSource,
		Namespace: subscription.Spec.CatalogSourceNamespace,
	}, catalogSource); err != nil {
		return nil, err
	}

	return catalogSource, nil
}

func addonPrefixed(name string) string {
	return fmt.Sprintf("addon-%s", name)
}
