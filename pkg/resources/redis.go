package resources

// Temporary helpers for the Redis to Valkey migration (MGDAPI-6598).
// After migration is complete, engine and version are no longer needed on the CR
// and ReconcileRedis can be as before — remove this file and redis_test.go.
// Corresponding cleanup in CRO is also required (e.g. default to Valkey, drop Redis engine support).

import (
	"context"

	crov1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RedisEngineForReconcile returns engine and version for a Redis CR during migration.
// New installations use Valkey. The RHOAM operator does not migrate existing Redis to
// Valkey; existing CRs keep their current spec (including an unset engine, which CRO
// treats as redis). Migrating existing deployments is a manual process handled outside
// the operator (e.g. by SRE/tech support).
func RedisEngineForReconcile(ctx context.Context, client k8sclient.Client, name, namespace string) (engine, engineVersion string, err error) {
	existing := &crov1.Redis{}
	err = client.Get(ctx, k8sclient.ObjectKey{Name: name, Namespace: namespace}, existing)
	if k8serr.IsNotFound(err) {
		return croTypes.EngineValkey, "", nil
	}
	if err != nil {
		return "", "", err
	}
	return existing.Spec.Engine, existing.Spec.EngineVersion, nil
}
