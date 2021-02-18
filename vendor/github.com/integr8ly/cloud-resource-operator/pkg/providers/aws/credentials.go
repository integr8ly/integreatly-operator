package aws

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	errorUtil "github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultProviderCredentialName = "cloud-resources-aws-credentials"

	defaultCredentialsKeyIDName = "aws_access_key_id"
	// #nosec G101
	defaultCredentialsSecretKeyName = "aws_secret_access_key"
)

var (
	operatorEntries = []v1.StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"s3:CreateBucket",
				"s3:DeleteBucket",
				"s3:ListBucket",
				"s3:ListAllMyBuckets",
				"s3:GetObject",
				"s3:DeleteObject",
				"s3:PutBucketTagging",
				"s3:PutBucketPublicAccessBlock",
				"s3:PutEncryptionConfiguration",
				"ec2:DescribeVpcs",
				"ec2:DescribeSubnets",
				"ec2:DescribeSecurityGroups",
				"ec2:DescribeInstanceTypes",
				"ec2:CreateSecurityGroup",
				"ec2:DeleteSecurityGroup",
				"ec2:AuthorizeSecurityGroupIngress",
				"ec2:AuthorizeSecurityGroupEgress",
				"ec2:DescribeAvailabilityZones",
				"ec2:CreateSubnet",
				"ec2:CreateTags",
				"ec2:CreateVpc",
				"ec2:DeleteVpc",
				"ec2:DeleteSubnet",
				"ec2:CreateVpcPeeringConnection",
				"ec2:DescribeVpcPeeringConnections",
				"ec2:AcceptVpcPeeringConnection",
				"ec2:DeleteVpcPeeringConnection",
				"ec2:DescribeRouteTables",
				"ec2:CreateRoute",
				"ec2:DeleteRoute",
				"ec2:DescribeInstanceTypeOfferings",
				"elasticache:CreateReplicationGroup",
				"elasticache:DeleteReplicationGroup",
				"elasticache:DescribeReplicationGroups",
				"elasticache:DescribeServiceUpdates",
				"elasticache:AddTagsToResource",
				"elasticache:DescribeSnapshots",
				"elasticache:CreateSnapshot",
				"elasticache:DeleteSnapshot",
				"elasticache:DescribeCacheClusters",
				"elasticache:DescribeCacheSubnetGroups",
				"elasticache:CreateCacheSubnetGroup",
				"elasticache:ModifyCacheSubnetGroup",
				"elasticache:DeleteCacheSubnetGroup",
				"elasticache:ModifyReplicationGroup",
				"rds:DescribeDBInstances",
				"rds:CreateDBInstance",
				"rds:DeleteDBInstance",
				"rds:ModifyDBInstance",
				"rds:AddTagsToResource",
				"rds:DescribeDBSnapshots",
				"rds:CreateDBSnapshot",
				"rds:DeleteDBSnapshot",
				"rds:DescribePendingMaintenanceActions",
				"rds:CreateDBSubnetGroup",
				"rds:DescribeDBSubnetGroups",
				"rds:DeleteDBSubnetGroup",
				"rds:ModifyDBSubnetGroup",
				"rds:ListTagsForResource",
				"rds:RemoveTagsFromResource",
				"sts:GetCallerIdentity",
				"iam:CreateServiceLinkedRole",
				"cloudwatch:ListMetrics",
				"cloudwatch:GetMetricData",
			},
			Resource: "*",
		},
	}
	sendRawMailEntries = []v1.StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"ses:SendRawEmail",
			},
			Resource: "*",
		},
	}
)

func buildPutBucketObjectEntries(bucket string) []v1.StatementEntry {
	return []v1.StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"s3:*",
			},
			Resource: fmt.Sprintf("arn:aws:s3:::%s", bucket),
		},
		{
			Effect: "Allow",
			Action: []string{
				"s3:*",
			},
			Resource: fmt.Sprintf("arn:aws:s3:::%s/*", bucket),
		},
	}
}

type Credentials struct {
	Username        string
	PolicyName      string
	AccessKeyID     string
	SecretAccessKey string
}

//go:generate moq -out credentials_moq.go . CredentialManager
type CredentialManager interface {
	ReconcileProviderCredentials(ctx context.Context, ns string) (*Credentials, error)
	ReconcileSESCredentials(ctx context.Context, name, ns string) (*Credentials, error)
	ReoncileBucketOwnerCredentials(ctx context.Context, name, ns, bucket string) (*Credentials, *v1.CredentialsRequest, error)
	ReconcileCredentials(ctx context.Context, name string, ns string, entries []v1.StatementEntry) (*v1.CredentialsRequest, *Credentials, error)
}

var _ CredentialManager = (*CredentialMinterCredentialManager)(nil)

// CredentialMinterCredentialManager Implementation of CredentialManager using the openshift cloud credential minter
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

//ReconcileProviderCredentials Ensure the credentials the AWS provider requires are available
func (m *CredentialMinterCredentialManager) ReconcileProviderCredentials(ctx context.Context, ns string) (*Credentials, error) {
	_, creds, err := m.ReconcileCredentials(ctx, m.ProviderCredentialName, ns, operatorEntries)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

func (m *CredentialMinterCredentialManager) ReconcileSESCredentials(ctx context.Context, name, ns string) (*Credentials, error) {
	_, creds, err := m.ReconcileCredentials(ctx, name, ns, sendRawMailEntries)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

func (m *CredentialMinterCredentialManager) ReoncileBucketOwnerCredentials(ctx context.Context, name, ns, bucket string) (*Credentials, *v1.CredentialsRequest, error) {
	cr, creds, err := m.ReconcileCredentials(ctx, name, ns, buildPutBucketObjectEntries(bucket))
	if err != nil {
		return nil, nil, err
	}
	return creds, cr, nil
}

func (m *CredentialMinterCredentialManager) ReconcileCredentials(ctx context.Context, name string, ns string, entries []v1.StatementEntry) (*v1.CredentialsRequest, *Credentials, error) {
	cr, err := m.reconcileCredentialRequest(ctx, name, ns, entries)
	if err != nil {
		return nil, nil, errorUtil.Wrapf(err, "failed to reconcile aws credential request %s", name)
	}
	err = wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
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

	codec, err := v1.NewCodec()
	if err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to create credentials codec")
	}
	awsProvStatus := &v1.AWSProviderStatus{}
	if err = codec.DecodeProviderSpec(cr.Status.ProviderStatus, awsProvStatus); err != nil {
		return nil, nil, errorUtil.Wrapf(err, "failed to decode credentials request %s", cr.Name)
	}
	accessKeyID, secAccessKey, err := m.reconcileAWSCredentials(ctx, cr)
	if err != nil {
		return nil, nil, errorUtil.Wrapf(err, "failed to reconcile aws credentials from credential request %s", cr.Name)
	}
	return cr, &Credentials{
		Username:        awsProvStatus.User,
		PolicyName:      awsProvStatus.Policy,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secAccessKey,
	}, nil
}

func (m *CredentialMinterCredentialManager) reconcileCredentialRequest(ctx context.Context, name string, ns string, entries []v1.StatementEntry) (*v1.CredentialsRequest, error) {
	codec, err := v1.NewCodec()
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to create provider codec")
	}
	providerSpec, err := codec.EncodeProviderSpec(&v1.AWSProviderSpec{
		TypeMeta: controllerruntime.TypeMeta{
			Kind: "AWSProviderSpec",
		},
		StatementEntries: entries,
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

func (m *CredentialMinterCredentialManager) reconcileAWSCredentials(ctx context.Context, cr *v1.CredentialsRequest) (string, string, error) {
	sec := &v12.Secret{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: cr.Spec.SecretRef.Name, Namespace: cr.Spec.SecretRef.Namespace}, sec)
	if err != nil {
		return "", "", errorUtil.Wrapf(err, "failed to get aws credentials secret %s", cr.Spec.SecretRef.Name)
	}
	awsAccessKeyID := string(sec.Data[defaultCredentialsKeyIDName])
	awsSecretAccessKey := string(sec.Data[defaultCredentialsSecretKeyName])
	if awsAccessKeyID == "" {
		return "", "", errorUtil.New(fmt.Sprintf("aws access key id is undefined in secret %s", sec.Name))
	}
	if awsSecretAccessKey == "" {
		return "", "", errorUtil.New(fmt.Sprintf("aws secret access key is undefined in secret %s", sec.Name))
	}
	return awsAccessKeyID, awsSecretAccessKey, nil
}
