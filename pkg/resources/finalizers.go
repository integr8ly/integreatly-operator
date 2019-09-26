package resources

import (
	"context"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddFinalizer adds a finalizer to the custom resource. This allows us to clean up oauth clients
// and other cluster level objects owned by the installation before the cr is deleted
func AddFinalizer(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client, product *v1alpha1.InstallationProductStatus, finalizer string) error {
	if !contains(inst.GetFinalizers(), finalizer) && inst.GetDeletionTimestamp() == nil {
		inst.SetFinalizers(append(inst.GetFinalizers(), finalizer))
		err := client.Update(ctx, inst)
		if err != nil {
			logrus.Error("Error adding finalizer to custom resource", err)
			return err
		}
	}
	return nil
}

// RemoveOauthClient deletes an oauth client owned by a product and removes its finalizer from
// the installation custom resource
func RemoveOauthClient(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client, oauthClient oauthClient.OauthV1Interface, oauthId string) error {
	err := oauthClient.OAuthClients().Delete(oauthId, &metav1.DeleteOptions{})
	if err != nil && !k8serr.IsNotFound(err) {
		logrus.Error("Error cleaning up oauth client", err)
		return err
	}
	return nil
}

// RemoveProductFinalizer removes a given finalizer from the installation custom resource
func RemoveProductFinalizer(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client, product string) error {
	finalizer := "finalizer." + product + ".integreatly.org"
	logrus.Infof("removing finalizer %s", finalizer)
	inst.SetFinalizers(remove(inst.GetFinalizers(), finalizer))
	err := client.Update(ctx, inst)
	if err != nil {
		logrus.Info("Error removing finalizer from custom resource", err)
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
