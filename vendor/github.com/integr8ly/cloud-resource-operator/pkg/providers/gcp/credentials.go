package gcp

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	errorUtil "github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultCredentialsServiceAccount = "service_account.json"
	defaultProviderCredentialName    = "cloud-resource-gcp-credentials"
)

var (
	operatorRoles = []string{
		"roles/compute.networkAdmin",
		"roles/storage.admin",
		"roles/redis.admin",
		"roles/cloudsql.admin",
		"roles/servicenetworking.networksAdmin",
		"roles/monitoring.viewer",
	}
	timeOut = time.Minute * 5
)

type Credentials struct {
	ServiceAccountID   string
	ServiceAccountJson []byte
}

//go:generate moq -out credentials_moq.go . CredentialManager
type CredentialManager interface {
	ReconcileProviderCredentials(ctx context.Context, ns string) (*Credentials, error)
	ReconcileCredentials(ctx context.Context, name string, ns string, roles []string) (*v1.CredentialsRequest, *Credentials, error)
}

type CredentialMinterCredentialManager struct {
	ProviderCredentialName string
	Client                 client.Client
}

func NewCredentialMinterCredentialManager(client client.Client) *CredentialMinterCredentialManager {
	return &CredentialMinterCredentialManager{
		ProviderCredentialName: defaultProviderCredentialName,
		Client:                 client,
	}
}

func (m *CredentialMinterCredentialManager) ReconcileProviderCredentials(ctx context.Context, ns string) (*Credentials, error) {
	_, creds, err := m.ReconcileCredentials(ctx, m.ProviderCredentialName, ns, operatorRoles)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

func (m *CredentialMinterCredentialManager) ReconcileCredentials(ctx context.Context, name string, ns string, roles []string) (*v1.CredentialsRequest, *Credentials, error) {
	cr, err := m.reconcileCredentialRequest(ctx, name, ns, roles)
	if err != nil {
		return nil, nil, errorUtil.Wrapf(err, "failed to reconcile gcp credential request %s", name)
	}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, timeOut, true, func(ctx2 context.Context) (bool, error) {
		if err = m.Client.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return cr.Status.Provisioned, nil
	})
	if err != nil {
		return nil, nil, errorUtil.Wrap(err, "timed out waiting for credential request to become provisioned")
	}
	gcpProvStatus := &v1.GCPProviderStatus{}
	if err = v1.Codec.DecodeProviderSpec(cr.Status.ProviderStatus, gcpProvStatus); err != nil {
		return nil, nil, errorUtil.Wrapf(err, "failed to decode credentials request %s", cr.Name)
	}
	gcpServiceAccountJson, err := m.reconcileGCPCredentials(ctx, cr)
	if err != nil {
		return nil, nil, errorUtil.Wrapf(err, "failed to reconcile gcp credentials from credential request %s", cr.Name)
	}
	return cr, &Credentials{
		ServiceAccountID:   gcpProvStatus.ServiceAccountID,
		ServiceAccountJson: gcpServiceAccountJson,
	}, nil
}

var _ CredentialManager = (*CredentialMinterCredentialManager)(nil)

func (m *CredentialMinterCredentialManager) reconcileCredentialRequest(ctx context.Context, name string, ns string, roles []string) (*v1.CredentialsRequest, error) {
	providerSpec, err := v1.Codec.EncodeProviderSpec(&v1.GCPProviderSpec{
		TypeMeta: controllerruntime.TypeMeta{
			Kind: "GCPProviderSpec",
		},
		PredefinedRoles:  roles,
		SkipServiceCheck: true,
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to encode provider spec")
	}
	cr := &v1.CredentialsRequest{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, m.Client, cr, func() error {
		cr.Spec.ProviderSpec = providerSpec
		cr.Spec.SecretRef = v12.ObjectReference{
			Name:      name,
			Namespace: ns,
		}
		return nil
	})
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to reconcile credential request %s in namespace %s", cr.Name, cr.Namespace)
	}
	return cr, nil
}

func (m *CredentialMinterCredentialManager) reconcileGCPCredentials(ctx context.Context, cr *v1.CredentialsRequest) ([]byte, error) {
	sec := &v12.Secret{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: cr.Spec.SecretRef.Name, Namespace: cr.Spec.SecretRef.Namespace}, sec)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get gcp credentials secret %s", cr.Spec.SecretRef.Name)
	}
	gcpServiceAccount := sec.Data[defaultCredentialsServiceAccount]
	if len(gcpServiceAccount) == 0 {
		return nil, errorUtil.New(fmt.Sprintf("gcp service account is undefined in secret %s", sec.Name))
	}
	return gcpServiceAccount, nil
}
