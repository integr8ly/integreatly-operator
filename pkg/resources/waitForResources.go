package resources

import (
	"context"
	"fmt"
	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	crotypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForRHSSOPostgresToBeComplete(serverClient k8sclient.Client, installName, installNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	postgres := &crov1alpha1.Postgres{}
	if err := serverClient.Get(context.TODO(), k8sclient.ObjectKey{Name: fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installName), Namespace: installNamespace}, postgres); err != nil {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	if postgres.Status.Phase == crotypes.PhaseComplete {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	return integreatlyv1alpha1.PhaseAwaitingComponents, nil
}
