package openshift

import (
	"context"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"

	"strconv"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	smtpPortPlaceholder = 587
	smtpTLSPlaceholder  = true
)

var _ providers.SMTPCredentialsProvider = (*SMTPCredentialProvider)(nil)

type SMTPCredentialProvider struct {
	Client client.Client
	Logger *logrus.Entry
}

func NewSMTPCredentialSetProvider(c client.Client, l *logrus.Entry) *SMTPCredentialProvider {
	return &SMTPCredentialProvider{
		Client: c,
		Logger: l,
	}
}

func (s SMTPCredentialProvider) GetName() string {
	return "openshift-smtp"
}

func (s SMTPCredentialProvider) SupportsStrategy(str string) bool {
	return providers.OpenShiftDeploymentStrategy == str
}

func (s SMTPCredentialProvider) GetReconcileTime(smtpCreds *v1alpha1.SMTPCredentialSet) time.Duration {
	return time.Second * 10
}

func (s SMTPCredentialProvider) CreateSMTPCredentials(ctx context.Context, smtpCreds *v1alpha1.SMTPCredentialSet) (*providers.SMTPCredentialSetInstance, types.StatusMessage, error) {
	dd := &aws.SMTPCredentialSetDetails{
		Username: varPlaceholder,
		Password: varPlaceholder,
		Port:     smtpPortPlaceholder,
		Host:     varPlaceholder,
		TLS:      smtpTLSPlaceholder,
	}

	if smtpCreds.Spec.SecretRef.Namespace == "" {
		smtpCreds.Spec.SecretRef.Namespace = smtpCreds.Namespace
	}

	if smtpCreds.Status.Phase != types.PhaseComplete || smtpCreds.Status.SecretRef.Name == "" || smtpCreds.Status.SecretRef.Namespace == "" {
		return &providers.SMTPCredentialSetInstance{
			DeploymentDetails: dd,
		}, "reconcile complete", nil
	}
	sec := &v1.Secret{}
	if err := s.Client.Get(ctx, client.ObjectKey{Name: smtpCreds.Status.SecretRef.Name, Namespace: smtpCreds.Status.SecretRef.Namespace}, sec); err != nil {
		return nil, "failed to reconcile", err
	}
	dd.Host = resources.StringOrDefault(string(sec.Data[aws.DetailsSMTPHostKey]), varPlaceholder)
	dd.Password = resources.StringOrDefault(string(sec.Data[aws.DetailsSMTPPasswordKey]), varPlaceholder)
	dd.Username = resources.StringOrDefault(string(sec.Data[aws.DetailsSMTPUsernameKey]), varPlaceholder)
	ddPortStr := resources.StringOrDefault(string(sec.Data[aws.DetailsSMTPPortKey]), strconv.Itoa(smtpPortPlaceholder))
	ddPort, err := strconv.Atoi(ddPortStr)
	if err != nil {
		ddPort = smtpPortPlaceholder
	}
	ddTLSStr := resources.StringOrDefault(string(sec.Data[aws.DetailsSMTPTLSKey]), strconv.FormatBool(smtpTLSPlaceholder))
	ddTLS, err := strconv.ParseBool(ddTLSStr)
	if err != nil {
		ddTLS = smtpTLSPlaceholder
	}
	dd.Port = ddPort
	dd.TLS = ddTLS
	return &providers.SMTPCredentialSetInstance{
		DeploymentDetails: dd,
	}, "reconcile complete", nil
}

func (s SMTPCredentialProvider) DeleteSMTPCredentials(ctx context.Context, smtpCreds *v1alpha1.SMTPCredentialSet) (types.StatusMessage, error) {
	return "deletion complete", nil
}
