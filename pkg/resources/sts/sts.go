package sts

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterCloudCredentialName = "cluster"
	AddonStsArnParameterName   = "sts-role-arn"
)

func IsClusterSTS(ctx context.Context, client k8sclient.Client, log logger.Logger) (bool, error) {
	cloudCredential := &cloudcredentialv1.CloudCredential{}
	if err := client.Get(ctx, k8sclient.ObjectKey{Name: clusterCloudCredentialName}, cloudCredential); err != nil {
		log.Error("failed to get cloudCredential whle checking if STS mode", err)
		return false, err
	}

	if cloudCredential.Spec.CredentialsMode == cloudcredentialv1.CloudCredentialsModeManual {
		log.Info("STS mode")
		return true, nil
	}
	log.Info("non STS mode")
	return false, nil
}

func GetStsRoleArn(ctx context.Context, client k8sclient.Client, namespace string) (string, error) {
	stsRoleArn, stsFound, err := addon.GetStringParameterByInstallType(
		ctx,
		client,
		integreatlyv1alpha1.InstallationTypeManagedApi,
		namespace,
		AddonStsArnParameterName,
	)
	if err != nil {
		return "", fmt.Errorf("failed while retrieving addon parameter %w", err)
	}
	if !stsFound || stsRoleArn == "" {
		return "", fmt.Errorf("no STS configuration found")
	}

	return stsRoleArn, nil
}
