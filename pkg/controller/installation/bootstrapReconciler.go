package installation

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	oauthv1 "github.com/openshift/api/oauth/v1"
	usersv1 "github.com/openshift/api/user/v1"
	"github.com/pkg/errors"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	dedicatedAdminsGroupName = "dedicated-admins"
	rhmiAdminsGroupName      = "rhmi-admins"
)

func NewBootstrapReconciler(configManager config.ConfigReadWriter, i *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	return &Reconciler{
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  i,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *v1alpha1.Installation
	*resources.Reconciler
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, in *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling bootstrap stage")

	phase, err := r.reconcileOauthSecrets(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.retrieveConsoleUrlAndSubdomain(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileRHMIAdminsGroup(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	logrus.Infof("Bootstrap stage reconciled successfully")
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileOauthSecrets(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// List of products that require secret for OAuthClient
	productsList := []v1alpha1.ProductName{
		v1alpha1.ProductRHSSO,
		v1alpha1.ProductRHSSOUser,
		v1alpha1.Product3Scale,
		v1alpha1.ProductMobileDeveloperConsole,
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: oauthClientSecrets.Namespace}, oauthClientSecrets)
	if !k8serr.IsNotFound(err) && err != nil {
		return v1alpha1.PhaseFailed, err
	} else if k8serr.IsNotFound(err) {
		oauthClientSecrets.Data = map[string][]byte{}
	}

	for _, product := range productsList {
		if _, ok := oauthClientSecrets.Data[string(product)]; !ok {
			oauthClient := &oauthv1.OAuthClient{
				ObjectMeta: metav1.ObjectMeta{
					Name: r.installation.Spec.NamespacePrefix + string(product),
				},
			}
			err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: oauthClientSecrets.Name}, oauthClient)
			if !k8serr.IsNotFound(err) && err != nil {
				return v1alpha1.PhaseFailed, err
			} else if k8serr.IsNotFound(err) {
				oauthClientSecrets.Data[string(product)] = []byte(generateSecret(32))
			} else {
				// recover secret from existing OAuthClient object in case Secret object was deleted
				oauthClientSecrets.Data[string(product)] = []byte(oauthClient.Secret)
				logrus.Warningf("OAuth client secret for %s recovered from OAutchClient object", string(product))
			}
		}
	}

	oauthClientSecrets.ObjectMeta.ResourceVersion = ""
	err = resources.CreateOrUpdate(ctx, serverClient, oauthClientSecrets)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "Error reconciling OAuth clients secrets")
	}
	logrus.Info("Bootstrap OAuth client secrets successfully reconciled")

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) retrieveConsoleUrlAndSubdomain(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {

	consoleRouteCR, err := getConsoleRouteCR(ctx, serverClient)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not find CR route"))
		}
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, fmt.Sprintf("could not retrieve CR route"))
	}

	r.installation.Spec.MasterURL = consoleRouteCR.Status.Ingress[0].Host
	r.installation.Spec.RoutingSubdomain = consoleRouteCR.Status.Ingress[0].RouterCanonicalHostname

	return v1alpha1.PhaseCompleted, nil

}

// Reconciles RHMI user group which contains users that are expected to be admins in the integreatly product suite.
// This group should include all users from the dedicated-admins group
func (r *Reconciler) reconcileRHMIAdminsGroup(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infoln("reconciling RHMI user group")

	rhmiAdminsGroup := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: rhmiAdminsGroupName,
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: rhmiAdminsGroup.Name}, rhmiAdminsGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to get %s group", rhmiAdminsGroup.Name)
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, rhmiAdminsGroup, func(existing runtime.Object) error {
		// Get users from the dedicated-admins group
		dedicatedAdminGroup := &usersv1.Group{}
		if err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: dedicatedAdminsGroupName}, dedicatedAdminGroup); err != nil {
			return err
		}

		rhmiAdminsGroup := existing.(*usersv1.Group)
		rhmiAdminUsers := []string{}
		rhmiAdminUsers = append(rhmiAdminUsers, rhmiAdminsGroup.Users...)

		// Ensure all users from dedicated-admins group are added to the rhmi-admins group
		for _, user := range dedicatedAdminGroup.Users {
			if !resources.Contains(rhmiAdminUsers, user) {
				rhmiAdminUsers = append(rhmiAdminUsers, user)
			}
		}

		rhmiAdminsGroup.Users = rhmiAdminUsers

		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create or update %s group", rhmiAdminsGroup.Name)
	}

	logrus.Infoln("The operation result for group " + rhmiAdminsGroup.Name + " was " + string(or))

	return v1alpha1.PhaseCompleted, nil
}

func getConsoleRouteCR(ctx context.Context, serverClient pkgclient.Client) (*routev1.Route, error) {
	// discover and set master url and routing subdomain
	consoleRouteCR := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "console",
			Namespace: "openshift-console",
		},
	}
	key := client.ObjectKey{
		Name:      consoleRouteCR.GetName(),
		Namespace: consoleRouteCR.GetNamespace(),
	}

	err := serverClient.Get(ctx, key, consoleRouteCR)
	if err != nil {
		return nil, err
	}
	return consoleRouteCR, nil
}

func generateSecret(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	buf := make([]rune, length)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	return string(buf)
}
