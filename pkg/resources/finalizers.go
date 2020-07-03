package resources

import (
	"context"

	"github.com/sirupsen/logrus"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	projectv1 "github.com/openshift/api/project/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddFinalizer adds a finalizer to the custom resource. This allows us to clean up oauth clients
// and other cluster level objects owned by the installation before the cr is deleted
func AddFinalizer(ctx context.Context, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client, finalizer string) error {
	if !Contains(inst.GetFinalizers(), finalizer) && inst.GetDeletionTimestamp() == nil {
		inst.SetFinalizers(append(inst.GetFinalizers(), finalizer))
		err := client.Update(ctx, inst)
		if err != nil {
			logrus.Error("Error adding finalizer to custom resource", err)
			return err
		}
	}
	return nil
}

// RemoveOauthClient deletes an oauth client by name
func RemoveOauthClient(oauthClient oauthClient.OauthV1Interface, oauthClientName string) error {
	err := oauthClient.OAuthClients().Delete(oauthClientName, &metav1.DeleteOptions{})
	if err != nil && !k8serr.IsNotFound(err) {
		logrus.Error("Error cleaning up oauth client", err)
		return err
	}
	return nil
}

// RemoveNamespace deletes a namespace of a product
func RemoveNamespace(ctx context.Context, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	ns, err := GetNS(ctx, namespace, client)
	if err != nil {
		// Since we are using ProjectRequests and limited permissions,
		// request can return "forbidden" error even when Namespace simply doesn't exist
		if k8serr.IsNotFound(err) || k8serr.IsForbidden(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		logrus.Error("Error getting a namespace", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if ns.Status.Phase == corev1.NamespaceTerminating {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	nsProject := &projectv1.Project{
		ObjectMeta: ns.ObjectMeta,
	}

	err = client.Delete(ctx, nsProject)
	logrus.Infof("namespace %s removal triggered, status will be checked on next reconcile", namespace)
	if err != nil && !k8serr.IsNotFound(err) {
		logrus.Error("Error deleting a namespace", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	return integreatlyv1alpha1.PhaseInProgress, nil
}

// RemoveProductFinalizer removes a given finalizer from the installation custom resource
func RemoveProductFinalizer(ctx context.Context, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client, product string) error {
	finalizer := "finalizer." + product + ".integreatly.org"
	inst.SetFinalizers(Remove(inst.GetFinalizers(), finalizer))
	err := client.Update(ctx, inst)
	if err != nil {
		logrus.Info("Error removing finalizer from custom resource", err)
		return err
	}
	return nil
}

// RemoveFinalizerAndUpdate removes a given finalizer from the installation custom resource
func RemoveFinalizerAndUpdate(ctx context.Context, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client, finalizer string) error {
	inst.SetFinalizers(Remove(inst.GetFinalizers(), finalizer))
	err := client.Update(ctx, inst)
	if err != nil {
		logrus.Info("Error removing finalizer from custom resource", err)
		return err
	}
	return nil
}

// Contains checks an array of strings for a specific string
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Remove removes a string from an array of strings
func Remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
