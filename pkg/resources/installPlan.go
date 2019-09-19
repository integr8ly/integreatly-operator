package resources

import (
	"context"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func upgradeApproval(ctx context.Context, client pkgclient.Client, ip *v1alpha1.InstallPlan, maxVersion *Version) error {
	if ip.Spec.Approved == false && len(ip.Spec.ClusterServiceVersionNames) > 0 {
		logrus.Infof("getting version for %s", ip.Spec.ClusterServiceVersionNames[0])
		// convert "keycloak.1.8.2" into "1.8.2"
		verStr := strings.SplitN(ip.Spec.ClusterServiceVersionNames[0], ".", 2)[1]
		resourceVersion, err := NewVersion(verStr)
		if err != nil {
			//bad version string, skip it
			return errors.Wrap(err, "error determining installplan version")
		}
		logrus.Infof("checking %s resource version: %s, against max version %s", ip.Name, resourceVersion.AsString(), maxVersion.AsString())
		if !resourceVersion.IsNewerThan(maxVersion) {
			ip.Spec.Approved = true
		}
		if ip.Spec.Approved {
			err = client.Update(ctx, ip)
			if err != nil {
				return errors.Wrap(err, "error approving installplan")
			}
		}
	}
	return nil
}
