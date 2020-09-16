package installation

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"

	"github.com/sirupsen/logrus"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewBootstrapReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	return &Reconciler{
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  installation,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling bootstrap stage")

	phase, err := r.reconcileOauthSecrets(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile oauth secrets", err)
		return phase, err
	}

	phase, err = r.reconcilerGithubOauthSecret(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile github oauth secrets", err)
		return phase, err
	}

	phase, err = r.reconcilerRHMIConfigCR(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile customer config", err)
		return phase, err
	}

	phase, err = r.reconcileRHMIConfigPermissions(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile customer config dedicated admin permissions", err)
		return phase, err
	}

	phase, err = r.retrieveConsoleURLAndSubdomain(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to retrieve console url and subdomain", err)
		return phase, err
	}

	phase, err = r.checkCloudResourcesConfig(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to check cloud resources config settings", err)
		return phase, err
	}
	events.HandleStageComplete(r.recorder, installation, integreatlyv1alpha1.BootstrapStage)

	metrics.SetRHMIInfo(installation)
	logrus.Info("Metric rhmi_info exposed")

	logrus.Info("Bootstrap stage reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) checkCloudResourcesConfig(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cloudConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:       DefaultCloudResourceConfigName,
			Namespace:  r.installation.Namespace,
			Finalizers: []string{deletionFinalizer},
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, cloudConfig, func() error {
		if cloudConfig.Data == nil {
			cloudConfig.Data = map[string]string{}
		}
		if strings.ToLower(r.installation.Spec.UseClusterStorage) == "true" {
			cloudConfig.Data["managed"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["workshop"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["self-managed"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["managed-api"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
		} else {
			cloudConfig.Data["managed"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
			cloudConfig.Data["workshop"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["self-managed"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
			cloudConfig.Data["managed-api"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilerRHMIConfigCR(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	rhmiConfig := &integreatlyv1alpha1.RHMIConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-config",
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, rhmiConfig, func() error {
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reconciling the Customer Config CR: %v", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileOauthSecrets(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// List of products that require secret for OAuthClient
	productsList := []integreatlyv1alpha1.ProductName{
		integreatlyv1alpha1.ProductRHSSO,
		integreatlyv1alpha1.ProductRHSSOUser,
		integreatlyv1alpha1.Product3Scale,
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: oauthClientSecrets.Namespace}, oauthClientSecrets)
	if !k8serr.IsNotFound(err) && err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
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
			err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name}, oauthClient)
			if !k8serr.IsNotFound(err) && err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling OAuth clients secrets: %w", err)
	}
	logrus.Info("Bootstrap OAuth client secrets successfully reconciled")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) retrieveConsoleURLAndSubdomain(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	consoleRouteCR, err := getConsoleRouteCR(ctx, serverClient)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not find CR route: %w", err)
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve CR route: %w", err)
	}

	r.installation.Spec.MasterURL = consoleRouteCR.Status.Ingress[0].Host
	r.installation.Spec.RoutingSubdomain = consoleRouteCR.Status.Ingress[0].RouterCanonicalHostname

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func getConsoleRouteCR(ctx context.Context, serverClient k8sclient.Client) (*routev1.Route, error) {
	// discover and set master url and routing subdomain
	consoleRouteCR := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "console",
			Namespace: "openshift-console",
		},
	}
	key := k8sclient.ObjectKey{
		Name:      consoleRouteCR.GetName(),
		Namespace: consoleRouteCR.GetNamespace(),
	}

	err := serverClient.Get(ctx, key, consoleRouteCR)
	if err != nil {
		return nil, err
	}
	return consoleRouteCR, nil
}

func (r *Reconciler) reconcilerGithubOauthSecret(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {

	githubOauthSecretCR := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetGHOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, githubOauthSecretCR, func() error {
		ownerutil.EnsureOwner(githubOauthSecretCR, installation)

		if len(githubOauthSecretCR.Data) == 0 {
			githubOauthSecretCR.Data = map[string][]byte{
				"clientId": []byte("dummy"),
				"secret":   []byte("dummy"),
			}
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling Github OAuth secrets: %w", err)
	}

	logrus.Info("Bootstrap Github OAuth secrets successfully reconciled")

	secretRoleCR := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetGHOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, secretRoleCR, func() error {
		secretRoleCR.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				Verbs:         []string{"update", "get"},
				ResourceNames: []string{r.ConfigManager.GetGHOauthClientsSecretName()},
			},
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error creating Github OAuth secrets role: %w", err)
	}

	secretRoleBindingCR := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetGHOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, secretRoleBindingCR, func() error {
		secretRoleBindingCR.RoleRef = rbacv1.RoleRef{
			Name: secretRoleCR.GetName(),
			Kind: "Role",
		}
		secretRoleBindingCR.Subjects = []rbacv1.Subject{
			{
				Name: "dedicated-admins",
				Kind: "Group",
			},
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error creating Github OAuth secrets role binding: %w", err)
	}
	logrus.Info("Bootstrap Github OAuth secrets Role and Role Binding successfully reconciled")

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) reconcileRHMIConfigPermissions(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	configRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmiconfig-dedicated-admins-role-binding",
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	// Get the dedicated admins role binding. If it's not found, return
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      "rhmiconfig-dedicated-admins-role-binding",
		Namespace: r.ConfigManager.GetOperatorNamespace(),
	}, configRoleBinding); err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}

		return integreatlyv1alpha1.PhaseFailed, err
	}

	logrus.Info("Found and deleted rhmiconfig-dedicated-admins-role-binding")

	// Delete the role binding
	if err := serverClient.Delete(ctx, configRoleBinding); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
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
