package resources

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRhmiCr(client k8sclient.Client, ctx context.Context, namespace string, log l.Logger) (*integreatlyv1alpha1.RHMI, error) {
	log.Infof("Looking for RHMI CR", l.Fields{"ns": namespace})

	installationList := &integreatlyv1alpha1.RHMIList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	err := client.List(ctx, installationList, listOpts...)
	if err != nil {
		return nil, err
	}
	if len(installationList.Items) == 0 {
		return nil, nil
	}
	if len(installationList.Items) != 1 {
		return nil, fmt.Errorf("Unexpected number of rhmi CRs: %w", err)
	}
	return &installationList.Items[0], nil
}
