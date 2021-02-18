package common

import (
	goctx "context"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	stageRestorationTimeOut        = 30 * time.Minute
	stageRetryInterval             = 15 * time.Second
	finalizerDeletionTimeout       = 2 * time.Minute
	finalizerDeletionRetryInterval = 10 * time.Second
)

type StageDeletion struct {
	productStageName integreatlyv1alpha1.StageName
	namespaces       []string
	removeFinalizers func(ctx *TestingContext) error
}

var (
	commonStages = []StageDeletion{
		{
			productStageName: integreatlyv1alpha1.AuthenticationStage,
			namespaces: []string{
				RHSSOProductNamespace,
				RHSSOOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return removeKeyCloakFinalizers(ctx, RHSSOProductNamespace)
			},
		},
		{
			productStageName: integreatlyv1alpha1.CloudResourcesStage,
			namespaces: []string{
				CloudResourceOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return nil
			},
		},
		{
			productStageName: integreatlyv1alpha1.MonitoringStage,
			namespaces: []string{
				MonitoringOperatorNamespace,
				MonitoringFederateNamespace,
				MonitoringSpecNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return removeMonitoringFinalizers(ctx, MonitoringOperatorNamespace)
			},
		},
	}

	rhmiSpecificStages = []StageDeletion{
		{
			productStageName: integreatlyv1alpha1.ProductsStage,
			namespaces: []string{
				AMQOnlineOperatorNamespace,
				ApicuritoProductNamespace,
				ApicuritoOperatorNamespace,
				CodeReadyProductNamespace,
				CodeReadyOperatorNamespace,
				FuseProductNamespace,
				FuseOperatorNamespace,
				RHSSOUserProductOperatorNamespace,
				RHSSOUserOperatorNamespace,
				ThreeScaleProductNamespace,
				ThreeScaleOperatorNamespace,
				UPSProductNamespace,
				UPSOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return removeKeyCloakFinalizers(ctx, RHSSOUserProductOperatorNamespace)
			},
		},
		{
			productStageName: integreatlyv1alpha1.SolutionExplorerStage,
			namespaces: []string{
				SolutionExplorerProductNamespace,
				SolutionExplorerOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return nil
			},
		},
	}

	managedApiStages = []StageDeletion{
		{
			productStageName: integreatlyv1alpha1.ProductsStage,
			namespaces: []string{
				CustomerGrafanaNamespace,
				Marin3rOperatorNamespace,
				Marin3rProductNamespace,
				RHSSOUserProductOperatorNamespace,
				RHSSOUserOperatorNamespace,
				ThreeScaleProductNamespace,
				ThreeScaleOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return removeKeyCloakFinalizers(ctx, RHSSOUserProductOperatorNamespace)
			},
		},
	}
)

func TestNamespaceRestoration(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatal(err)
	}

	for _, stage := range getStagesForInstallType(rhmi.Spec.Type) {

		// Delete all the namespaces defined in product stage
		for _, nameSpace := range stage.namespaces {
			nameSpaceForDeletion := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nameSpace,
					Namespace: nameSpace,
				},
			}

			err := ctx.Client.Delete(goctx.TODO(), nameSpaceForDeletion)

			if err != nil {
				t.Fatalf("Error deleting %s namespace: %s", nameSpace, err)
			}

			t.Logf("Deleted %s namespace", nameSpace)
		}

		// Remove any finalizers that may be preventing stage deletion
		err := stage.removeFinalizers(ctx)

		if err != nil {
			t.Fatalf("Failed to remove finalizers for stage %s: %s", stage.productStageName, err)
		}

		t.Logf("Success removing finalizers for %s stage", stage.productStageName)

		// Wait for product stage to be in progress
		err = waitForProductStageStatusInRHMI(t, ctx, stage.productStageName, integreatlyv1alpha1.PhaseInProgress)

		if err != nil {
			t.Fatalf("Failed to wait for %s stage to change to %s with error: %s", stage.productStageName, integreatlyv1alpha1.PhaseInProgress, err)
		}

		// Wait for product stage to complete
		err = waitForProductStageStatusInRHMI(t, ctx, stage.productStageName, integreatlyv1alpha1.PhaseCompleted)

		if err != nil {
			t.Fatalf("Failed to wait for %s stage to change to %s with error: %s", stage.productStageName, integreatlyv1alpha1.PhaseCompleted, err)
		}
	}
}

// Wait for the product stage to be a specific status
func waitForProductStageStatusInRHMI(t TestingTB, ctx *TestingContext, stage integreatlyv1alpha1.StageName, status integreatlyv1alpha1.StatusPhase) error {
	err := wait.Poll(stageRetryInterval, stageRestorationTimeOut, func() (done bool, err error) {
		rhmi, err := GetRHMI(ctx.Client, true)
		if err != nil {
			t.Logf("Got an error getting rhmi cr: %v", err)
			return false, err
		}

		if rhmi.Status.Stages[stage].Phase != status {
			t.Logf("Waiting for %s stage status to change to %s", stage, status)
			return false, nil
		}

		t.Logf("%s stage status changed to %s", stage, status)
		return true, nil
	})

	return err
}

// Remove finalizers from KeyCloak resources from a target namespace
func removeKeyCloakFinalizers(ctx *TestingContext, nameSpace string) error {
	err := removeKeyCloakClientFinalizers(ctx, nameSpace)
	if err != nil {
		return err
	}

	err = removeKeyCloakRealmFinalizers(ctx, nameSpace)
	if err != nil {
		return err
	}

	err = removeKeyCloakUserFinalizers(ctx, nameSpace)
	if err != nil {
		return err
	}

	return nil
}

// Poll removal of all finalizers from KeyCloakClients from a namespace
func removeKeyCloakClientFinalizers(ctx *TestingContext, nameSpace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		clients := &keycloakv1alpha1.KeycloakClientList{}

		err = ctx.Client.List(goctx.TODO(), clients, &k8sclient.ListOptions{
			Namespace: nameSpace,
		})

		if err != nil {
			return false, err
		}

		for _, client := range clients.Items {
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &client, func() error {
				client.Finalizers = []string{}
				return nil
			})
		}

		if err != nil {
			return false, err
		}

		return true, nil
	})

	return err
}

// Poll removal of all finalizers from KeyCloakRealms from a namespace
func removeKeyCloakRealmFinalizers(ctx *TestingContext, nameSpace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		realms := &keycloakv1alpha1.KeycloakRealmList{}

		err = ctx.Client.List(goctx.TODO(), realms, &k8sclient.ListOptions{
			Namespace: nameSpace,
		})

		if err != nil {
			return false, err
		}

		for _, realm := range realms.Items {
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &realm, func() error {
				realm.Finalizers = []string{}
				return nil
			})

			if err != nil {
				return false, err
			}
		}

		return true, nil
	})

	return err
}

// Poll removal of all finalizers from KeyCloakUsers from a namespace
func removeKeyCloakUserFinalizers(ctx *TestingContext, nameSpace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		users := &keycloakv1alpha1.KeycloakUserList{}

		err = ctx.Client.List(goctx.TODO(), users, &k8sclient.ListOptions{
			Namespace: nameSpace,
		})

		if err != nil {
			return false, err
		}

		for _, user := range users.Items {
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &user, func() error {
				user.Finalizers = []string{}
				return nil
			})

			if err != nil {
				return false, err
			}
		}

		return true, nil
	})

	return err
}

// Remove finalizers from Monitoring resources in a target namespace
func removeMonitoringFinalizers(ctx *TestingContext, nameSpace string) error {

	err := removeBlackBoxTargetFinalizers(ctx, nameSpace)
	if err != nil {
		return err
	}

	err = removeApplicationMonitoringFinalizers(ctx, nameSpace)
	if err != nil {
		return err
	}

	return nil
}

// Poll removal of all finalizers from BlackBoxTargets from a namespace
func removeBlackBoxTargetFinalizers(ctx *TestingContext, nameSpace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		blackBoxes := &monitoringv1alpha1.BlackboxTargetList{}

		err = ctx.Client.List(goctx.TODO(), blackBoxes, &k8sclient.ListOptions{
			Namespace: nameSpace,
		})

		if err != nil {
			return false, err
		}

		for _, blackBox := range blackBoxes.Items {
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &blackBox, func() error {
				blackBox.Finalizers = []string{}
				return nil
			})

			if err != nil {
				return false, err
			}
		}

		return true, nil
	})

	return err
}

// Poll removal of all finalizers from ApplicationMonitorings from a namespace
func removeApplicationMonitoringFinalizers(ctx *TestingContext, nameSpace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		applicationMonitorings := &monitoringv1alpha1.ApplicationMonitoringList{}

		err = ctx.Client.List(goctx.TODO(), applicationMonitorings, &k8sclient.ListOptions{
			Namespace: nameSpace,
		})

		if err != nil {
			return false, err
		}

		for _, applicationMonitoring := range applicationMonitorings.Items {
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &applicationMonitoring, func() error {
				applicationMonitoring.Finalizers = []string{}
				return nil
			})

			if err != nil {
				return false, err
			}
		}

		return true, nil
	})

	return err
}

func getStagesForInstallType(installType string) []StageDeletion {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return append(commonStages, managedApiStages...)
	} else {
		return append(commonStages, rhmiSpecificStages...)
	}
}
