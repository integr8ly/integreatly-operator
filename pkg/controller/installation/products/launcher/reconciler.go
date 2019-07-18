package launcher

import (
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "launcher"
)

type Reconciler struct {
	coreClient    kubernetes.Interface
	Config        *config.Launcher
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
}

func NewReconciler(coreClient *kubernetes.Clientset, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadLauncher()
	if err != nil {
		return nil, err
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	return &Reconciler{
		coreClient:    coreClient,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
	}, nil
}

func (r *Reconciler) Reconcile(inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	fmt.Println("Reconciling", inst.Spec)

	// TODO Need to add github_client_secret and github_client_id to the Installation CR

	// TODO Create launcher_oauth_github Secret

	// OauthClient
	return v1alpha1.PhaseInProgress, nil
}
