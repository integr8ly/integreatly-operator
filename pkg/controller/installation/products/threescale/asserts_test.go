package threescale

import (
	"context"
	"errors"
	"fmt"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AssertFunc func(installation *v1alpha1.Installation, configManager *config.Manager, fakeSigsClient pkgclient.Client, fakeThreeScaleClient *ThreeScaleInterfaceMock, fakeAppsV1Client appsv1Client.AppsV1Interface, fakeOauthClient oauthClient.OauthV1Interface, fakeMPM *marketplace.MarketplaceInterfaceMock) error

func assertNoop(*v1alpha1.Installation, *config.Manager, pkgclient.Client, *ThreeScaleInterfaceMock, appsv1Client.AppsV1Interface, oauthClient.OauthV1Interface) error {
	return nil
}

func assertInstallationSuccessfull(installation *v1alpha1.Installation, configManager *config.Manager, fakeSigsClient pkgclient.Client, fakeThreeScaleClient *ThreeScaleInterfaceMock, fakeAppsV1Client appsv1Client.AppsV1Interface, fakeOauthClient oauthClient.OauthV1Interface, fakeMPM *marketplace.MarketplaceInterfaceMock) error {
	ctx := context.TODO()

	// A namespace should have been created..
	ns := &corev1.Namespace{}
	err := fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: defaultInstallationNamespace}, ns)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("%s namespace should have been created", defaultInstallationNamespace))
	}

	// A subscription to the product operator should have been created.
	if len(fakeMPM.CreateSubscriptionCalls()) != 1 {
		return errors.New(fmt.Sprintf("%s operator subscription was not created", packageName))
	}

	// The main s3credentials should have been copied into the 3scale namespace.
	s3Credentials := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: s3CredentialsSecretName, Namespace: defaultInstallationNamespace}, s3Credentials)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("s3Credentials were not copied into %s namespace", defaultInstallationNamespace))
	}

	// The product custom resource should have been created.
	apim := &threescalev1.APIManager{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: apiManagerName, Namespace: defaultInstallationNamespace}, apim)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("APIManager '%s' was not created", apiManagerName))
	}
	if apim.Spec.WildcardDomain != installation.Spec.RoutingSubdomain {
		return errors.New(fmt.Sprintf("APIManager wildCardDomain is misconfigured. '%s' should be '%s'", apim.Spec.WildcardDomain, installation.Spec.RoutingSubdomain))
	}

	// RHSSO integration should be configured.
	rhssoConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return errors.New("Error getting RHSSO config")
	}
	kcr := &aerogearv1.KeycloakRealm{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: rhssoConfig.GetRealm(), Namespace: rhssoConfig.GetNamespace()}, kcr)
	if !containsClient(kcr.Spec.Clients, clientId) {
		return errors.New(fmt.Sprintf("Keycloak client '%s' was not created", clientId))
	}
	integrationCall := fakeThreeScaleClient.AddSSOIntegrationCalls()[0]
	if integrationCall.Data["client_id"] != clientId || integrationCall.Data["site"] != rhssoConfig.GetHost()+"/auth/realms/"+rhssoConfig.GetRealm() {
		return errors.New(fmt.Sprintf("SSO integration request to 3scale API was incorrect"))
	}

	// RHSSO admin user should be set as 3scale admin
	updateAdminCall := fakeThreeScaleClient.UpdateAdminPortalUserDetailsCalls()[0]
	if updateAdminCall.Username != rhsso.CustomerAdminUser.UserName || updateAdminCall.Email != rhsso.CustomerAdminUser.Email {
		return errors.New(fmt.Sprintf("Request to 3scale API to update admin details was incorrect"))
	}
	adminSecret := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: threeScaleAdminDetailsSecret.Name, Namespace: defaultInstallationNamespace}, adminSecret)
	if string(adminSecret.Data["ADMIN_USER"]) != rhsso.CustomerAdminUser.UserName || string(adminSecret.Data["ADMIN_EMAIL"]) != rhsso.CustomerAdminUser.Email {
		return errors.New(fmt.Sprintf("3scale admin secret details were not updated"))
	}

	// Service discovery should be configured
	threeScaleOauth, err := fakeOauthClient.OAuthClients().Get(oauthId, metav1.GetOptions{})
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("3scale should have an Ouath Client '%s' created", oauthId))
	}
	if threeScaleOauth.RedirectURIs[0] != installation.Spec.MasterURL {
		return errors.New(fmt.Sprintf("3scale Ouath Client redirect uri should be %s and is %s", installation.Spec.MasterURL, threeScaleOauth.RedirectURIs[0]))
	}
	serviceDiscoveryConfigMap := &corev1.ConfigMap{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: threeScaleServiceDiscoveryConfigMap.Name, Namespace: defaultInstallationNamespace}, serviceDiscoveryConfigMap)
	if string(adminSecret.Data["ADMIN_USER"]) != rhsso.CustomerAdminUser.UserName || string(adminSecret.Data["ADMIN_EMAIL"]) != rhsso.CustomerAdminUser.Email {
		return errors.New(fmt.Sprintf("3scale admin secret details were not updated"))
	}
	if string(serviceDiscoveryConfigMap.Data["service_discovery.yml"]) != sdConfig {
		return errors.New(fmt.Sprintf("Service discovery config is misconfigured"))
	}

	// system-app and system-sidekiq deploymentconfigs should have been rolled out on first reconcile.
	sa, err := fakeAppsV1Client.DeploymentConfigs(defaultInstallationNamespace).Get("system-app", metav1.GetOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting deplymentconfig: %v", err))
	}
	if sa.Status.LatestVersion == 1 {
		return errors.New(fmt.Sprintf("system-app was not rolled out"))
	}
	ssk, err := fakeAppsV1Client.DeploymentConfigs(defaultInstallationNamespace).Get("system-sidekiq", metav1.GetOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting deplymentconfig: %v", err))
	}
	if ssk.Status.LatestVersion == 1 {
		return errors.New(fmt.Sprintf("system-sidekiq was not rolled out"))
	}

	return nil
}
