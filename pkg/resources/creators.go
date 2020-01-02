package resources

import (
	"context"
	"fmt"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdate(ctx context.Context, serverClient k8sclient.Client, obj runtime.Object) error {
	err := serverClient.Create(ctx, obj)
	if err != nil && k8serr.IsAlreadyExists(err) {
		err = serverClient.Update(ctx, obj)
		if err != nil {
			return fmt.Errorf("error updating object: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error creating object: %w", err)
	}

	return nil
}
