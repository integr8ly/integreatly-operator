package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
	"github.com/integr8ly/integreatly-operator/version"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/controllers/subscription/csvlocator"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/rhmiConfigs"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	catalogsourceClient "github.com/integr8ly/integreatly-operator/pkg/resources/catalogsource"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

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
	ManagedAPIAddonSubscription     = "addon-managed-api-service"
	ManagedAPIAddonSubscriptionEdge = "addon-managed-api-service-internal"
	ManagedAPIolmSubscription       = "managed-api-service"
)

var subscriptionsToReconcile []string = []string{
	IntegreatlyPackage,
	ManagedAPIAddonSubscription,
	ManagedAPIAddonSubscriptionEdge,
	ManagedAPIolmSubscription,
}

func New(mgr manager.Manager) (*SubscriptionReconciler, error) {
	watchNS, err := k8s.GetWatchNamespace()
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
		csvLocator:          csvLocator,
	}, nil
}

type SubscriptionReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme

	operatorNamespace   string
	mgr                 manager.Manager
	catalogSourceClient catalogsourceClient.CatalogSourceClientInterface
	csvLocator          csvlocator.CSVLocator
}

// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions;subscriptions/status,verbs=get;list;watch;update;patch;delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups=operators.coreos.com,resources=installplans,verbs=get;list;watch;update;patch;delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;delete;list,namespace=integreatly-operator

// Reconcile will ensure that that Subscription object(s) have Manual approval for the upgrades
// In a namespaced installation of integreatly operator it will only reconcile Subscription of the integreatly operator itself
func (r *SubscriptionReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// skip any Subscriptions that are not integreatly operator
	if !r.shouldReconcileSubscription(request) {
		return ctrl.Result{}, nil
	}

	// Remove code below post 1.38.0 delivery to production
	// Temporary code to update backend redis 100mln to specific value
	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhoam",
			Namespace: "redhat-rhoam-operator",
		},
	}
	err := r.Get(context.TODO(), k8sclient.ObjectKey{Name: installation.Name, Namespace: installation.Namespace}, installation)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}
	// Only apply below logic for existing installations, we can't rely on toVersion not present due to possible race condition
	// between sub and rhmi controller
	if installation.Status.Version != "" {
		// Only apply below logic for 100 mln quota installations
		if installation.Status.Quota == "100 Million" {
			// at this point, it's safe to assume the backend redis exists
			backendRedis := &v1alpha1.Redis{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "threescale-backend-redis-rhoam",
					Namespace: "redhat-rhoam-operator",
				},
			}

			err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: backendRedis.Name, Namespace: backendRedis.Namespace}, backendRedis)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Only apply logic if backend redis node size is not cache.m5.xlarge
			if backendRedis.Spec.Size != "cache.m5.xlarge" {
				backendRedis.Spec.Size = "cache.m5.xlarge"

				err = r.Client.Update(ctx, backendRedis)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}
	// Remove code above post 1.38.0 delivery to production

	log.Infof("Reconciling subscription", l.Fields{"request": request, "opNS": r.operatorNamespace})
	subscription := &operatorsv1alpha1.Subscription{}
	err = r.Get(context.TODO(), request.NamespacedName, subscription)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if subscription.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalManual {
		subscription.Spec.InstallPlanApproval = operatorsv1alpha1.ApprovalManual

		// We need to get the latest InstallPlan to get the CSV in order to set the subscription's Config.Resources
		// The Config.Resources field needs to be explicitly set because otherwise r.Update will silently set the Config to {} if it's not already set
		latestInstallPlan := &operatorsv1alpha1.InstallPlan{}
		err := wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*3, false, func(ctx context.Context) (done bool, err error) {
			if subscription.Status.InstallPlanRef == nil {
				log.Info("InstallPlanRef from Subscription is nil, trying again...")
				return false, nil
			}
			err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: subscription.Status.InstallPlanRef.Name, Namespace: subscription.Status.InstallPlanRef.Namespace}, latestInstallPlan)
			if err != nil {
				log.Infof("Failed to get InstallPlan, trying again...", l.Fields{"Error": err})
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			log.Infof("Failed to get the latest InstallPlan", l.Fields{"Error": err})
			return ctrl.Result{}, err
		}
		csv, err := r.csvLocator.GetCSV(context.TODO(), r.Client, latestInstallPlan)
		if err != nil {
			log.Infof("Failed to get CSV from the latest InstallPlan", l.Fields{"Error": err})
			return ctrl.Result{}, err
		}
		deploymentSpecs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
		if len(deploymentSpecs) > 0 {
			containers := deploymentSpecs[0].Spec.Template.Spec.Containers
			if len(containers) > 0 {
				if subscription.Spec.Config == nil {
					subscription.Spec.Config = &operatorsv1alpha1.SubscriptionConfig{}
				}
				subscription.Spec.Config.Resources = &containers[0].Resources
			}
		}

		err = r.Update(context.TODO(), subscription)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	rhmiCr, err := rhmi.GetRhmiCr(r.Client, context.TODO(), request.NamespacedName.Namespace, log)
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

	for _, reconcilable := range subscriptionsToReconcile {
		if request.Name == reconcilable {
			return true
		}
	}

	return false
}

func (r *SubscriptionReconciler) HandleUpgrades(ctx context.Context, rhmiSubscription *operatorsv1alpha1.Subscription, installation *integreatlyv1alpha1.RHMI) (ctrl.Result, error) {
	if rhmiSubscription == nil || installation == nil {
		return ctrl.Result{}, nil
	}

	log.Infof("Verifying the fields in the Subscription", l.Fields{"StartingCSV": rhmiSubscription.Spec.StartingCSV, "InstallPlanRef": rhmiSubscription.Status.InstallPlanRef})
	latestInstallPlan := &operatorsv1alpha1.InstallPlan{}
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*5, false, func(ctx2 context.Context) (done bool, err error) {
		// gets the subscription with the recreated installplan
		err = r.Client.Get(ctx, k8sclient.ObjectKey{Name: rhmiSubscription.Name, Namespace: rhmiSubscription.Namespace}, rhmiSubscription)
		if err != nil {
			log.Infof("Couldn't retrieve the subscription due to an error", l.Fields{"Error": err})
			return false, nil
		}

		latestInstallPlan, err = rhmiConfigs.GetLatestInstallPlan(ctx, rhmiSubscription, r.Client)
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

	latestCSV, err := r.csvLocator.GetCSV(ctx, r.Client, latestInstallPlan)
	if err != nil {
		return ctrl.Result{}, err
	}

	isServiceAffecting := rhmiConfigs.IsUpgradeServiceAffecting(latestCSV)
	log.Info(fmt.Sprintf("Upgrade is service affecting: %v", isServiceAffecting))

	err = r.allowDatabaseUpdates(ctx, installation, isServiceAffecting)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("to version is currently: %s", installation.Status.ToVersion))
	if !rhmiConfigs.IsUpgradeAvailable(rhmiSubscription) {
		// if we have not started update but are due to, and it was a serviceAffecting upgrade
		// requeue so that we can enable maintenance window
		if (installation.Status.ToVersion != "" || installation.Status.Version != version.GetVersion()) && isServiceAffecting {
			log.Info("upgrade still in progress, requeue-ing")
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 10 * time.Second,
			}, nil
		}
		log.Info("no upgrade available")
		return ctrl.Result{}, nil
	}

	if !isServiceAffecting && !latestInstallPlan.Spec.Approved {
		eventRecorder := r.mgr.GetEventRecorderFor("Operator Upgrade")
		logrus.Infof("Approving install plan %s ", latestInstallPlan.Name)
		err = rhmiConfigs.ApproveUpgrade(ctx, r.Client, installation, latestInstallPlan, eventRecorder)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Requeue the reconciler until the operator subscription upgrade is complete
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

func (r *SubscriptionReconciler) allowDatabaseUpdates(ctx context.Context, installation *integreatlyv1alpha1.RHMI, isServiceAffecting bool) error {
	if installation.Status.Version != "" && installation.Status.ToVersion != "" && isServiceAffecting {
		log.Info("Service affecting and upgrading, setting maintenanceWindow to true")
		postgresInstances := &crov1alpha1.PostgresList{}
		if err := r.Client.List(ctx, postgresInstances); err != nil {
			return fmt.Errorf("failed to list postgres instances: %w", err)
		}
		for _, pgInst := range postgresInstances.Items {
			inst := pgInst
			inst.Spec.MaintenanceWindow = true
			if err := r.Client.Update(ctx, &inst); err != nil {
				return pkgerr.Wrap(err, fmt.Sprintf("failed to update maintenance window for postgres %s", inst.Name))
			}
		}

		redisInstances := &crov1alpha1.RedisList{}
		if err := r.Client.List(ctx, redisInstances); err != nil {
			return fmt.Errorf("failed to list redis instances: %w", err)
		}
		for _, rdInst := range redisInstances.Items {
			inst := rdInst
			inst.Spec.MaintenanceWindow = true
			if err := r.Client.Update(ctx, &inst); err != nil {
				return pkgerr.Wrap(err, fmt.Sprintf("failed to update maintenance window for postgres %s", inst.Name))
			}
		}
	}

	return nil
}
