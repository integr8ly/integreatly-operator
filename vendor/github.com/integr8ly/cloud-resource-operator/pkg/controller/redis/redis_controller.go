package redis

import (
	"context"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/openshift"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

var log = logf.Log.WithName("controller_redis")

// Add creates a new Redis Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	logger := logrus.WithFields(logrus.Fields{"controller": "controller_redis"})
	providerList := []providers.RedisProvider{aws.NewAWSRedisProvider(client, logger), openshift.NewOpenShiftRedisProvider(client, logger)}
	rp := resources.NewResourceProvider(client, mgr.GetScheme(), logger)
	return &ReconcileRedis{
		client:           mgr.GetClient(),
		scheme:           mgr.GetScheme(),
		logger:           logger,
		resourceProvider: rp,
		providerList:     providerList,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("redis-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Redis
	err = c.Watch(&source.Kind{Type: &integreatlyv1alpha1.Redis{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Redis
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &integreatlyv1alpha1.Redis{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRedis implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRedis{}

// ReconcileRedis reconciles a Redis object
type ReconcileRedis struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client           client.Client
	scheme           *runtime.Scheme
	logger           *logrus.Entry
	resourceProvider *resources.ReconcileResourceProvider
	providerList     []providers.RedisProvider
}

// Reconcile reads that state of the cluster for a Redis object and makes changes based on the state read
// and what is in the Redis.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRedis) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("Reconciling Redis")
	ctx := context.TODO()
	cfgMgr := providers.NewConfigManager(providers.DefaultProviderConfigMapName, request.Namespace, r.client)

	// Fetch the Redis instance
	instance := &integreatlyv1alpha1.Redis{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
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

	stratMap, err := cfgMgr.GetStrategyMappingForDeploymentType(ctx, instance.Spec.Type)
	if err != nil {
		if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, integreatlyv1alpha1.StatusDeploymentConfigNotFound.WrapError(err)); updateErr != nil {
			return reconcile.Result{}, updateErr
		}
		return reconcile.Result{}, errorUtil.Wrapf(err, "failed to read deployment type config for deployment %s", instance.Spec.Type)
	}

	for _, p := range r.providerList {
		if !p.SupportsStrategy(stratMap.Redis) {
			r.logger.Debugf("provider %s does not support deployment strategy %s, skipping", p.GetName(), stratMap.Redis)
			continue
		}

		// handle deletion of redis and remove any finalizers added
		if instance.GetDeletionTimestamp() != nil {
			msg, err := p.DeleteRedis(ctx, instance)
			if err != nil {
				if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, msg.WrapError(err)); updateErr != nil {
					return reconcile.Result{}, updateErr
				}
				return reconcile.Result{}, errorUtil.Wrapf(err, "failed to perform provider specific cluster deletion")
			}

			r.logger.Info("Waiting for redis cluster to successfully delete")
			if err = resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseDeleteInProgress, msg); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: p.GetReconcileTime(instance)}, nil
		}

		// handle creation of redis and apply any finalizers to instance required for deletion
		redis, msg, err := p.CreateRedis(ctx, instance)
		if err != nil {
			instance.Status.SecretRef = &v1alpha1.SecretRef{}
			if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, msg.WrapError(err)); updateErr != nil {
				return reconcile.Result{}, updateErr
			}
			return reconcile.Result{}, err
		}
		if redis == nil {
			instance.Status.SecretRef = &v1alpha1.SecretRef{}
			r.logger.Info("Waiting for redis cluster to become available")
			if err = resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseInProgress, msg); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: p.GetReconcileTime(instance)}, nil
		}

		// create the secret with the redis cluster connection details
		if err := r.resourceProvider.ReconcileResultSecret(ctx, instance, redis.DeploymentDetails.Data()); err != nil {
			return reconcile.Result{}, errorUtil.Wrap(err, "failed to reconcile secret")
		}

		// update the redis custom resource
		instance.Status.Phase = v1alpha1.PhaseComplete
		instance.Status.Message = msg
		instance.Status.SecretRef = instance.Spec.SecretRef
		instance.Status.Strategy = stratMap.Redis
		instance.Status.Provider = p.GetName()
		if err = r.client.Status().Update(ctx, instance); err != nil {
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to update instance %s in namespace %s", instance.Name, instance.Namespace)
		}
		return reconcile.Result{Requeue: true, RequeueAfter: p.GetReconcileTime(instance)}, nil
	}

	// unsupported strategy
	if err = resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseInProgress, integreatlyv1alpha1.StatusUnsupportedType.WrapError(err)); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, errorUtil.New(fmt.Sprintf("unsupported deployment strategy %s", stratMap.Redis))
}
