package addon

import (
	"context"
	"fmt"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"regexp"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	QuotaParamName      = "addon-managed-api-service"
	TrialQuotaParamName = "trial-quota"
	ManagedAPIService   = "managed-api-service"
	RHMI                = "rhmi"
	DefaultSecretName   = "addon-managed-api-service-parameters"
)

var (
	log = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "addon"})
)

// GetParameter retrieves the value for an addon parameter by finding the Subscription
// CR and selecting the addon name from a secret associated with it.
func GetParameter(ctx context.Context, client k8sclient.Client, namespace, parameter string) ([]byte, bool, error) {
	secret, err := GetAddonParametersSecret(ctx, client, namespace)
	if err != nil {
		return nil, false, err
	}

	value, ok := secret.Data[parameter]
	return value, ok, nil
}

// GetAddonParametersSecret retrieves addon parameters secret, provided operator namespace.
func GetAddonParametersSecret(ctx context.Context, client k8sclient.Client, namespace string) (*corev1.Secret, error) {
	parametersSecretName := DefaultSecretName

	opts := &k8sclient.ListOptions{
		Namespace: namespace,
	}
	subscriptions := &v1alpha1.SubscriptionList{}
	if err := client.List(ctx, subscriptions, opts); err != nil {
		return nil, err
	}
	if len(subscriptions.Items) > 1 {
		return nil, fmt.Errorf("received %d subscriptions in %s namespace. Expected one", len(subscriptions.Items), namespace)
	}
	if len(subscriptions.Items) == 1 {
		parametersSecretName = subscriptions.Items[0].Name + "-parameters"

		// catch olm and sandbox installations
		hasAddonPrefix, err := regexp.MatchString("^addon-*", parametersSecretName)
		if err != nil {
			return nil, err
		}
		if !hasAddonPrefix {
			parametersSecretName = "addon-" + parametersSecretName
		}
	}

	secret := &corev1.Secret{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: namespace,
		Name:      parametersSecretName,
	}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// GetStringParameter retrieves the string value for an addon parameter
func GetStringParameter(ctx context.Context, client k8sclient.Client, namespace, parameter string) (string, bool, error) {
	value, ok, err := GetParameter(ctx, client, namespace, parameter)
	if err != nil || !ok {
		return "", ok, err
	}

	return string(value), ok, nil
}

// GetIntParameter retrieves the integer value for an addon parameter
func GetIntParameter(ctx context.Context, client k8sclient.Client, namespace, parameter string) (int, bool, error) {
	value, ok, err := GetStringParameter(ctx, client, namespace, parameter)
	if err != nil || !ok {
		return 0, ok, err
	}

	valueInt, err := strconv.Atoi(value)
	return valueInt, ok, err
}

// GetBoolParameter retrieves the boolean value for an addon parameter
func GetBoolParameter(ctx context.Context, client k8sclient.Client, namespace, parameter string) (bool, bool, error) {
	value, ok, err := GetStringParameter(ctx, client, namespace, parameter)
	if err != nil || !ok {
		return false, ok, err
	}

	valueBool, err := strconv.ParseBool(value)
	return valueBool, ok, err
}

// ExistsParameterByInstallation checks for existence of given parameter in parameters secret
func ExistsParameterByInstallation(ctx context.Context, client k8sclient.Client, install *integreatlyv1alpha1.RHMI, parameter string) (bool, error) {
	_, found, err := GetParameter(ctx, client, install.Namespace, parameter)
	return found, err
}
