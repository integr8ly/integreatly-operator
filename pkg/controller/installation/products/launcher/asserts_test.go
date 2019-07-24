package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"

	launcherv1alpha2 "github.com/fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AssertFunc func(LauncherTestScenario, *config.Manager, *marketplace.MarketplaceInterfaceMock) error

func assertNoop(LauncherTestScenario, *config.Manager, *marketplace.MarketplaceInterfaceMock) error {
	return nil
}

func assertInstallationSuccessfullyReconciled(scenario LauncherTestScenario, configManager *config.Manager, fakeMPM *marketplace.MarketplaceInterfaceMock) error {
	ctx := context.TODO()

	launcherConfig, err := configManager.ReadLauncher()
	if err != nil {
		return err
	}

	// A namespace should have been created.
	ns := &corev1.Namespace{}
	err = scenario.FakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: launcherConfig.GetNamespace()}, ns)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("%s namespace should have been created", launcherConfig.GetNamespace()))
	}

	// A subscription to the product operator should have been created.
	if len(fakeMPM.InstallOperatorCalls()) != 1 {
		return errors.New(fmt.Sprintf("%s operator subscription was not created", defaultSubscriptionName))
	}

	launcherInst := &launcherv1alpha2.Launcher{}
	err = scenario.FakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: defaultLauncherName, Namespace: launcherConfig.GetNamespace()}, launcherInst)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("%s custom resource was not created", defaultLauncherName))
	}
	if launcherInst.Spec.OpenShift.ConsoleURL != scenario.Installation.Spec.MasterURL {
		return errors.New(fmt.Sprintf("%s ConsoleURL not equal MasterURL ", defaultLauncherName))
	}

	// RHSSO integration should be configured.
	rhssoConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return errors.New("Error getting RHSSO config")
	}
	kcr := &aerogearv1.KeycloakRealm{}
	err = scenario.FakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: rhssoConfig.GetRealm(), Namespace: rhssoConfig.GetNamespace()}, kcr)
	if !aerogearv1.ContainsClient(kcr.Spec.Clients, clientId) {
		return errors.New(fmt.Sprintf("Keycloak client '%s' was not created", clientId))
	}

	// Launcher route should have been written to the Launcher config
	launcherRoute := &routev1.Route{}
	err = scenario.FakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: launcherRouteName, Namespace: launcherConfig.GetNamespace()}, launcherRoute)
	if err != nil {
		return err
	}

	launcherUrl := "https://" + launcherRoute.Spec.Host
	if launcherConfig.GetHost() != launcherUrl {
		return errors.New(fmt.Sprintf("Launcher url is incorrect: %s", launcherConfig.GetHost()))
	}

	return nil
}
