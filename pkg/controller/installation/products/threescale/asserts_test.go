package threescale

import (
	"context"
	"errors"
	"fmt"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	oauthv1 "github.com/openshift/api/oauth/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AssertFunc func(ThreeScaleTestScenario, *config.Manager) error

func assertNoop(ThreeScaleTestScenario, *config.Manager) error {
	return nil
}

func assertInstallationSuccessfull(scenario ThreeScaleTestScenario, configManager *config.Manager) error {
	ctx := context.TODO()
	accessToken := "test123"
	fakeSigsClient := scenario.FakeSigsClient
	installation := scenario.Installation
	fakeThreeScaleClient := scenario.FakeThreeScaleClient
	fakeAppsV1Client := scenario.FakeAppsV1Client

	tsConfig, err := configManager.ReadThreeScale()
	if err != nil {
		return err
	}

	oauthId := installation.Spec.NamespacePrefix + string(tsConfig.GetProductName())
	oauthClientSecrets := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: configManager.GetOauthClientsSecretName(), Namespace: configManager.GetOperatorNamespace()}, oauthClientSecrets)
	sdConfig := fmt.Sprintf("production:\n  enabled: true\n  authentication_method: oauth\n  oauth_server_type: builtin\n  client_id: '%s'\n  client_secret: '%s'\n", oauthId, oauthClientSecrets.Data[string(tsConfig.GetProductName())])

	// A namespace should have been created.
	ns := &corev1.Namespace{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: tsConfig.GetNamespace()}, ns)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("%s namespace should have been created", tsConfig.GetNamespace()))
	}

	// A subscription to the product operator should have been created.
	sub := &coreosv1alpha1.Subscription{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: packageName, Namespace: tsConfig.GetNamespace()}, sub)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("%s operator subscription was not created", packageName))
	}

	// The main s3credentials should have been copied into the 3scale namespace.
	s3Credentials := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: s3CredentialsSecretName, Namespace: tsConfig.GetNamespace()}, s3Credentials)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("s3Credentials were not copied into %s namespace", tsConfig.GetNamespace()))
	}

	// The product custom resource should have been created.
	apim := &threescalev1.APIManager{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: apiManagerName, Namespace: tsConfig.GetNamespace()}, apim)
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
	if !aerogearv1.ContainsClient(kcr.Spec.Clients, clientId) {
		return errors.New(fmt.Sprintf("Keycloak client '%s' was not created", clientId))
	}
	authProvider, err := fakeThreeScaleClient.GetAuthenticationProviderByName(rhssoIntegrationName, accessToken)
	if tsIsNotFoundError(err) {
		return errors.New(fmt.Sprintf("SSO integration was not created"))
	}
	if authProvider.ProviderDetails.ClientId != clientId || authProvider.ProviderDetails.Site != rhssoConfig.GetHost()+"/auth/realms/"+rhssoConfig.GetRealm() {
		return errors.New(fmt.Sprintf("SSO integration request to 3scale API was incorrect"))
	}

	// RHSSO CustomerAdmin admin user should be set as the default 3scale admin
	defaultAdminUser, err := fakeThreeScaleClient.GetUser(rhsso.CustomerAdminUser.UserName, accessToken)
	if err != nil {
		return err
	}
	if defaultAdminUser.UserDetails.Email != rhsso.CustomerAdminUser.Email {
		return errors.New(fmt.Sprintf("Request to 3scale API to update admin details was incorrect"))
	}
	adminSecret := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: threeScaleAdminDetailsSecret.Name, Namespace: tsConfig.GetNamespace()}, adminSecret)
	if string(adminSecret.Data["ADMIN_USER"]) != rhsso.CustomerAdminUser.UserName || string(adminSecret.Data["ADMIN_EMAIL"]) != rhsso.CustomerAdminUser.Email {
		return errors.New(fmt.Sprintf("3scale admin secret details were not updated"))
	}

	// Service discovery should be configured
	threeScaleOauth := &oauthv1.OAuthClient{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: oauthId}, threeScaleOauth)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("3scale should have an Oauth Client '%s' created", oauthId))
	}
	if threeScaleOauth.RedirectURIs[0] != installation.Spec.MasterURL {
		return errors.New(fmt.Sprintf("3scale Oauth Client redirect uri should be %s and is %s", installation.Spec.MasterURL, threeScaleOauth.RedirectURIs[0]))
	}

	serviceDiscoveryConfigMap := &corev1.ConfigMap{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: threeScaleServiceDiscoveryConfigMap.Name, Namespace: tsConfig.GetNamespace()}, serviceDiscoveryConfigMap)
	if string(adminSecret.Data["ADMIN_USER"]) != rhsso.CustomerAdminUser.UserName || string(adminSecret.Data["ADMIN_EMAIL"]) != rhsso.CustomerAdminUser.Email {
		return errors.New(fmt.Sprintf("3scale admin secret details were not updated"))
	}
	if string(serviceDiscoveryConfigMap.Data["service_discovery.yml"]) != sdConfig {
		return errors.New(fmt.Sprintf("Service discovery config is misconfigured"))
	}

	// rhsso users should be users in 3scale. If an rhsso user is also in rhmi-admins group that user should be an admin in 3scale.
	tsUsers, _ := fakeThreeScaleClient.GetUsers("accessToken")
	if len(tsUsers.Users) != len(kcr.Spec.Users) {
		return errors.New(fmt.Sprintf("Rhsso users should be mapped into 3scale users"))
	}
	test1User, _ := fakeThreeScaleClient.GetUser(rhssoTest1.UserName, "accessToken")
	if test1User.UserDetails.Role != adminRole {
		return errors.New(fmt.Sprintf("%s should be an admin user in 3scale", test1User.UserDetails.Username))
	}
	test2User, _ := fakeThreeScaleClient.GetUser(rhssoTest2.UserName, "accessToken")
	if test2User.UserDetails.Role != memberRole {
		return errors.New(fmt.Sprintf("%s should be a member user in 3scale", test2User.UserDetails.Username))
	}

	// system-app and system-sidekiq deploymentconfigs should have been rolled out on first reconcile.
	sa, err := fakeAppsV1Client.DeploymentConfigs(tsConfig.GetNamespace()).Get("system-app", metav1.GetOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting deplymentconfig: %v", err))
	}
	if sa.Status.LatestVersion == 1 {
		return errors.New(fmt.Sprintf("system-app was not rolled out"))
	}
	ssk, err := fakeAppsV1Client.DeploymentConfigs(tsConfig.GetNamespace()).Get("system-sidekiq", metav1.GetOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting deplymentconfig: %v", err))
	}
	if ssk.Status.LatestVersion == 1 {
		return errors.New(fmt.Sprintf("system-sidekiq was not rolled out"))
	}

	return nil
}
