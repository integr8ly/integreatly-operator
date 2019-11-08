package aws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"

	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	smtpCredentialProviderName = "aws-ses"
	// default create options
	detailsSMTPUsernameKey = "username"
	detailsSMTPPasswordKey = "password"
	detailsSMTPPortKey     = "port"
	detailsSMTPHostKey     = "host"
	detailsSMTPTLSKey      = "tls"
)

// SMTPCredentialSetDetails Provider-specific details about SMTP credentials derived from an AWS IAM role
type SMTPCredentialSetDetails struct {
	Username string
	Password string
	Port     int
	Host     string
	TLS      bool
}

func (d *SMTPCredentialSetDetails) Data() map[string][]byte {
	return map[string][]byte{
		detailsSMTPUsernameKey: []byte(d.Username),
		detailsSMTPPasswordKey: []byte(d.Password),
		detailsSMTPPortKey:     []byte(strconv.Itoa(d.Port)),
		detailsSMTPHostKey:     []byte(d.Host),
		detailsSMTPTLSKey:      []byte(strconv.FormatBool(d.TLS)),
	}
}

var _ providers.SMTPCredentialsProvider = (*SMTPCredentialProvider)(nil)

type SMTPCredentialProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewAWSSMTPCredentialProvider(client client.Client, logger *logrus.Entry) *SMTPCredentialProvider {
	return &SMTPCredentialProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": smtpCredentialProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}
}

func (p *SMTPCredentialProvider) GetName() string {
	return smtpCredentialProviderName
}

func (p *SMTPCredentialProvider) SupportsStrategy(d string) bool {
	p.Logger.Infof("checking for support of strategy %s, supported strategies are %s", d, providers.AWSDeploymentStrategy)
	return providers.AWSDeploymentStrategy == d
}

func (p *SMTPCredentialProvider) GetReconcileTime(smtpCreds *v1alpha1.SMTPCredentialSet) time.Duration {
	if smtpCreds.Status.Phase != v1alpha1.PhaseComplete {
		return time.Second * 30
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (p *SMTPCredentialProvider) CreateSMTPCredentials(ctx context.Context, smtpCreds *v1alpha1.SMTPCredentialSet) (*providers.SMTPCredentialSetInstance, v1alpha1.StatusMessage, error) {
	p.Logger.Infof("creating smtp credential instance %s via aws ses", smtpCreds.Name)

	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, smtpCreds, DefaultFinalizer); err != nil {
		errMsg := "failed to set finalizer"
		return nil, v1alpha1.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// retrieve deployment strategy for provided tier
	p.Logger.Infof("getting credential set strategy from aws config")
	stratCfg, err := p.ConfigManager.ReadSMTPCredentialSetStrategy(ctx, smtpCreds.Spec.Tier)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read deployment strategy for smtp credential instance %s", smtpCreds.Name)
		return nil, v1alpha1.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	awsRegion := stratCfg.Region
	if awsRegion == "" {
		awsRegion = DefaultRegion
	}
	sesSMTPHost := p.ConfigManager.GetDefaultRegionSMTPServerMapping()[awsRegion]
	if sesSMTPHost == "" {
		errMsg := fmt.Sprintf("unsupported aws ses smtp region %s", sesSMTPHost)
		return nil, v1alpha1.StatusMessage(errMsg), errorUtil.New(errMsg)
	}

	// create smtp credentials from generated iam role
	p.Logger.Info("creating iam role required to send mail through aws ses")
	credSecName, err := buildInfraNameFromObject(ctx, p.Client, smtpCreds.ObjectMeta, 40)
	if err != nil {
		msg := "failed to generate smtp credentials secret name"
		return nil, v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	sendMailCreds, err := p.CredentialManager.ReconcileSESCredentials(ctx, credSecName, smtpCreds.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create aws ses credentials request for smtp credentials instance %s", smtpCreds.Name)
		return nil, v1alpha1.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	p.Logger.Info("creating smtp credentials from created iam role")
	smtpPass, err := getSMTPPasswordFromAWSSecret(sendMailCreds.SecretAccessKey)
	if err != nil {
		errMsg := "failed to create smtp credentials from aws iam role"
		return nil, v1alpha1.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	// hardcoded settings based on https://docs.aws.amazon.com/ses/latest/DeveloperGuide/configure-email-client.html
	smtpCredsInst := &providers.SMTPCredentialSetInstance{
		DeploymentDetails: &SMTPCredentialSetDetails{
			Username: sendMailCreds.AccessKeyID,
			Password: smtpPass,
			Port:     465,
			Host:     sesSMTPHost,
			TLS:      true,
		},
	}

	p.Logger.Infof("creation handler for smtp credential instance %s in namespace %s finished successfully", smtpCreds.Name, smtpCreds.Namespace)
	return smtpCredsInst, "creation successful", nil
}

func (p *SMTPCredentialProvider) DeleteSMTPCredentials(ctx context.Context, smtpCreds *v1alpha1.SMTPCredentialSet) (v1alpha1.StatusMessage, error) {
	// remove the credentials request created by the provider
	endUserCredsReq := &v1.CredentialsRequest{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      smtpCreds.Name,
			Namespace: smtpCreds.Namespace,
		},
	}
	if err := p.Client.Delete(ctx, endUserCredsReq); err != nil && !errors.IsNotFound(err) {
		errMsg := fmt.Sprintf("failed to delete credential request %s", smtpCreds.Name)
		return v1alpha1.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// remove the finalizer added by the provider
	p.Logger.Infof("deleting finalizer %s from smtp credentials %s in namespace %s", DefaultFinalizer, smtpCreds.Name, smtpCreds.Namespace)
	resources.RemoveFinalizer(&smtpCreds.ObjectMeta, DefaultFinalizer)
	if err := p.Client.Update(ctx, smtpCreds); err != nil {
		errMsg := fmt.Sprintf("failed to update instance %s as part of finalizer reconcile", smtpCreds.Name)
		return v1alpha1.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	p.Logger.Infof("deletion handler for smtp credentials %s in namespace %s finished successfully", smtpCreds.Name, smtpCreds.Namespace)
	return "deletion complete", nil
}

// https://docs.aws.amazon.com/ses/latest/DeveloperGuide/example-create-smtp-credentials.html
func getSMTPPasswordFromAWSSecret(secAccessKey string) (string, error) {
	sig, err := makeHmac([]byte(secAccessKey), []byte("SendRawEmail"))
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to create hmac using ami secret")
	}
	sig = append([]byte{0x02}, sig...)
	return base64.StdEncoding.EncodeToString(sig), nil
}

func makeHmac(key []byte, data []byte) ([]byte, error) {
	hash := hmac.New(sha256.New, key)
	if _, err := hash.Write(data); err != nil {
		return nil, errorUtil.Wrap(err, "failed to populate hash")
	}
	return hash.Sum(nil), nil
}
