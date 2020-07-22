package subscription

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/controller/subscription/csvlocator"
	"github.com/integr8ly/integreatly-operator/pkg/controller/subscription/rhmiConfigs"
	"github.com/integr8ly/integreatly-operator/pkg/controller/subscription/webapp"
	"github.com/integr8ly/integreatly-operator/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	catalogsourceClient "github.com/integr8ly/integreatly-operator/pkg/resources/catalogsource"

	"github.com/sirupsen/logrus"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// IntegreatlyPackage - package name is used for Subsription name
	IntegreatlyPackage = "integreatly"
	CSVNamePrefix      = "integreatly-operator"
)

// Add creates a new Subscription Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconcile, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconcile)
}

func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	operatorNs := "redhat-rhmi-operator"

	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10

	client, err := k8sclient.New(restConfig, k8sclient.Options{})
	if err != nil {
		return nil, err
	}

	catalogSourceClient, err := catalogsourceClient.NewClient(context.TODO(), client)
	if err != nil {
		return nil, err
	}

	webappNotifierClient := webapp.NewLazyUpgradeNotifier(func() (k8sclient.Client, error) {
		restConfig := controllerruntime.GetConfigOrDie()
		return k8sclient.New(restConfig, k8sclient.Options{})
	})

	csvLocator := csvlocator.NewCachedCSVLocator(csvlocator.NewConditionalCSVLocator(
		csvlocator.SwitchLocators(
			csvlocator.ForReference,
			csvlocator.ForEmbedded,
		),
	))

	return &ReconcileSubscription{
		mgr:                 mgr,
		client:              client,
		scheme:              mgr.GetScheme(),
		operatorNamespace:   operatorNs,
		catalogSourceClient: catalogSourceClient,
		webbappNotifier:     webappNotifierClient,
		csvLocator:          csvLocator,
	}, nil
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("subscription-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &operatorsv1alpha1.Subscription{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &integreatlyv1alpha1.RHMIConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorsv1alpha1.Subscription{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSubscription implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSubscription{}

type ReconcileSubscription struct {
	client              k8sclient.Client
	scheme              *runtime.Scheme
	operatorNamespace   string
	mgr                 manager.Manager
	catalogSourceClient catalogsourceClient.CatalogSourceClientInterface
	webbappNotifier     webapp.UpgradeNotifier
	csvLocator          csvlocator.CSVLocator
}

// Reconcile will ensure that that Subscription object(s) have Manual approval for the upgrades
// In a namespaced installation of integreatly operator it will only reconcile Subscription of the integreatly operator itself
func (r *ReconcileSubscription) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// skip any Subscriptions that are not integreatly operator
	if request.Namespace != r.operatorNamespace ||
		(request.Name != IntegreatlyPackage && request.Name != "addon-rhmi") {
		logrus.Infof("not our subscription: %+v, %s", request, r.operatorNamespace)
		return reconcile.Result{}, nil
	}

	subscription := &operatorsv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), request.NamespacedName, subscription)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if subscription.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalManual {
		subscription.Spec.InstallPlanApproval = operatorsv1alpha1.ApprovalManual
		err = r.client.Update(context.TODO(), subscription)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// TODO: investigate a better approach to getting RHMI rather than hardcoding values
	installation := &integreatlyv1alpha1.RHMI{}
	err = r.client.Get(context.TODO(), k8sclient.ObjectKey{Name: "rhmi", Namespace: request.NamespacedName.Namespace}, installation)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	return r.HandleUpgrades(context.TODO(), subscription, installation)
}

func (r *ReconcileSubscription) HandleUpgrades(ctx context.Context, rhmiSubscription *operatorsv1alpha1.Subscription, installation *integreatlyv1alpha1.RHMI) (reconcile.Result, error) {
	if !rhmiConfigs.IsUpgradeAvailable(rhmiSubscription) {
		logrus.Infof("no upgrade available")

		if err := r.webbappNotifier.ClearNotification(); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	latestRHMIInstallPlan, err := rhmiConfigs.GetLatestInstallPlan(ctx, rhmiSubscription, r.client)
	if err != nil {
		if errors.IsNotFound(err) {
			// if installplan is not found trigger the creation of a new one
			err = rhmiConfigs.CreateInstallPlan(ctx, rhmiSubscription, r.client)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, err
	}

	latestRHMICSV, err := r.csvLocator.GetCSV(ctx, r.client, latestRHMIInstallPlan)
	if err != nil {
		return reconcile.Result{}, err
	}

	config := &integreatlyv1alpha1.RHMIConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-config",
			Namespace: r.operatorNamespace,
		},
	}
	err = r.client.Get(ctx, k8sclient.ObjectKey{Name: config.Name, Namespace: config.Namespace}, config)
	if err != nil {
		return reconcile.Result{}, err
	}

	// checks if the operator is running locally don't use the catalogsource
	csvFromCatalogSource := latestRHMICSV
	if os.Getenv(k8sutil.ForceRunModeEnv) != string(k8sutil.LocalRunMode) {
		objectKey := k8sclient.ObjectKey{
			Name:      rhmiSubscription.Spec.CatalogSource,
			Namespace: rhmiSubscription.Spec.CatalogSourceNamespace,
		}
		csvFromCatalogSource, err = r.catalogSourceClient.GetLatestCSV(objectKey, rhmiSubscription.Spec.Package, rhmiSubscription.Spec.Channel)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Error getting the csv from catalogsource %w", err)
		}
	}

	isInstallPlanDeleted := false
	currentOperatorVersionName := fmt.Sprintf("%s.v%s", CSVNamePrefix, version.Version)
	if csvFromCatalogSource.Spec.Replaces != currentOperatorVersionName {

		if csvFromCatalogSource.Spec.Replaces != latestRHMICSV.Spec.Replaces {
			err = rhmiConfigs.DeleteInstallPlan(ctx, latestRHMIInstallPlan, r.client)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("error deleting installplan %w", err)
			}

			isInstallPlanDeleted = true
			logrus.Info("Installplan deleted for the install of patch upgrade")

			err = rhmiConfigs.CreateInstallPlan(ctx, rhmiSubscription, r.client)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		if isInstallPlanDeleted {
			// Requeue reconciler until the installplan is recreated
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 10 * time.Second,
			}, nil
		}

		// gets the subscription wit the recreated installplan
		err := r.client.Get(ctx, k8sclient.ObjectKey{Name: rhmiSubscription.Name, Namespace: rhmiSubscription.Namespace}, rhmiSubscription)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	isServiceAffecting := rhmiConfigs.IsUpgradeServiceAffecting(latestRHMICSV)

	// checks if there is changes in the rhmiconfig cr and recalculate the status
	if reschedule, ok := config.Annotations[v1alpha1.RecalculateScheduleAnnotation]; ok && reschedule == "true" {
		err = rhmiConfigs.UpdateStatus(ctx, r.client, config, latestRHMIInstallPlan, rhmiSubscription.Status.CurrentCSV)
		if err != nil {
			return reconcile.Result{}, err
		}

		config.Annotations[v1alpha1.RecalculateScheduleAnnotation] = ""
		err = r.client.Update(ctx, config)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if isServiceAffecting && rhmiSubscription.Status.CurrentCSV != config.Status.TargetVersion {
		err = rhmiConfigs.UpdateStatus(ctx, r.client, config, latestRHMIInstallPlan, rhmiSubscription.Status.CurrentCSV)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	canUpgradeNow, err := rhmiConfigs.CanUpgradeNow(config, installation)
	if err != nil {
		return reconcile.Result{}, err
	}

	phase, err := r.webbappNotifier.NotifyUpgrade(config, latestRHMICSV.Spec.Version.String(), isServiceAffecting)
	if err != nil {
		return reconcile.Result{}, err
	}
	if phase == integreatlyv1alpha1.PhaseInProgress {
		logrus.Infof("WebApp instance not found yet, skipping upgrade addition")
	}

	if !isServiceAffecting || canUpgradeNow {
		eventRecorder := r.mgr.GetEventRecorderFor("RHMI Upgrade")

		err = rhmiConfigs.ApproveUpgrade(ctx, r.client, installation, latestRHMIInstallPlan, r.csvLocator, eventRecorder)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Requeue the reconciler until the RHMI subscription upgrade is complete
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}
	logrus.Infof("not automatically upgrading a Service Affecting Release")
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Minute,
	}, nil
}
