package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/custom-smtp"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"os"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	alertManagerConfigSecretName = "alertmanager-application-monitoring"
)

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
		existingSMTPFromAddress = installation.Spec.AlertFromAddress
	}

	if existingSMTPFromAddress == "" {
		log.Warning("Couldn't find SMTP in installation spec or custom domain secret, retrieving it from the envar")
		existingSMTPFromAddress = os.Getenv(v1alpha1.EnvKeyAlertSMTPFrom)
	}

	return existingSMTPFromAddress, nil
}
