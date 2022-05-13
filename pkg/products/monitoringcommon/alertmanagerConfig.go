package monitoringcommon

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	v1 "github.com/openshift/api/route/v1"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

const (
	// Cluster infrastructure
	clusterInfraName = "cluster"

	// For Cluster ID
	clusterVersionName = "version"

	// For OpenShift console
	openShiftConsoleRoute     = "console"
	openShiftConsoleNamespace = "openshift-console"
)

func getSmtpHost(smtpSecret *corev1.Secret) string {
	host := "smtp.example.com"
	if smtpSecret.Data != nil && string(smtpSecret.Data["host"]) != "" {
		host = string(smtpSecret.Data["host"])
	}
	return host
}

func getSmtpPort(smtpSecret *corev1.Secret) string {
	port := "587"
	if smtpSecret.Data != nil && string(smtpSecret.Data["port"]) != "" {
		port = string(smtpSecret.Data["port"])
	}
	return port
}

func getSmtpUsername(smtpSecret *corev1.Secret) string {
	username := "smtp_username"
	if smtpSecret.Data != nil && string(smtpSecret.Data["username"]) != "" {
		username = string(smtpSecret.Data["username"])
	}
	return username
}

func getSmtpPassword(smtpSecret *corev1.Secret) string {
	password := "smtp_password"
	if smtpSecret.Data != nil && string(smtpSecret.Data["password"]) != "" {
		password = string(smtpSecret.Data["password"])
	}
	return password
}

func ReconcileAlertManagerSecrets(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI, productNamespace string, alertManagerRouteName string) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()

	log.Info("reconciling alertmanager configuration secret")

	integreatlyOperatorNs := installation.Namespace

	// handle alert manager route
	alertmanagerRoute := &v1.Route{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: alertManagerRouteName, Namespace: productNamespace}, alertmanagerRoute); err != nil {
		if k8serr.IsNotFound(err) {
			log.Infof("alert manager route not available, cannot create alert manager config secret", l.Fields{"route": alertManagerRouteName})
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not obtain alert manager route: %w", err)
	}

	// handle smtp credentials
	smtpSecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: installation.Spec.SMTPSecret, Namespace: integreatlyOperatorNs}, smtpSecret); err != nil {
		log.Warningf("Could not obtain smtp credentials secret", l.Fields{"error": err.Error()})
	}

	//Get pagerduty credentials
	pagerDutySecret, err := getPagerDutySecret(ctx, serverClient, *installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	//Get dms credentials
	dmsSecret, err := getDMSSecret(ctx, serverClient, *installation)
	if err != nil {
		log.Warningf("Could not get DMS secret", l.Fields{"error": err.Error()})
	}

	// only set the to address to a real value for managed deployments
	smtpToSREAddress := fmt.Sprintf("noreply@%s", alertmanagerRoute.Spec.Host)
	smtpToSREAddressCRVal := installation.Spec.AlertingEmailAddresses.CSSRE
	if smtpToSREAddressCRVal != "" {
		smtpToSREAddress = smtpToSREAddressCRVal
	}

	smtpToBUAddress := fmt.Sprintf("noreply@%s", alertmanagerRoute.Spec.Host)
	smtpToBUAddressCRVal := installation.Spec.AlertingEmailAddresses.BusinessUnit
	if smtpToBUAddressCRVal != "" {
		smtpToBUAddress = smtpToBUAddressCRVal
	}

	smtpToCustomerAddress := fmt.Sprintf("noreply@%s", alertmanagerRoute.Spec.Host)
	smtpToCustomerAddressCRVal := installation.Spec.AlertingEmailAddress
	if smtpToCustomerAddressCRVal != "" {
		smtpToCustomerAddress = prepareEmailAddresses(smtpToCustomerAddressCRVal)
	}

	var existingSMTPFromAddress = ""
	if installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManaged) {
		existingSMTPFromAddress, err = resources.GetExistingSMTPFromAddress(ctx, serverClient, productNamespace)
		if err != nil {
			if !apiErrors.IsNotFound(err) {
				log.Error("Error getting application monitoring secret", err)
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to fetch get application monitoring secret: %w", err)
			}
		}
	}

	smtpAlertFromAddress := os.Getenv(integreatlyv1alpha1.EnvKeyAlertSMTPFrom)

	// If SMTPFromAddress already set for RHMI prod customers, do not reset
	if installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManaged) && existingSMTPFromAddress != "" {
		log.Infof("setting smtpAlertFromAddress to existing value ", l.Fields{"FromAddress": existingSMTPFromAddress})
		smtpAlertFromAddress = existingSMTPFromAddress
	} else if installation.Spec.AlertFromAddress != "" {
		smtpAlertFromAddress = installation.Spec.AlertFromAddress
	}

	clusterInfra := &configv1.Infrastructure{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: clusterInfraName}, clusterInfra); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to fetch cluster infra details for alertmanager config: %w", err)
	}

	clusterVersion := &configv1.ClusterVersion{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: clusterVersionName}, clusterVersion); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to fetch cluster ID details for alertmanager config: %w", err)
	}

	clusterRoute := &routev1.Route{}
	if err := serverClient.Get(context.TODO(), types.NamespacedName{Name: openShiftConsoleRoute, Namespace: openShiftConsoleNamespace}, clusterRoute); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to fetch OpenShift console URL details for alertmanager config: %w", err)
	}

	clusterConsoleRoute := fmt.Sprintf(`https://%v`, clusterRoute.Spec.Host)
	clusterName := clusterInfra.Status.InfrastructureName
	clusterID := string(clusterVersion.Spec.ClusterID)

	// parse the config template into a secret object
	templateUtil := NewTemplateHelper(map[string]string{
		"SMTPHost":              getSmtpHost(smtpSecret),
		"SMTPPort":              getSmtpPort(smtpSecret),
		"SMTPFrom":              smtpAlertFromAddress,
		"SMTPUsername":          getSmtpUsername(smtpSecret),
		"SMTPPassword":          getSmtpPassword(smtpSecret),
		"SMTPToCustomerAddress": smtpToCustomerAddress,
		"SMTPToSREAddress":      smtpToSREAddress,
		"SMTPToBUAddress":       smtpToBUAddress,
		"PagerDutyServiceKey":   pagerDutySecret,
		"DeadMansSnitchURL":     dmsSecret,
		"Subject":               fmt.Sprintf(`{{template "email.integreatly.subject" . }}`),
		"clusterID":             clusterID,
		"clusterName":           clusterName,
		"clusterConsole":        clusterConsoleRoute,
		"html":                  fmt.Sprintf(`{{ template "email.integreatly.html" . }}`),
	})

	templatePath := GetTemplatePath()
	path := fmt.Sprintf("%s/%s", templatePath, config.AlertManagerCustomTemplatePath)

	// generate alertmanager custom email template
	emailConfigContents, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not read alertmanager custom email template file: %w", err)
	}

	emailConfigContentsStr := string(emailConfigContents)
	cluster_vars := map[string]string{
		"${CLUSTER_NAME}":    clusterName,
		"${CLUSTER_ID}":      clusterID,
		"${CLUSTER_CONSOLE}": clusterConsoleRoute,
	}

	for name, val := range cluster_vars {
		emailConfigContentsStr = strings.ReplaceAll(emailConfigContentsStr, name, val)
	}

	configSecretData, err := templateUtil.LoadTemplate(config.AlertManagerConfigTemplatePath)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not parse alert manager configuration template: %w", err)
	}
	configSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.AlertManagerConfigSecretName,
			Namespace: productNamespace,
		},
		Type: corev1.SecretTypeOpaque,
	}

	// create the config secret
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, configSecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(configSecret, installation)
		configSecret.Data = map[string][]byte{
			config.AlertManagerConfigSecretFileName:        configSecretData,
			config.AlertManagerEmailTemplateSecretFileName: []byte(emailConfigContentsStr),
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create or update alert manager secret: %w", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getPagerDutySecret(ctx context.Context, serverClient k8sclient.Client, installation integreatlyv1alpha1.RHMI) (string, error) {

	var secret string

	pagerdutySecret := &corev1.Secret{}
	err := serverClient.Get(ctx, types.NamespacedName{Name: installation.Spec.PagerDutySecret,
		Namespace: installation.Namespace}, pagerdutySecret)

	if err != nil {
		return "", fmt.Errorf("could not obtain pagerduty credentials secret: %w", err)
	}

	if len(pagerdutySecret.Data["PAGERDUTY_KEY"]) != 0 {
		secret = string(pagerdutySecret.Data["PAGERDUTY_KEY"])
	} else if len(pagerdutySecret.Data["serviceKey"]) != 0 {
		secret = string(pagerdutySecret.Data["serviceKey"])
	}

	if secret == "" {
		return "", fmt.Errorf("secret key is undefined in pager duty secret")
	}

	return secret, nil
}

func getDMSSecret(ctx context.Context, serverClient k8sclient.Client, installation integreatlyv1alpha1.RHMI) (string, error) {

	var secret string

	dmsSecret := &corev1.Secret{}
	err := serverClient.Get(ctx, types.NamespacedName{Name: installation.Spec.DeadMansSnitchSecret,
		Namespace: installation.Namespace}, dmsSecret)

	if err != nil {
		return "", fmt.Errorf("could not obtain dead mans snitch credentials secret: %w", err)
	}

	if len(dmsSecret.Data["SNITCH_URL"]) != 0 {
		secret = string(dmsSecret.Data["SNITCH_URL"])
	} else if len(dmsSecret.Data["url"]) != 0 {
		secret = string(dmsSecret.Data["url"])
	} else {
		return "", fmt.Errorf("url is undefined in dead mans snitch secret")
	}

	return secret, nil
}

// prepareEmailAddresses converts a space separated string into a comma separated
// string. Example:
//
// "foo@example.org bar@example.org" -> "foo@example.org, bar@example.org"
func prepareEmailAddresses(list string) string {
	addresses := strings.Split(strings.TrimSpace(list), " ")
	return strings.Join(addresses, ", ")
}
