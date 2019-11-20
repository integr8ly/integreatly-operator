package resources

import (
	"context"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// define a function that allows us to perform modification logic on the
// custom resource (e.g. setting owner refs) before creating or updating it
type modifyResourceFunc func(cr metav1.Object) error

// ReconcileBlobStorage creates or updates a blob storage custom resource
func ReconcileBlobStorage(ctx context.Context, client client.Client, deploymentType, tier, name, ns, secretName, secretNs string, modifyFunc modifyResourceFunc) (*v1alpha1.BlobStorage, error) {
	bs := &v1alpha1.BlobStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
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
	_, err := controllerutil.CreateOrUpdate(ctx, client, bs, func(existing runtime.Object) error {
		c := existing.(*v1alpha1.BlobStorage)
		c.Spec.Type = deploymentType
		c.Spec.Tier = tier
		c.Spec.SecretRef = &types.SecretRef{
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

// ReconcileSMTPCredentialSet creates or updates an SMTP credential set
func ReconcileSMTPCredentialSet(ctx context.Context, client client.Client, deploymentType, tier, name, ns, secretName, secretNs string, modifyFunc modifyResourceFunc) (*v1alpha1.SMTPCredentialSet, error) {
	smtp := &v1alpha1.SMTPCredentialSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}

	// execute logic to modify the resource before creation
	// e.g. add owner refs
	if modifyFunc != nil {
		err := modifyFunc(smtp)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to execute modification function on resource %s", name)
		}
	}

	// Create or update the resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, smtp, func(existing runtime.Object) error {
		c := existing.(*v1alpha1.SMTPCredentialSet)
		c.Spec.Type = deploymentType
		c.Spec.Tier = tier
		c.Spec.SecretRef = &types.SecretRef{
			Name:      secretName,
			Namespace: secretNs,
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile smtp credential set request for %s", name)
	}

	return smtp, nil
}

// ReconcilePostgres creates or updates a postgres custom resource
func ReconcilePostgres(ctx context.Context, client client.Client, deploymentType, tier, name, ns, secretName, secretNs string, modifyFunc modifyResourceFunc) (*v1alpha1.Postgres, error) {
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
	_, err := controllerutil.CreateOrUpdate(ctx, client, pg, func(existing runtime.Object) error {
		c := existing.(*v1alpha1.Postgres)
		c.Spec.Type = deploymentType
		c.Spec.Tier = tier
		c.Spec.SecretRef = &types.SecretRef{
			Name:      secretName,
			Namespace: secretNs,
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile postgres request for %s", name)
	}

	return pg, nil
}

// ReconcileRedis creates or updates a redis custom resource
func ReconcileRedis(ctx context.Context, client client.Client, deploymentType, tier, name, ns, secretName, secretNs string, modifyFunc modifyResourceFunc) (*v1alpha1.Redis, error) {
	r := &v1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
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
	_, err := controllerutil.CreateOrUpdate(ctx, client, r, func(existing runtime.Object) error {
		c := existing.(*v1alpha1.Redis)
		c.Spec.Type = deploymentType
		c.Spec.Tier = tier
		c.Spec.SecretRef = &types.SecretRef{
			Name:      secretName,
			Namespace: secretNs,
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile redis request for %s", name)
	}

	return r, nil
}
