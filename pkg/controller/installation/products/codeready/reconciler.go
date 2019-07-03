package codeready

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	errors2 "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	chev1 "github.com/integr8ly/integreatly-operator/pkg/apis/che/v1alpha1"
	keycloakv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
)

var (
	defaultInstallationNamespace = "codeready-workspaces"
	defaultClientName = "che-client"
	defaultCheClusterName = "integreatly-cluster"
)

func NewReconciler(client pkgclient.Client, rc *rest.Config, coreClient *kubernetes.Clientset, configManager config.ConfigReadWriter, instance *v1alpha1.Installation) (*Reconciler, error) {
	config, err := configManager.ReadCodeReady()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	mpm := marketplace.NewManager(client, rc)
	return &Reconciler{client: client,
		coreClient:    coreClient,
		restConfig:    rc,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
	}, nil
}

type Reconciler struct {
	client        pkgclient.Client
	restConfig    *rest.Config
	coreClient    *kubernetes.Clientset
	Config        *config.CodeReady
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
}

func (r *Reconciler) Reconcile(phase v1alpha1.StatusPhase) (v1alpha1.StatusPhase, error) {

	switch phase {
	case v1alpha1.PhaseNone:
		return r.handleNoPhase()
	case v1alpha1.PhaseAwaitingNS:
		return r.handleAwaitingNSPhase()
	case v1alpha1.PhaseCreatingSubscription:
		return r.handleCreatingSubscription()
	case v1alpha1.PhaseCreatingComponents:
		return r.handleCreatingComponents()
	case v1alpha1.PhaseAwaitingOperator:
		return r.handleAwaitingOperator()
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase()
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do a dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of CodeReady failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for CodeReady: " + string(phase))
	}
}

func (r *Reconciler) handleNoPhase() (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      r.Config.GetNamespace(),
		},
	}
	err := r.client.Create(context.TODO(), ns)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleAwaitingNSPhase() (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}
	err := r.client.Get(context.TODO(), pkgclient.ObjectKey{ Name: r.Config.GetNamespace() }, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, nil
	}
	if ns.Status.Phase == v1.NamespaceActive {
		return v1alpha1.PhaseCreatingSubscription, nil
	}
	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleCreatingSubscription() (v1alpha1.StatusPhase, error) {
	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		"codeready-workspaces",
		"integreatly",
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseAwaitingOperator, nil
}

func (r *Reconciler) handleCreatingComponents() (v1alpha1.StatusPhase, error) {
	serverClient, err := pkgclient.New(r.restConfig, pkgclient.Options{})
	if err != nil {
		logrus.Infof("Error creating server client")
		return v1alpha1.PhaseFailed, err
	}

	kcCfg, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if kcCfg.GetRealm() == "" {
		return v1alpha1.PhaseFailed, errors.New("Keycloak config Realm is not set")
	}
	if kcCfg.GetURL() == "" {
		return v1alpha1.PhaseFailed, errors.New("Keycloak config URL is not set")
	}

	codeready := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf(
				"%s/%s",
				chev1.SchemeGroupVersion.Group,
				chev1.SchemeGroupVersion.Version,
			),
			Kind: "CheCluster",
		},
		Spec: chev1.CheClusterSpec{
			Server: chev1.CheClusterSpecServer{
				CheFlavor: "codeready",
				TlsSupport: true,
				SelfSignedCert: false,
			},
			Database: chev1.CheClusterSpecDB{
				ExternalDB: false,
				ChePostgresDBHostname: "",
				ChePostgresPassword: "",
				ChePostgresPort: "",
				ChePostgresUser: "",
				ChePostgresDb: "",
			},
			Auth: chev1.CheClusterSpecAuth{
				OpenShiftOauth: false,
				ExternalKeycloak: true,
				KeycloakURL: kcCfg.GetURL(),
				KeycloakRealm: kcCfg.GetRealm(),
				KeycloakClientId: defaultClientName,
			},
			Storage: chev1.CheClusterSpecStorage{
				PvcStrategy: "per-workspace",
				PvcClaimSize: "1Gi",
				PreCreateSubPaths: true,
			},
		},
	}
	err = serverClient.Create(context.TODO(), codeready)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleAwaitingOperator() (v1alpha1.StatusPhase, error) {
	ip, err := r.mpm.GetSubscriptionInstallPlan("codeready-workspaces", r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			logrus.Infof("No installplan created yet")
			return v1alpha1.PhaseAwaitingOperator, nil
		}

		logrus.Infof("Error getting codeready subscription installplan")
		return v1alpha1.PhaseFailed, err
	}

	if ip.Status.Phase != coreosv1alpha1.InstallPlanPhaseComplete {
		logrus.Infof("codeready installplan phase is %s", ip.Status.Phase)
		return v1alpha1.PhaseAwaitingOperator, nil
	}

	logrus.Infof("codeready installplan phase is %s", coreosv1alpha1.InstallPlanPhaseComplete)

	return v1alpha1.PhaseCreatingComponents, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	// check CodeReady is in ready state
	pods, err := r.coreClient.CoreV1().Pods(r.Config.GetNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return v1alpha1.PhaseFailed, errors2.Wrap(err, "Failed to check CodeReady installation")
	}

	//expecting 3 pods in total
	if len(pods.Items) < 3 {
		return v1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
checkPodStatus:
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == v1.ContainersReady {
				if cnd.Status != v1.ConditionStatus("True") {
					logrus.Infof("pod not ready, returning in progress: %+v", cnd.Status)
					return v1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}

	serverClient, err := pkgclient.New(r.restConfig, pkgclient.Options{})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	serverClient.Get(context.TODO(), pkgclient.ObjectKey{ Name: defaultCheClusterName, Namespace: r.Config.GetNamespace() }, cheCluster)
	cheURL := cheCluster.Status.CheURL
	if cheURL == "" {
		return v1alpha1.PhaseFailed, errors.New("che URL is not set")
	}

	kcCfg, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if kcCfg.GetRealm() == "" {
		return v1alpha1.PhaseFailed, errors.New("keycloak config Realm is not set")
	}
	if kcCfg.GetURL() == "" {
		return v1alpha1.PhaseFailed, errors.New("keycloak config URL is not set")
	}

	kcRealm := &keycloakv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name: kcCfg.GetRealm(),
			Namespace: kcCfg.GetNamespace(),
		},
	}
	serverClient.Get(context.TODO(), pkgclient.ObjectKey{ Name: kcCfg.GetRealm(), Namespace: kcCfg.GetNamespace() }, kcRealm)

	kcCheClient := &keycloakv1.KeycloakClient{}
	if existingKcCheClient := findCheKeycloakClient(kcRealm.Spec.Clients); existingKcCheClient != nil {
		kcCheClient = existingKcCheClient
	}

	kcCheClient.KeycloakApiClient = &keycloakv1.KeycloakApiClient{
		ID: defaultClientName,
		Name: defaultClientName,
		Enabled: true,
		StandardFlowEnabled: true,
		DirectAccessGrantsEnabled: true,
		RootURL: cheURL,
		RedirectUris: []string{ cheURL, fmt.Sprintf("%s/*", cheURL) },
		AdminURL: cheURL,
		WebOrigins: []string{ cheURL, fmt.Sprintf("%s/*", cheURL) },
		PublicClient: true,
		ClientAuthenticatorType: "client-secret",
	}
	kcRealm.Spec.Clients = findOrAppendCheKeycloakClient(kcRealm.Spec.Clients, kcCheClient)
	serverClient.Update(context.TODO(), kcRealm)

	return v1alpha1.PhaseCompleted, nil
}

func findOrAppendCheKeycloakClient(clients []*keycloakv1.KeycloakClient, toAppend *keycloakv1.KeycloakClient) []*keycloakv1.KeycloakClient {
	if existingKcCheClient := findCheKeycloakClient(clients); existingKcCheClient != nil {
		return clients
	}
	return append(clients, toAppend)
}

func findCheKeycloakClient(clients []*keycloakv1.KeycloakClient) *keycloakv1.KeycloakClient {
	for _, client := range clients {
		if client.ID == defaultClientName {
			return client
		}
	}
	return nil
}