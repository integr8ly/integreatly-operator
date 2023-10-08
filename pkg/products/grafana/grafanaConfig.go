package grafana

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "customer-monitoring"
	defaultRoutename             = "grafana-route"
	gfSecurityAdminUser          = "admin"
)

func ReconcileGrafanaSecrets(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()
	log.Info("reconciling Grafana configuration secrets")
	nsPrefix := installation.Spec.NamespacePrefix

	grafanaProxySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-k8s-proxy",
			Namespace: nsPrefix + "customer-monitoring",
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, grafanaProxySecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(grafanaProxySecret, installation)
		if grafanaProxySecret.Data == nil {
			grafanaProxySecret.Data = map[string][]byte{}
		}
		grafanaProxySecret.Data["session_secret"] = []byte(resources.GenerateRandomPassword(20, 2, 2, 2))
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	grafanaAdminCredsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-admin-credentials",
			Namespace: nsPrefix + "customer-monitoring",
		},
		Data: getAdminCredsSecretData(),
		Type: corev1.SecretTypeOpaque,
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, grafanaAdminCredsSecret, func() error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getAdminCredsSecretData() map[string][]byte {
	password := []byte(RandStringRunes(10))
	credentials := map[string][]byte{
		"GF_SECURITY_ADMIN_USER":     []byte(gfSecurityAdminUser),
		"GF_SECURITY_ADMIN_PASSWORD": password,
	}

	// Make the credentials available to the environment, similar is it was done in Grafana operator (resolve admin login issue?)
	err := os.Setenv("GF_SECURITY_ADMIN_USER", string(credentials["GF_SECURITY_ADMIN_USER"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars (optional)")
		return credentials
	}
	err = os.Setenv("GF_SECURITY_ADMIN_PASSWORD", string(credentials["GF_SECURITY_ADMIN_PASSWORD"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars (optional)")
		return credentials
	}

	return credentials
}

func GetGrafanaConsoleURL(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (string, error) {

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	grafanaRoute := &routev1.Route{}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultRoutename, Namespace: ns}, grafanaRoute)
	if err != nil {
		return "", err
	}

	return "https://" + grafanaRoute.Spec.Host, nil
}

func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func RandStringRunes(s int) string {
	b := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}
