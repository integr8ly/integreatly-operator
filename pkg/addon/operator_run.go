package addon

import (
	"context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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

var runTypesBySubscription map[string]OperatorRunType = map[string]OperatorRunType{
	// RHOAM - Add-on
	addonPrefixed(ManagedAPIService): AddonRunType,
	// RHOAM - OLM
	ManagedAPIService: OLMRunType,

	// RHMI - Add-on
	addonPrefixed(RHMI): AddonRunType,
	// RHMI - OLM
	"integreatly": OLMRunType,
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

// OperatorInstalledViaOLM checks if the operator was installed through OLM
func OperatorInstalledViaOLM(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (bool, error) {
	runType, err := InferOperatorRunType(ctx, client, installation)
	if err != nil {
		return false, err
	}

	return runType == AddonRunType || runType == OLMRunType, nil
}

// IsClusterRunType checks if the operator is run on a cluster
func IsClusterRunType(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (bool, error) {
	deploymentPrefix := "rhmi"
	if installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		deploymentPrefix = "rhoam"
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-operator", deploymentPrefix),
			Namespace: installation.Namespace,
		},
	}

	return resources.Exists(ctx, client, deployment)
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

	return nil, nil
}

func addonPrefixed(name string) string {
	return fmt.Sprintf("addon-%s", name)
}
