package threescale

import (
	"context"
	"fmt"
	"net/http"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "3scale"
	packageName                  = "integreatly-3scale"
	apiManagerName               = "3scale"
	clientId                     = "3scale"
	oauthId                      = "3scale"
	clientSecret                 = "placeholder" // this should be replaced in INTLY-2784
	s3BucketSecretName           = "s3-bucket"
	s3CredentialsSecretName      = "s3-credentials"
	rhssoIntegrationName         = "rhsso"
	finalizer                    = "finalizer.3scale.integreatly.org"
)

var (
	sdConfig = fmt.Sprintf("production:\n  enabled: true\n  authentication_method: oauth\n  oauth_server_type: builtin\n  client_id: '%s'\n  client_secret: '%s'\n", oauthId, clientSecret)
)

func NewReconciler(configManager config.ConfigReadWriter, i *v1alpha1.Installation, appsv1Client appsv1Client.AppsV1Interface, oauthv1Client oauthClient.OauthV1Interface, tsClient ThreeScaleInterface, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	ns := i.Spec.NamespacePrefix + defaultInstallationNamespace
	tsConfig, err := configManager.ReadThreeScale()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve threescale config")
	}
	if tsConfig.GetNamespace() == "" {
		tsConfig.SetNamespace(ns)
		configManager.WriteConfig(tsConfig)
	}
	return &Reconciler{
		ConfigManager: configManager,
		Config:        tsConfig,
		mpm:           mpm,
		installation:  i,
		tsClient:      tsClient,
		appsv1Client:  appsv1Client,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *v1alpha1.Installation
	tsClient      ThreeScaleInterface
	appsv1Client  appsv1Client.AppsV1Interface
	oauthv1Client oauthClient.OauthV1Interface
	*resources.Reconciler
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-app",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, in *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", packageName)

	phase, err := r.ReconcileFinalizer(ctx, serverClient, in, product, finalizer, func() error {
		return resources.RemoveOauthClient(ctx, in, serverClient, r.oauthv1Client, finalizer, oauthId)
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), in, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, in, marketplace.Target{Pkg: packageName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	logrus.Infof("%s is successfully deployed", packageName)

	phase, err = r.reconcileRHSSOIntegration(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileUpdatingDefaultAdminDetails(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileOpenshiftUsers(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileOauthClient(ctx, in, &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: oauthId,
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			r.installation.Spec.MasterURL,
		},
		GrantMethod: oauthv1.GrantHandlerPrompt,
	}, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileServiceDiscovery(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	logrus.Infof("%s installation is reconciled successfully", packageName)
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	bucket := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: s3BucketSecretName,
		},
	}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: bucket.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, bucket)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	s3SecretName := s3CredentialsSecretName
	tsS3 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3SecretName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: tsS3.Name, Namespace: tsS3.Namespace}, tsS3)
	if err != nil && k8serr.IsNotFound(err) {
		// We are copying the s3 details for now but this is not ideal as the secrets can get out of sync.
		// We need to revise how this secret is set
		s3 := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: s3SecretName,
			},
		}
		err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: s3.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, s3)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

		tsS3.Data = s3.Data
		err = serverClient.Create(ctx, tsS3)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
	}

	resourceRequirements := false
	apim := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiManagerName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: threescalev1.APIManagerSpec{
			APIManagerCommonSpec: threescalev1.APIManagerCommonSpec{
				WildcardDomain:              r.installation.Spec.RoutingSubdomain,
				ResourceRequirementsEnabled: &resourceRequirements,
			},
			System: &threescalev1.SystemSpec{
				FileStorageSpec: &threescalev1.SystemFileStorageSpec{
					S3: &threescalev1.SystemS3Spec{
						AWSBucket: string(bucket.Data["AWS_BUCKET"]),
						AWSRegion: string(bucket.Data["AWS_REGION"]),
						AWSCredentials: v1.LocalObjectReference{
							Name: s3SecretName,
						},
					},
				},
			},
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: apim.Name, Namespace: r.Config.GetNamespace()}, apim)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}

	if err != nil {
		logrus.Infof("Creating API Manager")
		err := serverClient.Create(ctx, apim)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
	}

	if len(apim.Status.Deployments.Starting) == 0 && len(apim.Status.Deployments.Stopped) == 0 && len(apim.Status.Deployments.Ready) > 0 {
		return v1alpha1.PhaseCompleted, nil
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) reconcileRHSSOIntegration(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	rhssoNamespace := rhssoConfig.GetNamespace()
	rhssoRealm := rhssoConfig.GetRealm()
	if rhssoNamespace == "" || rhssoRealm == "" {
		logrus.Info("Cannot configure SSO integration without SSO namespace and SSO realm")
		return v1alpha1.PhaseInProgress, nil
	}

	kcr := &aerogearv1.KeycloakRealm{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: rhssoRealm, Namespace: rhssoNamespace}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if !aerogearv1.ContainsClient(kcr.Spec.Clients, clientId) {
		logrus.Infof("Adding keycloak realm client")

		kcr.Spec.Clients = append(kcr.Spec.Clients, &aerogearv1.KeycloakClient{
			KeycloakApiClient: &aerogearv1.KeycloakApiClient{
				ID:                      clientId,
				ClientID:                clientId,
				Enabled:                 true,
				Secret:                  clientSecret,
				ClientAuthenticatorType: "client-secret",
				RedirectUris: []string{
					fmt.Sprintf("https://3scale-admin.%s/*", r.installation.Spec.RoutingSubdomain),
				},
				StandardFlowEnabled: true,
				RootURL:             fmt.Sprintf("https://3scale-admin.%s", r.installation.Spec.RoutingSubdomain),
				FullScopeAllowed:    true,
				Access: map[string]bool{
					"view":      true,
					"configure": true,
					"manage":    true,
				},
				ProtocolMappers: []aerogearv1.KeycloakProtocolMapper{
					{
						Name:            "given name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${givenName}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "firstName",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "given_name",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "email verified",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${emailVerified}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "emailVerified",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "email_verified",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "full name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-full-name-mapper",
						ConsentRequired: true,
						ConsentText:     "${fullName}",
						Config: map[string]string{
							"id.token.claim":     "true",
							"access.token.claim": "true",
						},
					},
					{
						Name:            "family name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${familyName}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "lastName",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "family_name",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "role list",
						Protocol:        "saml",
						ProtocolMapper:  "saml-role-list-mapper",
						ConsentRequired: false,
						ConsentText:     "${familyName}",
						Config: map[string]string{
							"single":               "false",
							"attribute.nameformat": "Basic",
							"attribute.name":       "Role",
						},
					},
					{
						Name:            "email",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${email}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "email",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "email",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "org_name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: false,
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "org_name",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "org_name",
							"jsonType.label":       "String",
						},
					},
				},
			},
			OutputSecret: clientId + "-secret",
		})

		err = serverClient.Update(ctx, kcr)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
	}

	r.Config.SetHost(fmt.Sprintf("https://3scale-admin.%s", r.installation.Spec.RoutingSubdomain))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	accessToken, err := r.GetAdminToken(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	_, err = r.tsClient.GetAuthenticationProviderByName(rhssoIntegrationName, *accessToken)
	if err != nil && !tsIsNotFoundError(err) {
		return v1alpha1.PhaseFailed, err
	}
	if tsIsNotFoundError(err) {
		site := rhssoConfig.GetHost() + "/auth/realms/" + rhssoRealm
		res, err := r.tsClient.AddAuthenticationProvider(map[string]string{
			"kind":                              "keycloak",
			"name":                              rhssoIntegrationName,
			"client_id":                         clientId,
			"client_secret":                     clientSecret,
			"site":                              site,
			"skip_ssl_certificate_verification": "true",
			"published":                         "true",
		}, *accessToken)
		if err != nil || res.StatusCode != http.StatusCreated {
			return v1alpha1.PhaseFailed, err
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileUpdatingDefaultAdminDetails(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	rhssoNamespace := rhssoConfig.GetNamespace()
	rhssoRealm := rhssoConfig.GetRealm()
	if rhssoNamespace == "" || rhssoRealm == "" {
		logrus.Info("Cannot update admin details without SSO namespace and SSO realm")
		return v1alpha1.PhaseInProgress, nil
	}

	kcr := &aerogearv1.KeycloakRealm{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: rhssoRealm, Namespace: rhssoNamespace}, kcr)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}

	kcUsers := filterUsers(kcr.Spec.Users, func(u *aerogearv1.KeycloakUser) bool {
		return u.UserName == rhsso.CustomerAdminUser.UserName
	})
	if len(kcUsers) == 1 {
		s := &corev1.Secret{}
		err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: "system-seed", Namespace: r.Config.GetNamespace()}, s)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

		currentAdminUser := string(s.Data["ADMIN_USER"])
		accessToken, err := r.GetAdminToken(ctx, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
		tsAdmin, err := r.tsClient.GetUser(currentAdminUser, *accessToken)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

		kcCaUser := kcUsers[0]
		if tsAdmin.UserDetails.Username != kcCaUser.UserName && tsAdmin.UserDetails.Email != kcCaUser.Email {
			res, err := r.tsClient.UpdateUser(tsAdmin.UserDetails.Id, kcCaUser.UserName, kcCaUser.Email, *accessToken)
			if err != nil || res.StatusCode != http.StatusOK && res.StatusCode != http.StatusUnprocessableEntity {
				return v1alpha1.PhaseFailed, err
			}
		}

		currentUsername, currentEmail, err := r.GetAdminNameAndPassFromSecret(ctx, serverClient)
		if *currentUsername != kcCaUser.UserName || *currentEmail != kcCaUser.Email {
			err = r.SetAdminDetailsOnSecret(ctx, serverClient, kcCaUser.UserName, kcCaUser.Email)
			if err != nil {
				return v1alpha1.PhaseFailed, err
			}

			err = r.RolloutDeployment("system-app")
			if err != nil {
				return v1alpha1.PhaseFailed, err
			}
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileOpenshiftUsers(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling openshift users to 3scale")

	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	kcr := &aerogearv1.KeycloakRealm{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: rhssoConfig.GetRealm(), Namespace: rhssoConfig.GetNamespace()}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	accessToken, err := r.GetAdminToken(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	tsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	added, deleted := r.getUserDiff(kcr.Spec.Users, tsUsers.Users)
	for _, kcUser := range added {
		if kcUser.UserName == rhsso.CustomerAdminUser.UserName {
			continue
		}

		res, err := r.tsClient.AddUser(kcUser.UserName, kcUser.Email, "", *accessToken)
		if err != nil || res.StatusCode != http.StatusCreated {
			return v1alpha1.PhaseFailed, err
		}
	}
	for _, tsUser := range deleted {
		res, err := r.tsClient.DeleteUser(tsUser.UserDetails.Id, *accessToken)
		if err != nil || res.StatusCode != http.StatusOK {
			return v1alpha1.PhaseFailed, err
		}
	}

	openshiftAdminGroup := &usersv1.Group{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: "dedicated-admins"}, openshiftAdminGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}
	newTsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	for _, tsUser := range newTsUsers.Users {
		if tsUser.UserDetails.Username == rhsso.CustomerAdminUser.UserName {
			continue
		}

		if userIsOpenshiftAdmin(tsUser, openshiftAdminGroup) && tsUser.UserDetails.Role != adminRole {
			res, err := r.tsClient.SetUserAsAdmin(tsUser.UserDetails.Id, *accessToken)
			if err != nil || res.StatusCode != http.StatusOK {
				return v1alpha1.PhaseFailed, err
			}
		} else if !userIsOpenshiftAdmin(tsUser, openshiftAdminGroup) && tsUser.UserDetails.Role != memberRole {
			res, err := r.tsClient.SetUserAsMember(tsUser.UserDetails.Id, *accessToken)
			if err != nil || res.StatusCode != http.StatusOK {
				return v1alpha1.PhaseFailed, err
			}
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileServiceDiscovery(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	cm := &corev1.ConfigMap{}
	// if this errors it can be ignored
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: "system-environment", Namespace: r.Config.GetNamespace()}, cm)
	if err == nil && string(r.Config.GetProductVersion()) != cm.Data["AMP_RELEASE"] {
		r.Config.SetProductVersion(cm.Data["AMP_RELEASE"])
		r.ConfigManager.WriteConfig(r.Config)
	}

	system := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system",
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: system.Name, Namespace: system.Namespace}, system)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if system.Data["service_discovery.yml"] != sdConfig {
		system.Data["service_discovery.yml"] = sdConfig
		err := serverClient.Update(ctx, system)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

		err = r.RolloutDeployment("system-app")
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

		err = r.RolloutDeployment("system-sidekiq")
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) GetAdminNameAndPassFromSecret(ctx context.Context, serverClient pkgclient.Client) (*string, *string, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "system-seed",
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: s.Name, Namespace: r.Config.GetNamespace()}, s)
	if err != nil {
		return nil, nil, err
	}

	username := string(s.Data["ADMIN_USER"])
	email := string(s.Data["ADMIN_EMAIL"])
	return &username, &email, nil
}

func (r *Reconciler) SetAdminDetailsOnSecret(ctx context.Context, serverClient pkgclient.Client, username string, email string) error {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "system-seed",
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: s.Name, Namespace: r.Config.GetNamespace()}, s)
	if err != nil {
		return err
	}

	currentAdminUser := string(s.Data["ADMIN_USER"])
	currentAdminEmail := string(s.Data["ADMIN_EMAIL"])
	if currentAdminUser == username && currentAdminEmail == email {
		return nil
	}

	s.Data["ADMIN_USER"] = []byte(username)
	s.Data["ADMIN_EMAIL"] = []byte(email)
	err = serverClient.Update(ctx, s)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) GetAdminToken(ctx context.Context, serverClient pkgclient.Client) (*string, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: s.Name, Namespace: r.Config.GetNamespace()}, s)
	if err != nil {
		return nil, err
	}

	accessToken := string(s.Data["ADMIN_ACCESS_TOKEN"])
	return &accessToken, nil
}

func (r *Reconciler) RolloutDeployment(name string) error {
	_, err := r.appsv1Client.DeploymentConfigs(r.Config.GetNamespace()).Instantiate(name, &appsv1.DeploymentRequest{
		Name:   name,
		Force:  true,
		Latest: true,
	})

	return err
}

type predicateFunc func(*aerogearv1.KeycloakUser) bool

func filterUsers(u []*aerogearv1.KeycloakUser, predicate predicateFunc) []*aerogearv1.KeycloakUser {
	var result []*aerogearv1.KeycloakUser
	for _, s := range u {
		if predicate(s) {
			result = append(result, s)
		}
	}

	return result
}

func (r *Reconciler) getUserDiff(kcUsers []*aerogearv1.KeycloakUser, tsUsers []*User) ([]*aerogearv1.KeycloakUser, []*User) {
	var added []*aerogearv1.KeycloakUser
	for _, kcUser := range kcUsers {
		if !tsContainsKc(tsUsers, kcUser) {
			added = append(added, kcUser)
		}
	}

	var deleted []*User
	for _, tsUser := range tsUsers {
		if !kcContainsTs(kcUsers, tsUser) {
			deleted = append(deleted, tsUser)
		}
	}

	return added, deleted
}

func kcContainsTs(kcUsers []*aerogearv1.KeycloakUser, tsUser *User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == tsUser.UserDetails.Username {
			return true
		}
	}

	return false
}

func tsContainsKc(tsusers []*User, kcUser *aerogearv1.KeycloakUser) bool {
	for _, tsu := range tsusers {
		if tsu.UserDetails.Username == kcUser.UserName {
			return true
		}
	}

	return false
}

func userIsOpenshiftAdmin(tsUser *User, adminGroup *usersv1.Group) bool {
	for _, userName := range adminGroup.Users {
		if tsUser.UserDetails.Username == userName {
			return true
		}
	}

	return false
}
