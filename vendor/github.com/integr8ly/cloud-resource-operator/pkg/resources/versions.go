package resources

import (
	"context"

	"github.com/hashicorp/go-version"
	v1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	errorUtil "github.com/pkg/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func VerifyVersionUpgradeNeeded(currentVersion string, desiredVersion string) (bool, error) {
	current, err := version.NewVersion(currentVersion)

	if err != nil {
		return false, errorUtil.Wrap(err, "failed to parse current version")
	}
	desired, err := version.NewVersion(desiredVersion)

	if err != nil {
		return false, errorUtil.Wrap(err, "failed to parse desired version")
	}

	return current.LessThan(desired), nil
}

func VerifyPostgresMaintenanceWindow(ctx context.Context, client k8sclient.Client, namespace string, name string) (bool, error) {
	postgres := &v1.Postgres{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, postgres); err != nil {
		return false, err
	}

	if postgres.Spec.MaintenanceWindow {
		return true, nil
	}

	return false, nil
}

func VerifyRedisMaintenanceWindow(ctx context.Context, client k8sclient.Client, namespace string, name string) (bool, error) {
	redis := &v1.Redis{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, redis); err != nil {
		return false, err
	}

	if redis.Spec.MaintenanceWindow {
		return true, nil
	}

	return false, nil
}
