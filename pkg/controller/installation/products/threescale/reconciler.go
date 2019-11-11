package threescale

import (
	"context"
	"fmt"
	"net/http"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "3scale"
	packageName                  = "integreatly-3scale"
	apiManagerName               = "3scale"
	clientId                     = "3scale"
	rhssoIntegrationName         = "rhsso"

	tier                           = "production"
	s3BucketSecretName             = "s3-bucket"
	s3CredentialsSecretName        = "s3-credentials"
	externalRedisSecretName        = "system-redis"
	externalBackendRedisSecretName = "backend-redis"
	externalPostgresSecretName     = "system-database"
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

	phase, err := r.ReconcileFinalizer(ctx, serverClient, in, product, func() error {
		return resources.RemoveOauthClient(ctx, in, serverClient, r.oauthv1Client, r.getOAuthClientName())
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), in, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	// setup smtp credential configmap
	if in.Spec.UseExternalResources {
		phase, err = r.reconcileSMTPCredentials(ctx, serverClient)
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = r.reconcileBlobStorage(ctx, serverClient)
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = r.reconcileExternalDatasources(ctx, serverClient)
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}
	}

	phase, err = r.ReconcilePullSecret(ctx, r.Config.GetNamespace(), "", in, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	version, err := resources.NewVersion(v1alpha1.OperatorVersion3Scale)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for launcher")
	}
	phase, err = r.ReconcileSubscription(ctx, in, marketplace.Target{Pkg: packageName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, serverClient, version)
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

	clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	phase, err = r.ReconcileOauthClient(ctx, in, &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
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

func (r *Reconciler) getOauthClientSecret(ctx context.Context, serverClient pkgclient.Client) (string, error) {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return "", errors.Wrapf(err, "Could not find %s Secret", oauthClientSecrets.Name)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return "", errors.Wrapf(err, "Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	return string(clientSecretBytes), nil
}

func (r *Reconciler) reconcileSMTPCredentials(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling smtp")
	ns := r.installation.Namespace

	// setup smtp credential set cr for the cloud resource operator
	smtpCredName := fmt.Sprintf("3scale-smtp-%s", r.installation.Name)
	smtpCred, err := croUtil.ReconcileSMTPCredentialSet(ctx, serverClient, r.installation.Spec.Type, tier, smtpCredName, ns, smtpCredName, ns, func(cr metav1.Object) error {
		resources.PrepareObject(cr, r.installation)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile smtp credential request")
	}

	// wait for the smtp credential set cr to reconcile
	if smtpCred.Status.Phase != crov1.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret containing smtp credentials
	credSec := &v1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: smtpCred.Status.SecretRef.Name, Namespace: smtpCred.Status.SecretRef.Namespace}, credSec)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get smtp credential secret")
	}
	smtpCfgMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "smtp",
			Namespace: r.Config.GetNamespace(),
		},
	}

	// reconcile the smtp configmap for 3scale
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, smtpCfgMap, func(existing runtime.Object) error {
		cm := existing.(*v1.ConfigMap)
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["address"] = string(credSec.Data["host"])
		cm.Data["authentication"] = "login"
		cm.Data["domain"] = fmt.Sprintf("3scale-admin.%s", r.installation.Spec.RoutingSubdomain)
		cm.Data["openssl.verify.mode"] = ""
		cm.Data["password"] = string(credSec.Data["password"])
		cm.Data["port"] = string(credSec.Data["port"])
		cm.Data["username"] = string(credSec.Data["username"])
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create or update 3scale smtp configmap")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	fss, err := r.getFileStorageSpec(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	// create the 3scale api manager
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
			HighAvailability: &threescalev1.HighAvailabilitySpec{
				Enabled: r.installation.Spec.UseExternalResources,
			},
			System: &threescalev1.SystemSpec{
				DatabaseSpec: &threescalev1.SystemDatabaseSpec{
					PostgreSQL: &threescalev1.SystemPostgreSQLSpec{},
				},
				FileStorageSpec: fss,
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

func (r *Reconciler) reconcileBlobStorage(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling blob storage")
	ns := r.installation.Namespace

	// setup blob storage cr for the cloud resource operator
	blobStorageName := fmt.Sprintf("3scale-s3-%s", r.installation.Name)
	blobStorage, err := croUtil.ReconcileBlobStorage(ctx, serverClient, r.installation.Spec.Type, tier, blobStorageName, ns, blobStorageName, ns, func(cr metav1.Object) error {
		resources.PrepareObject(cr, r.installation)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile blob storage request")
	}

	// wait for the blob storage cr to reconcile
	if blobStorage.Status.Phase != crov1.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileExternalDatasources(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling external datastores")
	ns := r.installation.Namespace

	// setup backend redis custom resource
	// this will be used by the cloud resources operator to provision an elasticache instance
	logrus.Info("Creating backend elasticache instance")
	backendRedisName := fmt.Sprintf("3scale-backend-redis-%s", r.installation.Name)
	backendRedis, err := croUtil.ReconcileRedis(ctx, serverClient, r.installation.Spec.Type, tier, backendRedisName, ns, backendRedisName, ns, func(cr metav1.Object) error {
		resources.PrepareObject(cr, r.installation)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile backend redis request")
	}

	// setup system redis custom resource
	// this will be used by the cloud resources operator to provision an elasticache instance
	logrus.Info("Creating system elasticache instance")
	systemRedisName := fmt.Sprintf("3scale-redis-%s", r.installation.Name)
	systemRedis, err := croUtil.ReconcileRedis(ctx, serverClient, r.installation.Spec.Type, tier, systemRedisName, ns, systemRedisName, ns, func(cr metav1.Object) error {
		resources.PrepareObject(cr, r.installation)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile system redis request")
	}

	// setup postgres cr for the cloud resource operator
	// this will be used by the cloud resources operator to provision an rds instance
	logrus.Info("Creating rds instance")
	postgresName := fmt.Sprintf("3scale-postgres-%s", r.installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, r.installation.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		resources.PrepareObject(cr, r.installation)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile postgres request")
	}

	// wait for the backend redis cr to reconcile
	if backendRedis.Status.Phase != crov1.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret created by the cloud resources operator
	// containing backend redis connection details
	credSec := &v1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: backendRedis.Status.SecretRef.Name, Namespace: backendRedis.Status.SecretRef.Namespace}, credSec)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get backend redis credential secret")
	}

	// create backend redis external connection secret needed for the 3scale apimanager
	backendRedisSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalBackendRedisSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, backendRedisSecret, func(existing runtime.Object) error {
		uri := credSec.Data["uri"]
		port := credSec.Data["port"]
		backendRedisSecret.Data["REDIS_STORAGE_URL"] = []byte(fmt.Sprintf("redis://%s:%s/0", uri, port))
		backendRedisSecret.Data["REDIS_QUEUES_URL"] = []byte(fmt.Sprintf("redis://%s:%s/1", uri, port))
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create or update 3scale %s connection secret", externalBackendRedisSecretName)
	}

	// wait for the system redis cr to reconcile
	if systemRedis.Status.Phase != crov1.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret created by the cloud resources operator
	// containing system redis connection details
	systemCredSec := &v1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: systemRedis.Status.SecretRef.Name, Namespace: systemRedis.Status.SecretRef.Namespace}, systemCredSec)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get system redis credential secret")
	}

	// create system redis external connection secret needed for the 3scale apimanager
	redisSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalRedisSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, redisSecret, func(existing runtime.Object) error {
		uri := systemCredSec.Data["uri"]
		port := systemCredSec.Data["port"]
		conn := fmt.Sprintf("redis://%s:%s/1", uri, port)
		redisSecret.Data["URL"] = []byte(conn)
		redisSecret.Data["MESSAGE_BUS_URL"] = []byte(conn)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create or update 3scale %s connection secret", externalRedisSecretName)
	}

	// wait for the postgres cr to reconcile
	if postgres.Status.Phase != crov1.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret containing redis credentials
	postgresCredSec := &v1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, postgresCredSec)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get postgres credential secret")
	}

	// create postgres external connection secret
	postgresSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalPostgresSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, postgresSecret, func(existing runtime.Object) error {
		username := postgresCredSec.Data["username"]
		password := postgresCredSec.Data["password"]
		url := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", username, password, postgresCredSec.Data["host"], postgresCredSec.Data["port"], postgresCredSec.Data["database"])

		postgresSecret.Data["URL"] = []byte(url)
		postgresSecret.Data["DB_USER"] = username
		postgresSecret.Data["DB_PASSWORD"] = password
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create or update 3scale %s connection secret", externalPostgresSecretName)
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getFileStorageSpec(ctx context.Context, serverClient pkgclient.Client) (*threescalev1.SystemFileStorageSpec, error) {
	// if cro is being used for blob storage
	if r.installation.Spec.UseExternalResources {
		return r.getBlobStorageFileStorageSpec(ctx, serverClient)
	}

	// get existing aws bucket secret
	bucket := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: s3BucketSecretName,
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: bucket.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, bucket)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get aws bucket secret in namespace %s", r.ConfigManager.GetOperatorNamespace())
	}

	// get existing aws credentials secret from the operator namespace
	operatorCredSec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3CredentialsSecretName,
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: operatorCredSec.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, operatorCredSec)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get aws credentials secret in namespace %s", r.ConfigManager.GetOperatorNamespace())
	}

	// copy it into the 3scale namespace
	namespaceCredSec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3CredentialsSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, namespaceCredSec, func(existing runtime.Object) error {
		namespaceCredSec.Data["AWS_ACCESS_KEY_ID"] = operatorCredSec.Data["credentialKeyID"]
		namespaceCredSec.Data["AWS_SECRET_ACCESS_KEY"] = operatorCredSec.Data["credentialSecretKey"]
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create or update blob storage aws credentials secret")
	}

	return &threescalev1.SystemFileStorageSpec{
		S3: &threescalev1.SystemS3Spec{
			AWSBucket: string(bucket.Data["AWS_BUCKET"]),
			AWSRegion: string(bucket.Data["AWS_REGION"]),
			AWSCredentials: v1.LocalObjectReference{
				Name: s3CredentialsSecretName,
			},
		},
	}, nil
}

func (r *Reconciler) getBlobStorageFileStorageSpec(ctx context.Context, serverClient pkgclient.Client) (*threescalev1.SystemFileStorageSpec, error) {
	// create blob storage cr
	blobStorage := &crov1.BlobStorage{}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: fmt.Sprintf("3scale-s3-%s", r.installation.Name), Namespace: r.installation.Namespace}, blobStorage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get blob storage custom resource")
	}

	// get blob storage connection secret
	blobStorageSec := &v1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: blobStorage.Status.SecretRef.Name, Namespace: blobStorage.Status.SecretRef.Namespace}, blobStorageSec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get blob storage connection secret")
	}

	// create s3 credentials secret
	credSec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3CredentialsSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, credSec, func(existing runtime.Object) error {
		credSec.Data["AWS_ACCESS_KEY_ID"] = blobStorageSec.Data["credentialKeyID"]
		credSec.Data["AWS_SECRET_ACCESS_KEY"] = blobStorageSec.Data["credentialSecretKey"]
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create or update blob storage aws credentials secret")
	}
	// return the file storage spec
	return &threescalev1.SystemFileStorageSpec{
		S3: &threescalev1.SystemS3Spec{
			AWSBucket: string(blobStorageSec.Data["bucketName"]),
			AWSRegion: string(blobStorageSec.Data["bucketRegion"]),
			AWSCredentials: v1.LocalObjectReference{
				Name: s3CredentialsSecretName,
			},
		},
	}, nil
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

		clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

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

	clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
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

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
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
	clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	sdConfig := fmt.Sprintf("production:\n  enabled: true\n  authentication_method: oauth\n  oauth_server_type: builtin\n  client_id: '%s'\n  client_secret: '%s'\n", r.getOAuthClientName(), clientSecret)

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
