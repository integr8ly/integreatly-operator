package resources

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetRhmiCr retrieves the RHMI CR for the current installation
//
// Deprecated: Use pkg/resources/rhmi.GetRhmiCr instead
func GetRhmiCr(client k8sclient.Client, ctx context.Context, namespace string, log l.Logger) (*integreatlyv1alpha1.RHMI, error) {
	return rhmi.GetRhmiCr(client, ctx, namespace, log)
}
