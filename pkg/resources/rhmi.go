package resources

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRhmiCr(ctx context.Context, client k8sclient.Client, namespace string) (*integreatlyv1alpha1.RHMI, error) {
	logrus.Infof("Looking for RHMI CR in %s namespace", namespace)

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
