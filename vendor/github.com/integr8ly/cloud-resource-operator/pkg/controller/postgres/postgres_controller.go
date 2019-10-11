package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers/openshift"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	errorUtil "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_postgres")

// Add creates a new Postgres Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	ctx := context.TODO()
	logger := logrus.WithFields(logrus.Fields{"controller": "controller_postgres"})
	providerList := []providers.PostgresProvider{openshift.NewOpenShiftPostgresProvider(client, logger), aws.NewAWSPostgresProvider(client, logger)}
	cfgMgr := providers.NewConfigManager(providers.DefaultProviderConfigMapName, providers.DefaultConfigNamespace, client)
	return &ReconcilePostgres{client: client, scheme: mgr.GetScheme(), logger: logger, ctx: ctx, providerList: providerList, cfgMgr: cfgMgr}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("postgres-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Postgres
	err = c.Watch(&source.Kind{Type: &v1alpha1.Postgres{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Postgres
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.Postgres{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePostgres implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePostgres{}

// ReconcilePostgres reconciles a Postgres object
type ReconcilePostgres struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *runtime.Scheme
	logger       *logrus.Entry
	ctx          context.Context
	providerList []providers.PostgresProvider
	cfgMgr       providers.ConfigManager
}

// Reconcile reads that state of the cluster for a Postgres object and makes changes based on the state read
// and what is in the Postgres.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePostgres) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("Reconciling Postgres")

	// Fetch the Postgres instance
	instance := &v1alpha1.Postgres{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	stratMap, err := r.cfgMgr.GetStrategyMappingForDeploymentType(r.ctx, instance.Spec.Type)
	if err != nil {
		if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, "failed to read deployment type config for deployment"); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, errorUtil.Wrapf(err, "failed to read deployment type config for deployment %s", instance.Spec.Type)
	}

	for _, p := range r.providerList {
		if !p.SupportsStrategy(stratMap.Postgres) {
			continue
		}

		// delete the postgres if the deletion timestamp exists
		if instance.DeletionTimestamp != nil {
			msg, err := p.DeletePostgres(r.ctx, instance)
			if err != nil {
				if errUpdate := resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, msg); errUpdate != nil {
					return reconcile.Result{}, errUpdate
				}
				return reconcile.Result{}, errorUtil.Wrapf(err, "failed to perform provider-specific storage deletion")
			}

			r.logger.Info("Waiting on Postgres to successfully delete")
			if errUpdate := resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseDeleteInProgress, msg); errUpdate != nil {
				return reconcile.Result{}, errUpdate
			}
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 30}, nil
		}

		// create the postgres instance
		ps, msg, err := p.CreatePostgres(r.ctx, instance)
		if err != nil {
			instance.Status.SecretRef = &v1alpha1.SecretRef{}
			if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, msg); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		if ps == nil {
			r.logger.Info("Secret data is still reconciling, postgres instance is nil")
			instance.Status.SecretRef = &v1alpha1.SecretRef{}
			if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseInProgress, msg); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 30}, nil
		}

		// return the connection secret
		sec := &corev1.Secret{
			ObjectMeta: controllerruntime.ObjectMeta{
				Name:      instance.Spec.SecretRef.Name,
				Namespace: instance.Namespace,
			},
		}
		_, err = controllerruntime.CreateOrUpdate(r.ctx, r.client, sec, func(existing runtime.Object) error {
			e := existing.(*corev1.Secret)
			if err = controllerutil.SetControllerReference(instance, e, r.scheme); err != nil {
				if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, "failed to set owner on secret"); err != nil {
					return err
				}
				return errorUtil.Wrapf(err, "failed to set owner on secret %s", sec.Name)
			}
			e.Data = ps.DeploymentDetails.Data()
			e.Type = corev1.SecretTypeOpaque
			return nil
		})
		if err != nil {
			if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, "failed to reconcile instance secret"); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to reconcile postgres instance secret %s", sec.Name)
		}

		instance.Status.Phase = v1alpha1.PhaseComplete
		instance.Status.Message = msg
		instance.Status.SecretRef = instance.Spec.SecretRef
		instance.Status.Strategy = stratMap.Postgres
		instance.Status.Provider = p.GetName()
		if err = r.client.Status().Update(r.ctx, instance); err != nil {
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to update instance %s in namespace %s", instance.Name, instance.Namespace)
		}
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 30}, nil
	}

	// unsupported strategy
	if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, "unsupported deployment strategy"); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, errorUtil.New(fmt.Sprintf("unsupported deployment strategy %s", stratMap.Postgres))
}
