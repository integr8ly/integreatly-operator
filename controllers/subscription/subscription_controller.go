package controllers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/resources"

	"github.com/blang/semver"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/csvlocator"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/rhmiConfigs"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/webapp"
	"github.com/integr8ly/integreatly-operator/version"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	catalogsourceClient "github.com/integr8ly/integreatly-operator/pkg/resources/catalogsource"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	pkgerr "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "subscription_controller"})

const (
	// IntegreatlyPackage - package name is used for Subscription name
	IntegreatlyPackage              = "integreatly"
	RHMIAddonSubscription           = "addon-rhmi"
	RHMIAddonSubscriptionEdge       = "addon-rhmi-internal"
	ManagedAPIAddonSubscription     = "addon-managed-api-service"
	ManagedAPIAddonSubscriptionEdge = "addon-managed-api-service-internal"
)

var subscriptionsToReconcile []string = []string{
	IntegreatlyPackage,
	RHMIAddonSubscription,
	ManagedAPIAddonSubscription,
	RHMIAddonSubscriptionEdge,
	ManagedAPIAddonSubscriptionEdge,
}

func New(mgr manager.Manager) (*SubscriptionReconciler, error) {
	watchNS, err := resources.GetWatchNamespace()
	if err != nil {
		return nil, pkgerr.Wrap(err, "could not get watch namespace from subscription controller")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	operatorNs := namespacePrefix + "operator"

	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10

	client, err := k8sclient.New(restConfig, k8sclient.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		return nil, err
	}

	catalogSourceClient, err := catalogsourceClient.NewClient(context.TODO(), client, log)
	if err != nil {
		return nil, err
	}

	webappNotifierClient := webapp.NewLazyUpgradeNotifier(func() (k8sclient.Client, error) {
		restConfig := controllerruntime.GetConfigOrDie()
		return k8sclient.New(restConfig, k8sclient.Options{
			Scheme: mgr.GetScheme(),
		})
	})

	csvLocator := csvlocator.NewCachedCSVLocator(csvlocator.NewConditionalCSVLocator(
		csvlocator.SwitchLocators(
			csvlocator.ForReference,
			csvlocator.ForEmbedded,
		),
	))

	return &SubscriptionReconciler{
		mgr:                 mgr,
		Client:              client,
		Scheme:              mgr.GetScheme(),
		operatorNamespace:   operatorNs,
		catalogSourceClient: catalogSourceClient,
		webbappNotifier:     webappNotifierClient,
		csvLocator:          csvLocator,
	}, nil
}

type SubscriptionReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme

	operatorNamespace   string
	mgr                 manager.Manager
	catalogSourceClient catalogsourceClient.CatalogSourceClientInterface
	webbappNotifier     webapp.UpgradeNotifier
	csvLocator          csvlocator.CSVLocator
}

// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions;subscriptions/status,verbs=get;list;watch;update;patch;delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups=operators.coreos.com,resources=installplans,verbs=get;list;watch;update;patch;delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;delete;list,namespace=integreatly-operator

// Reconcile will ensure that that Subscription object(s) have Manual approval for the upgrades
// In a namespaced installation of integreatly operator it will only reconcile Subscription of the integreatly operator itself
func (r *SubscriptionReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	// skip any Subscriptions that are not integreatly operator
	if !r.shouldReconcileSubscription(request) {
		log.Infof("Not our subscription", l.Fields{"request": request, "opNS": r.operatorNamespace})
		return ctrl.Result{}, nil
	}

	subscription := &operatorsv1alpha1.Subscription{}
	err := r.Get(context.TODO(), request.NamespacedName, subscription)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if subscription.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalManual {
		subscription.Spec.InstallPlanApproval = operatorsv1alpha1.ApprovalManual
		err = r.Update(context.TODO(), subscription)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	rhmiCr, err := resources.GetRhmiCr(r.Client, context.TODO(), request.NamespacedName.Namespace, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	if rhmiCr == nil {
		// Request object not found, could have been deleted after reconcile request. Return and don't requeue
		return ctrl.Result{}, nil
	}

	return r.HandleUpgrades(context.TODO(), subscription, rhmiCr)
}

func (r *SubscriptionReconciler) shouldReconcileSubscription(request ctrl.Request) bool {
	if request.Namespace != r.operatorNamespace {
		return false
	}

	for _, reconcileable := range subscriptionsToReconcile {
		if request.Name == reconcileable {
			return true
		}
	}

	return false
}

func (r *SubscriptionReconciler) HandleUpgrades(ctx context.Context, rhmiSubscription *operatorsv1alpha1.Subscription, installation *integreatlyv1alpha1.RHMI) (ctrl.Result, error) {
	if !rhmiConfigs.IsUpgradeAvailable(rhmiSubscription) {
		log.Info("no upgrade available")

		namespaceSegments := strings.Split(rhmiSubscription.Namespace, "-")
		namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
		if err := r.webbappNotifier.ClearNotification(namespacePrefix); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}
	log.Infof("Verifying the fields in the RHMI Subscription", l.Fields{"StartingCSV": rhmiSubscription.Spec.StartingCSV, "InstallPlanRef": rhmiSubscription.Status.InstallPlanRef})
	latestRHMIInstallPlan := &olmv1alpha1.InstallPlan{}
	err := wait.Poll(time.Second*5, time.Minute*5, func() (done bool, err error) {
		// gets the subscription with the recreated installplan
		err = r.Client.Get(ctx, k8sclient.ObjectKey{Name: rhmiSubscription.Name, Namespace: rhmiSubscription.Namespace}, rhmiSubscription)
		if err != nil {
			log.Infof("Couldn't retrieve the subscription due to an error", l.Fields{"Error": err})
			return false, nil
		}

		latestRHMIInstallPlan, err = rhmiConfigs.GetLatestInstallPlan(ctx, rhmiSubscription, r.Client)
		if err != nil {
			log.Infof("Install Plan was not created due to an error", l.Fields{"Error": err})
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		log.Infof("Triggering a new reconcile loop due to an error", l.Fields{"Error": err})

		return ctrl.Result{}, err
	}

	latestRHMICSV, err := r.csvLocator.GetCSV(ctx, r.Client, latestRHMIInstallPlan)
	if err != nil {
		return ctrl.Result{}, err
	}

	config := &integreatlyv1alpha1.RHMIConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-config",
			Namespace: r.operatorNamespace,
		},
	}
	err = r.Get(ctx, k8sclient.ObjectKey{Name: config.Name, Namespace: config.Namespace}, config)
	if err != nil {
		return ctrl.Result{}, err
	}

	// checks if the operator is running locally don't use the catalogsource
	csvFromCatalogSource := latestRHMICSV
	if resources.IsRunInCluster() {
		objectKey := k8sclient.ObjectKey{
			Name:      rhmiSubscription.Spec.CatalogSource,
			Namespace: rhmiSubscription.Spec.CatalogSourceNamespace,
		}
		csvFromCatalogSource, err = r.catalogSourceClient.GetLatestCSV(objectKey, rhmiSubscription.Spec.Package, rhmiSubscription.Spec.Channel)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("Error getting the csv from catalogsource %w", err)
		}

		if skipRangeStr, ok := csvFromCatalogSource.GetAnnotations()["olm.skipRange"]; ok {
			regex := regexp.MustCompile(`(?m)integreatly-operator\.v(.*)$`)
			rhmiPreviousVersion := regex.FindAllStringSubmatch(csvFromCatalogSource.Spec.Replaces, -1)[0][1]

			if rhmiPreviousVersion == "" {
				return ctrl.Result{}, fmt.Errorf("Error getting the version from replace field %w", err)
			}

			v, err := semver.Parse(rhmiPreviousVersion)
			expectedRange, err := semver.ParseRange(skipRangeStr)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("Error getting the version from the csv %w", err)
			}
			if expectedRange(v) && csvFromCatalogSource.Spec.Replaces != latestRHMICSV.Spec.Replaces {
				csvFromCatalogSource = latestRHMICSV
			}
		}
	}

	//default prefix
	csvNamePrefix := "integreatly-operator"
	if integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		csvNamePrefix = "managed-api-service"
	}

	currentOperatorVersionName := fmt.Sprintf("%s.v%s", csvNamePrefix, version.GetVersion())
	if csvFromCatalogSource.Spec.Replaces != currentOperatorVersionName { //if Catalog Source CSV does not replace currently installed operator
		err = rhmiConfigs.DeleteInstallPlan(ctx, latestRHMIInstallPlan, r.Client)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error deleting installplan %w", err)
		}

		log.Info("Installplan deleted for the install of patch upgrade")

		err = rhmiConfigs.CreateInstallPlan(ctx, rhmiSubscription, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		err := wait.Poll(time.Second*5, time.Minute*1, func() (done bool, err error) {
			// gets the subscription with the recreated installplan
			log.Info("Waiting for the new install plan to be created")
			err = r.Client.Get(ctx, k8sclient.ObjectKey{Name: rhmiSubscription.Name, Namespace: rhmiSubscription.Namespace}, rhmiSubscription)
			log.Info("got subscription")
			if err != nil {
				log.Infof("Couldn't get subscription", l.Fields{"Error": err})
				return false, nil
			}
			log.Infof("install plan read from the new subscription", l.Fields{"InstallPlanName": latestRHMIInstallPlan.Name})
			latestRHMIInstallPlan, err = rhmiConfigs.GetLatestInstallPlan(ctx, rhmiSubscription, r.Client)
			if err != nil {
				log.Infof("Install plan was not created", l.Fields{"Error": err})
				return false, nil
			}
			log.Infof("new install plan was created by OLM", l.Fields{"InstallPlanName": latestRHMIInstallPlan.Name})
			return true, nil
		})
		if err != nil {
			log.Infof("Triggering a new reconcile loop due to an error after waiting for the creation of a new install plan", l.Fields{"Error": err})

			return ctrl.Result{}, err
		}
	}
	isServiceAffecting := rhmiConfigs.IsUpgradeServiceAffecting(latestRHMICSV)

	if isServiceAffecting && !latestRHMIInstallPlan.Spec.Approved && config.Status.UpgradeAvailable == nil {
		newUpgradeAvailable := &integreatlyv1alpha1.UpgradeAvailable{
			TargetVersion: rhmiSubscription.Status.CurrentCSV,
			AvailableAt:   latestRHMIInstallPlan.CreationTimestamp,
		}

		config.Status.UpgradeAvailable = newUpgradeAvailable
		if err := r.Status().Update(context.TODO(), config); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	if isServiceAffecting {
		// Fixes cases where an upgrade is already in progress eg: patch upgrades
		// In that case we want to clear the toVersion so that the operator will reset it to the new upgrade
		// If it's not cleared, the installation controller logic can't handle it, which will block the upgrade from completing.
		if installation.Status.ToVersion != "" {
			installation.Status.ToVersion = ""
		}
		if err := r.Status().Update(context.TODO(), installation); err != nil {
			return ctrl.Result{}, err
		}
	}

	phase, err := r.webbappNotifier.NotifyUpgrade(config, latestRHMICSV.Spec.Version.String(), isServiceAffecting)
	if err != nil {
		return ctrl.Result{}, err
	}
	if phase == integreatlyv1alpha1.PhaseInProgress {
		log.Info("WebApp instance not found yet, skipping upgrade addition")
	}

	if !isServiceAffecting {
		eventRecorder := r.mgr.GetEventRecorderFor("RHMI Upgrade")
		if config.Status.UpgradeAvailable != nil && config.Status.UpgradeAvailable.TargetVersion == rhmiSubscription.Status.CurrentCSV {
			config.Status.UpgradeAvailable = nil
			if err := r.Status().Update(context.TODO(), config); err != nil {
				return ctrl.Result{}, err
			}
		}

		err = rhmiConfigs.ApproveUpgrade(ctx, r.Client, installation, latestRHMIInstallPlan, eventRecorder)
		logrus.Infof("Approving install plan %s ", latestRHMIInstallPlan.Name)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Requeue the reconciler until the RHMI subscription upgrade is complete
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Minute,
	}, nil
}

func (r *SubscriptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorsv1alpha1.Subscription{}).
		Complete(r)
}
