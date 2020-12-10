package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"time"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func upgradeApproval(ctx context.Context, preUpgradeBackupExecutor backup.BackupExecutor, client k8sclient.Client, ip *v1alpha1.InstallPlan, log l.Logger) error {
	if ip.Spec.Approved == false && len(ip.Spec.ClusterServiceVersionNames) > 0 {
		log.Infof("Approving", l.Fields{"installPlan": ip.Name, "csv's": ip.Spec.ClusterServiceVersionNames[0]})
		ip.Spec.Approved = true

		// Perform a backup of the product before updating the InstalPlan. We
		// must check that the product is already installed, as this function
		// is also called when the product is first installed
		if ip.Generation > 1 {
			backupTimeout := time.Minute * 20
			log.Infof("Triggering pre-upgrade backups", l.Fields{"backupTimeout": backupTimeout})
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
