package subscription

import (
	"context"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/controller/subscription/rhmiConfigs"
	"github.com/integr8ly/integreatly-operator/pkg/controller/subscription/webapp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"github.com/sirupsen/logrus"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

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
)

// Add creates a new Subscription Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, _ []string) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	ctx, cancel := context.WithCancel(context.Background())
	operatorNs := "redhat-rhmi-operator"

	restConfig := controllerruntime.GetConfigOrDie()

	return &ReconcileSubscription{
		mgr:               mgr,
		client:            mgr.GetClient(),
		context:           ctx,
		cancel:            cancel,
		restConfig:        restConfig,
		scheme:            mgr.GetScheme(),
		operatorNamespace: operatorNs,
	}
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
	return nil
}

// blank assignment to verify that ReconcileSubscription implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSubscription{}

type ReconcileSubscription struct {
	client            k8sclient.Client
	restConfig        *rest.Config
	context           context.Context
	cancel            context.CancelFunc
	scheme            *runtime.Scheme
	operatorNamespace string
	mgr               manager.Manager
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
	err := r.client.Get(r.context, request.NamespacedName, subscription)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if subscription.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalManual {
		subscription.Spec.InstallPlanApproval = operatorsv1alpha1.ApprovalManual
		err = r.client.Update(r.context, subscription)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// TODO: investigate a better approach to getting RHMI rather than hardcoding values
	installation := &integreatlyv1alpha1.RHMI{}
	err = r.client.Get(r.context, k8sclient.ObjectKey{Name: "rhmi", Namespace: request.NamespacedName.Namespace}, installation)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	return r.HandleUpgrades(subscription, installation)
}

func (r *ReconcileSubscription) HandleUpgrades(rhmiSubscription *operatorsv1alpha1.Subscription, installation *integreatlyv1alpha1.RHMI) (reconcile.Result, error) {
	if !rhmiConfigs.IsUpgradeAvailable(rhmiSubscription) {
		logrus.Infof("no upgrade available")
		return reconcile.Result{}, nil
	}

	logrus.Infof("RHMI upgrade available")

	latestRHMIInstallPlan, err := rhmiConfigs.GetLatestInstallPlan(r.context, rhmiSubscription, r.client)
	if err != nil {
		return reconcile.Result{}, err
	}

	latestRHMICSV, err := rhmiConfigs.GetCSV(latestRHMIInstallPlan)
	if err != nil {
		return reconcile.Result{}, err
	}

	config := &integreatlyv1alpha1.RHMIConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-config",
			Namespace: r.operatorNamespace,
		},
	}
	err = r.client.Get(r.context, k8sclient.ObjectKey{Name: config.Name, Namespace: config.Namespace}, config)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = rhmiConfigs.UpdateStatus(r.context, r.client, config, latestRHMIInstallPlan)
	if err != nil {
		return reconcile.Result{}, err
	}

	canUpgradeNow, err := rhmiConfigs.CanUpgradeNow(config, installation)
	if err != nil {
		return reconcile.Result{}, err
	}

	isServiceAffecting := rhmiConfigs.IsUpgradeServiceAffecting(latestRHMICSV)

	webbappNotifier, err := webapp.NewUpgradeNotifier(r.context, r.restConfig)
	if err != nil {
		return reconcile.Result{}, err
	}

	phase, err := webbappNotifier.NotifyUpgrade(config, latestRHMICSV.Spec.Version.String(), isServiceAffecting)
	if err != nil {
		return reconcile.Result{}, err
	}
	if phase == integreatlyv1alpha1.PhaseInProgress {
		logrus.Infof("WebApp instance not found yet, skipping upgrade addition")
	}

	if !isServiceAffecting || canUpgradeNow {
		eventRecorder := r.mgr.GetEventRecorderFor("RHMI Upgrade")

		if err := webbappNotifier.ClearNotification(); err != nil {
			return reconcile.Result{}, err
		}

		err = rhmiConfigs.ApproveUpgrade(r.context, r.client, config, installation, latestRHMIInstallPlan, eventRecorder)
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
