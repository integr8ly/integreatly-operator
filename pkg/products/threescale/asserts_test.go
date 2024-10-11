package threescale

import (
	"context"
	"errors"
	"fmt"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	apiMachineryTypes "k8s.io/apimachinery/pkg/types"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (t ThreeScaleTestScenario) assertInstallationSuccessful() error {
	ctx := context.TODO()
	accessToken := "test123"
	fakeSigsClient := t.fields.sigsClient
	installationCR := t.args.installation
	fakeThreeScaleClient := t.fields.threeScaleClient
	configManager := t.fields.fakeConfig

	tsConfig, err := configManager.ReadThreeScale()
	if err != nil {
		return err
	}

	oauthID := installationCR.Spec.NamespacePrefix + string(tsConfig.GetProductName())
	oauthClientSecret := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: configManager.GetOauthClientsSecretName(), Namespace: configManager.GetOperatorNamespace()}, oauthClientSecret)
	if err != nil {
		return err
	}
	sdConfig := fmt.Sprintf("production:\n  enabled: true\n  authentication_method: oauth\n  oauth_server_type: builtin\n  client_id: '%s'\n  client_secret: '%s'\n", oauthID, oauthClientSecret.Data[string(tsConfig.GetProductName())])

	// A ns should have been created.
	ns := &corev1.Namespace{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: tsConfig.GetNamespace()}, ns)
	if k8serr.IsNotFound(err) {
		return fmt.Errorf("%s ns should have been created", tsConfig.GetNamespace())
	}

	// A subscription to the product operator should have been created.
	sub := &operatorsv1alpha1.Subscription{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: constants.ThreeScaleSubscriptionName, Namespace: defaultInstallationNamespace}, sub)
	if k8serr.IsNotFound(err) {
		return fmt.Errorf("%s operator subscription was not created", constants.ThreeScaleSubscriptionName)
	}

	// The main s3credentials should have been copied into the 3scale ns.
	s3Credentials := &corev1.Secret{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: s3CredentialsSecretName, Namespace: tsConfig.GetNamespace()}, s3Credentials)
	if k8serr.IsNotFound(err) {
		return fmt.Errorf("s3Credentials were not copied into %s ns", tsConfig.GetNamespace())
	}

	// The product custom resource should have been created.
	apim := &threescalev1.APIManager{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: apiManagerName, Namespace: tsConfig.GetNamespace()}, apim)
	if k8serr.IsNotFound(err) {
		return fmt.Errorf("APIManager '%s' was not created", apiManagerName)
	}
	if apim.Spec.WildcardDomain != installationCR.Spec.RoutingSubdomain {
		return fmt.Errorf("APIManager wildCardDomain is misconfigured. '%s' should be '%s'", apim.Spec.WildcardDomain, installationCR.Spec.RoutingSubdomain)
	}

	// RHSSO integration should be configured.
	rhssoConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return errors.New("error getting RHSSO config")
	}
	authProvider, err := fakeThreeScaleClient.GetAuthenticationProviderByName(rhssoIntegrationName, accessToken)
	if tsIsNotFoundError(err) {
		return fmt.Errorf("SSO integration was not created")
	}
	if authProvider.ProviderDetails.ClientId != clientID || authProvider.ProviderDetails.Site != rhssoConfig.GetHost()+"/auth/realms/"+rhssoConfig.GetRealm() {
		return fmt.Errorf("SSO integration request to 3scale API was incorrect")
	}

	// Service discovery should be configured
	threeScaleOauth := &oauthv1.OAuthClient{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: oauthID}, threeScaleOauth)
	if k8serr.IsNotFound(err) {
		return fmt.Errorf("3scale should have an Oauth client '%s' created", oauthID)
	}
	if len(threeScaleOauth.RedirectURIs) == 0 || threeScaleOauth.RedirectURIs[0] == "" {
		return fmt.Errorf("3scale Oauth Client redirect uri should be set, but is empty")
	}

	serviceDiscoveryConfigMap := &corev1.ConfigMap{}
	err = fakeSigsClient.Get(ctx, k8sclient.ObjectKey{Name: threeScaleServiceDiscoveryConfigMap.Name, Namespace: tsConfig.GetNamespace()}, serviceDiscoveryConfigMap)
	if err != nil {
		return err
	}
	if string(serviceDiscoveryConfigMap.Data["service_discovery.yml"]) != sdConfig {
		return fmt.Errorf("service discovery config is misconfigured")
	}

	if v1alpha1.IsRHOAMSingletenant(v1alpha1.InstallationType(t.args.installation.Spec.Type)) {
		// rhsso users should be users in 3scale. If an rhsso user is also in dedicated-admins group that user should be an admin in 3scale.
		test1User, err := fakeThreeScaleClient.GetUser(rhssoTest1.Spec.User.UserName, "accessToken")
		if err != nil {
			return err
		}

		if test1User.UserDetails.Role != adminRole {
			return fmt.Errorf("%s should be an admin user in 3scale", test1User.UserDetails.Username)
		}

		test2User, err := fakeThreeScaleClient.GetUser(rhssoTest2.Spec.User.UserName, "accessToken")
		if err != nil {
			return err
		}

		if test2User.UserDetails.Role != memberRole {
			return fmt.Errorf("%s should be a member user in 3scale", test2User.UserDetails.Username)
		}
	}

	// system-app and system-sidekiq deployments should have been rolled out on first reconcile.
	systemAppDeployment := &appsv1.Deployment{}
	err = fakeSigsClient.Get(ctx, apiMachineryTypes.NamespacedName{Namespace: tsConfig.GetNamespace(), Name: "system-app"}, systemAppDeployment)
	if err != nil {
		return fmt.Errorf("error getting deplyment: %v", err)
	}

	if systemAppDeployment.Status.ObservedGeneration == 1 {
		return fmt.Errorf("system-app was not rolled out")
	}

	systemSidekiqDeployment := &appsv1.Deployment{}
	err = fakeSigsClient.Get(ctx, apiMachineryTypes.NamespacedName{Namespace: tsConfig.GetNamespace(), Name: "system-sidekiq"}, systemSidekiqDeployment)
	if err != nil {
		return fmt.Errorf("error getting deplyment: %v", err)
	}

	if systemSidekiqDeployment.Status.ObservedGeneration == 1 {
		return fmt.Errorf("system-sidekiq was not rolled out")
	}

	return nil
}
