package common

import (
	goctx "context"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	keycloakv1alpha1 "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	observabilityoperator "github.com/redhat-developer/observability-operator/v3/api/v1"
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
	}

	managedApiStages = []StageDeletion{
		{
			productStageName: integreatlyv1alpha1.ProductsStage,
			namespaces: []string{
				CustomerGrafanaNamespace,
				Marin3rOperatorNamespace,
				Marin3rProductNamespace,
				RHSSOUserProductNamespace,
				RHSSOUserOperatorNamespace,
				ThreeScaleProductNamespace,
				ThreeScaleOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				if err := removeKeyCloakFinalizers(ctx, RHSSOUserProductNamespace); err != nil {
					return err
				}

				return removeEnvoyConfigRevisionFinalizers(ctx, ThreeScaleProductNamespace)
			},
		},
		{
			productStageName: integreatlyv1alpha1.ObservabilityStage,
			namespaces: []string{
				ObservabilityOperatorNamespace,
				ObservabilityProductNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return removeObservabilityFinalizers(ctx, ObservabilityProductNamespace)
			},
		},
	}

	mtManagedApiStages = []StageDeletion{
		{
			productStageName: integreatlyv1alpha1.ProductsStage,
			namespaces: []string{
				CustomerGrafanaNamespace,
				Marin3rOperatorNamespace,
				Marin3rProductNamespace,
				ThreeScaleProductNamespace,
				ThreeScaleOperatorNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				if err := removeKeyCloakFinalizers(ctx, RHSSOUserProductNamespace); err != nil {
					return err
				}

				return removeEnvoyConfigRevisionFinalizers(ctx, ThreeScaleProductNamespace)
			},
		},
		{
			productStageName: integreatlyv1alpha1.ObservabilityStage,
			namespaces: []string{
				ObservabilityOperatorNamespace,
				ObservabilityProductNamespace,
			},
			removeFinalizers: func(ctx *TestingContext) error {
				return removeObservabilityFinalizers(ctx, ObservabilityProductNamespace)
			},
		},
	}
)

func TestNamespaceRestoration(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatal(err)
	}

	for _, stage := range getStagesForInstallType(rhmi.Spec.Type, ctx) {

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

		for i := range clients.Items {
			client := clients.Items[i]
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

		for i := range realms.Items {
			realm := realms.Items[i]
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

		for i := range users.Items {
			user := users.Items[i]
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

// Poll removal of all finalizers from EnvoyConfigRevisions from a namespace
func removeEnvoyConfigRevisionFinalizers(ctx *TestingContext, nameSpace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		envoyConfigRevisions := &marin3rv1alpha1.EnvoyConfigRevisionList{}

		err = ctx.Client.List(goctx.TODO(), envoyConfigRevisions, &k8sclient.ListOptions{
			Namespace: nameSpace,
		})

		if err != nil {
			return false, err
		}

		for i := range envoyConfigRevisions.Items {
			envoyConfigRevision := envoyConfigRevisions.Items[i]
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &envoyConfigRevision, func() error {
				envoyConfigRevision.Finalizers = []string{}
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

func getStagesForInstallType(installType string, ctx *TestingContext) []StageDeletion {
	if GetPlatformType(ctx) == string(configv1.GCPPlatformType) {
		commonStages = append(commonStages, []StageDeletion{
			{
				productStageName: integreatlyv1alpha1.ProductsStage,
				namespaces: []string{
					McgOperatorNamespace,
				},
				removeFinalizers: func(ctx *TestingContext) error {
					return nil
				},
			},
		}...)
	}
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return append(commonStages, mtManagedApiStages...)
	} else {
		return append(commonStages, managedApiStages...)
	}
}

func removeObservabilityFinalizers(ctx *TestingContext, namespace string) error {
	err := wait.Poll(finalizerDeletionRetryInterval, finalizerDeletionTimeout, func() (done bool, err error) {
		observabilityList := &observabilityoperator.ObservabilityList{}

		err = ctx.Client.List(goctx.TODO(), observabilityList, &k8sclient.ListOptions{
			Namespace: namespace,
		})

		if err != nil {
			return false, err
		}

		for i := range observabilityList.Items {
			observability := observabilityList.Items[i]
			_, err = controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, &observability, func() error {
				observability.Finalizers = []string{}
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
