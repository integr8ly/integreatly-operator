package rhsso

import (
	"context"
	"errors"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/client-go/kubernetes"

	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var (
	defaultInstallationNamespace = "rhsso"
	keycloakName                 = "rhsso"
)

func NewReconciler(client pkgclient.Client, serverClient pkgclient.Client, coreClient *kubernetes.Clientset, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, clusterHasOLM bool) (*Reconciler, error) {
	config, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	var mpm marketplace.MarketplaceInterface
	if clusterHasOLM {
		mpm = marketplace.NewManager(client)
	}
	return &Reconciler{client: client,
		coreClient:    coreClient,
		serverClient:  serverClient,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
	}, nil
}

type Reconciler struct {
	client        pkgclient.Client
	serverClient  pkgclient.Client
	coreClient    *kubernetes.Clientset
	Config        *config.RHSSO
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
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase()
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do a dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of RHSSO failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for RHSSO: " + string(phase))
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
	err := r.client.Get(context.TODO(), pkgclient.ObjectKey{Name: r.Config.GetNamespace()}, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if ns.Status.Phase == v1.NamespaceActive {
		// 23/04/19 pbrookes: if mpm is nil we are not in an OLM environment, so do not create a subscription
		//instead skip to creating components and assume operator is set up already
		if r.mpm != nil {
			logrus.Infof("Creating subscription")
			return v1alpha1.PhaseCreatingSubscription, nil
		} else {
			logrus.Infof("jumping to component creation")
			return v1alpha1.PhaseCreatingComponents, nil
		}
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleCreatingSubscription() (v1alpha1.StatusPhase, error) {
	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		"rhsso",
		"integreatly",
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCreatingComponents, nil
}

func (r *Reconciler) handleCreatingComponents() (v1alpha1.StatusPhase, error) {
	logrus.Infof("Creating Components")
	logrus.Infof("Creating Keycloak")
	keycloak := &aerogearv1.Keycloak{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf(
				"%s/%s",
				aerogearv1.SchemeGroupVersion.Group,
				aerogearv1.SchemeGroupVersion.Version),
			Kind: aerogearv1.KeycloakKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: aerogearv1.KeycloakSpec{
			AdminCredentials: "",
			Plugins: []string{
				"keycloak-metrics-spi",
			},
			Backups:   []aerogearv1.KeycloakBackup{},
			Provision: true,
		},
	}

	err := r.client.Create(context.TODO(), keycloak)

	//if this fails due to no matches for kind, keep trying until OLM creates the CRDs
	if err != nil && strings.Contains(err.Error(), "no matches for kind") {
		logrus.Infof("no match for kind error")
		return v1alpha1.PhaseCreatingComponents, nil
	}

	if err != nil && !k8serr.IsAlreadyExists(err) {
		logrus.Infof("Keycloak create error")
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("checking ready status for rhsso")
	kc := &aerogearv1.Keycloak{}

	// We need to get the newly created Keycloak from the API server instead of the cache.
	// cache client is not updated.
	err := r.serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if kc.Status.Phase == "reconcile" {
		return v1alpha1.PhaseCompleted, nil
	}

	return v1alpha1.PhaseInProgress, nil
}
