package client

import (
	"context"

	"github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// define a function that allows us to perform modification logic on the
// custom resource (e.g. setting owner refs) before creating it
type modifyResourceFunc func(cr metav1.Object) error

// ReconcileBlobStorage creates or updates a blob storage custom resource
func ReconcileBlobStorage(ctx context.Context, client client.Client, productName, deploymentType, tier, name, ns, secretName, secretNs string, modifyFunc modifyResourceFunc) (*v1alpha1.BlobStorage, error) {
	bs := &v1alpha1.BlobStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"productName": productName,
			},
		},
	}

	// execute logic to modify the resource before creation
	// e.g. add owner refs
	if modifyFunc != nil {
		err := modifyFunc(bs)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to execute modification function on resource %s", name)
		}
	}

	// Create or update the resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, bs, func() error {
		bs.Spec.Type = deploymentType
		bs.Spec.Tier = tier
		bs.Spec.SecretRef = &croType.SecretRef{
			Name:      secretName,
			Namespace: secretNs,
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile blob storage request for %s", name)
	}

	return bs, nil
}

// ReconcilePostgres creates or updates a postgres custom resource
func ReconcilePostgres(ctx context.Context, client client.Client, productName, deploymentType, tier, name, ns, secretName, secretNs string, applyImmediately bool, snapshotFrequency, snapshotRetention croType.Duration, modifyFunc modifyResourceFunc) (*v1alpha1.Postgres, error) {
	pg := &v1alpha1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}

	// execute logic to modify the resource before creation
	// e.g. add owner refs
	if modifyFunc != nil {
		err := modifyFunc(pg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to execute modification function on resource %s", name)
		}
	}

	// Create or update the resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, pg, func() error {
		pg.Labels = map[string]string{"productName": productName}
		pg.Spec.Type = deploymentType
		pg.Spec.Tier = tier
		pg.Spec.SecretRef = &croType.SecretRef{
			Name:      secretName,
			Namespace: secretNs,
		}
		pg.Spec.ApplyImmediately = applyImmediately
		pg.Spec.SnapshotFrequency = snapshotFrequency
		pg.Spec.SnapshotRetention = snapshotRetention

		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile postgres request for %s", name)
	}

	return pg, nil
}

// ReconcileRedis creates or updates a redis custom resource
func ReconcileRedis(ctx context.Context, client client.Client, productName, deploymentType, tier, name, ns, secretName, secretNs, size string, applyImmediately, maintenanceWindow bool, modifyFunc modifyResourceFunc) (*v1alpha1.Redis, error) {
	r := &v1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"productName": productName,
			},
		},
	}

	// execute logic to modify the resource before creation
	// e.g. add owner refs
	if modifyFunc != nil {
		err := modifyFunc(r)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to execute modification function on resource %s", name)
		}
	}

	// Create or update the resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, r, func() error {
		r.Spec.Type = deploymentType
		r.Spec.Tier = tier
		r.Spec.SecretRef = &croType.SecretRef{
			Name:      secretName,
			Namespace: secretNs,
		}
		r.Spec.Size = size
		r.Spec.ApplyImmediately = applyImmediately
		r.Spec.MaintenanceWindow = maintenanceWindow

		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile redis request for %s", name)
	}

	return r, nil
}
