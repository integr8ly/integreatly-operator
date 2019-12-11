package resources

import (
	pkgerr "github.com/pkg/errors"

	"golang.org/x/net/context"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdate(ctx context.Context, serverClient pkgclient.Client, obj runtime.Object) error {
	err := serverClient.Create(ctx, obj)
	if err != nil && k8serr.IsAlreadyExists(err) {
		err = serverClient.Update(ctx, obj)
		if err != nil {
			return pkgerr.Wrapf(err, "error updating object")
		}
	} else if err != nil {
		return pkgerr.Wrapf(err, "error creating object")
	}

	return nil
}
