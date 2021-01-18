package addon

import (
	"context"
	"fmt"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetParameter retrieves the value for an addon parameter by finding the RHMI
// CR and selecting the addon name for its installation type.
func GetParameter(ctx context.Context, client k8sclient.Client, namespace, parameter string) ([]byte, bool, error) {
	rhmi, err := resources.GetRhmiCr(client, ctx, namespace, log)
	if err != nil {
		return nil, false, err
	}

	return GetParameterByInstallType(
		ctx,
		client,
		integreatlyv1alpha1.InstallationType(rhmi.Spec.Type),
		namespace,
		parameter,
	)
}

// GetParameterByInstallType retrieves the value for an addon parameter by
// selecting the addon name for installationType
func GetParameterByInstallType(ctx context.Context, client k8sclient.Client, installationType integreatlyv1alpha1.InstallationType, namespace, parameter string) ([]byte, bool, error) {
	addonName := GetName(installationType)
	secretName := fmt.Sprintf("addon-%s-parameters", addonName)

	secret := &corev1.Secret{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      secretName,
		Namespace: namespace,
	}, secret); err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("failed to retrieve parameters secret: %v", err)
	}

	value, ok := secret.Data[parameter]
	return value, ok, nil
}

// GetStringParameterByInstallType retrieves the string value for an addon
// parameter given the installation type
func GetStringParameterByInstallType(ctx context.Context, client k8sclient.Client, installationType integreatlyv1alpha1.InstallationType, namespace, parameter string) (string, bool, error) {
	value, ok, err := GetParameterByInstallType(ctx, client, installationType, namespace, parameter)
	if err != nil || !ok {
		return "", ok, err
	}

	return string(value), ok, nil
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
