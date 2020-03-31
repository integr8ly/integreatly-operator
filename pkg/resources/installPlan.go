package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func upgradeApproval(ctx context.Context, preUpgradeBackupExecutor backup.BackupExecutor, client k8sclient.Client, ip *v1alpha1.InstallPlan) error {
	if ip.Spec.Approved == false && len(ip.Spec.ClusterServiceVersionNames) > 0 {
		logrus.Infof("Approving %s resource version: %s", ip.Name, ip.Spec.ClusterServiceVersionNames[0])
		ip.Spec.Approved = true

		// Perform a backup of the product before updating the InstalPlan. We
		// must check that the product is already installed, as this function
		// is also called when the product is first installed
		if ip.Generation > 1 {
			backupTimeout := time.Minute * 20
			logrus.Infof("Triggering pre-upgrade backups with timeout of %v", backupTimeout)
			if err := preUpgradeBackupExecutor.PerformBackup(client, backupTimeout); err != nil {
				return fmt.Errorf("error performing pre-upgrade backup: %w", err)
			}
		}

		err := client.Update(ctx, ip)
		if err != nil {
			return fmt.Errorf("error approving installplan: %w", err)
		}

	}
	return nil
}
