package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/custom-smtp"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	alertManagerConfigSecretFileName = "alertmanager.yaml"
	alertManagerConfigSecretName     = "alertmanager-application-monitoring"
)

type alertManagerConfig struct {
	Global map[string]string `yaml:"global"`
}

func GetExistingSMTPFromAddress(ctx context.Context, client k8sclient.Client, ns string) (string, error) {
	amSecret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      alertManagerConfigSecretName,
		Namespace: ns,
	}, amSecret)
	if err != nil {
		return "", err
	}
	monitoring, ok := amSecret.Data[alertManagerConfigSecretFileName]
	if !ok {
		return "", fmt.Errorf("failed to find %s in %s secret data", alertManagerConfigSecretFileName, alertManagerConfigSecretName)
	}
	var config alertManagerConfig
	err = yaml.Unmarshal(monitoring, &config)
	if err != nil {
		return "", fmt.Errorf("failed to parse alert monitoring yaml: %w", err)
	}
	smtpFrom, ok := config.Global["smtp_from"]
	if !ok {
		return "", fmt.Errorf("failed to find smtp_from in alert manager config map")
	}
	return smtpFrom, nil
}

// GetSMTPFromAddress returns the correct from address depending on how the operator is configured
// For addon installs returns the address stated in the alertmanger.yaml or the address configured by the custom SMTP feature in ocm
// For sandbox it returns the default hardcoded value of: test@rhmw.io
func GetSMTPFromAddress(ctx context.Context, serverClient k8sclient.Client, log logger.Logger, installation *v1alpha1.RHMI) (string, error) {

	if installation == nil {
		return "", fmt.Errorf("nil pointer passed for installation")
	}

	if v1alpha1.IsRHOAMMultitenant(v1alpha1.InstallationType(installation.Spec.Type)) {
		return "test@rhmw.io", nil
	}

	var existingSMTPFromAddress string
	var err error

	if installation.Status.CustomSmtp != nil && installation.Status.CustomSmtp.Enabled {
		existingSMTPFromAddress, err = custom_smtp.GetFromAddress(ctx, serverClient, installation.Namespace)

		if err != nil {
			log.Error("error getting smtp_from address from custom smtp secret", err)
			return "", err
		}
	} else {
		existingSMTPFromAddress, err = GetExistingSMTPFromAddress(ctx, serverClient, installation.Namespace)

		if err != nil {
			if !errors.IsNotFound(err) {
				log.Error("Error getting smtp_from address from secret alertmanager-application-monitoring", err)
				return "", err
			}
			log.Warning("failure finding secret alertmanager-application-monitoring: " + err.Error())
		}
	}

	if existingSMTPFromAddress == "" {
		log.Warning("Couldn't find SMTP in a secret, retrieving it from the envar")
		existingSMTPFromAddress = os.Getenv(v1alpha1.EnvKeyAlertSMTPFrom)
	}

	return existingSMTPFromAddress, nil
}
