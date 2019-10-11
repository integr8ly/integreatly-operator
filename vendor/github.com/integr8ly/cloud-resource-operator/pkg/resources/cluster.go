package resources

import (
	"context"

	v1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterId(ctx context.Context, c client.Client) (string, error) {
	infra := &v1.Infrastructure{}
	if err := c.Get(ctx, types.NamespacedName{Name: "cluster"}, infra); err != nil {
		return "", errors.Wrap(err, "failed to retrieve cluster infrastructure")
	}
	return infra.Status.InfrastructureName, nil
}
