package resources

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func upgradeApproval(ctx context.Context, client pkgclient.Client, ip *v1alpha1.InstallPlan) error {
	if ip.Spec.Approved == false && len(ip.Spec.ClusterServiceVersionNames) > 0 {
		logrus.Infof("Approving %s resource version: %s", ip.Name, ip.Spec.ClusterServiceVersionNames[0])
		ip.Spec.Approved = true
		err := client.Update(ctx, ip)
		if err != nil {
			return errors.Wrap(err, "error approving installplan")
		}

	}
	return nil
}
