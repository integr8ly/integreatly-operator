package smtpcredentialset

import (
	"context"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	v1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_smtpcredentialset")

// Add creates a new SMTPCredentials Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	ctx := context.TODO()
	logger := logrus.WithFields(logrus.Fields{"controller": "controller_smtpcredentialset"})
	providerList := []providers.SMTPCredentialsProvider{aws.NewAWSSMTPCredentialProvider(client, logger)}
	cfgMgr := providers.NewConfigManager(providers.DefaultProviderConfigMapName, providers.DefaultConfigNamespace, client)
	return &ReconcileSMTPCredentialSet{client: mgr.GetClient(), scheme: mgr.GetScheme(), logger: logger, ctx: ctx, providerList: providerList, cfgMgr: cfgMgr}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("smtpcredentialset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SMTPCredentials
	err = c.Watch(&source.Kind{Type: &v1alpha1.SMTPCredentialSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSMTPCredentials implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSMTPCredentialSet{}

// ReconcileSMTPCredentials reconciles a SMTPCredentials object
type ReconcileSMTPCredentialSet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *runtime.Scheme
	logger       *logrus.Entry
	ctx          context.Context
	providerList []providers.SMTPCredentialsProvider
	cfgMgr       providers.ConfigManager
}

// Reconcile reads that state of the cluster for a SMTPCredentials object and makes changes based on the state read
// and what is in the SMTPCredentials.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSMTPCredentialSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("Reconciling SMTPCredentials")

	// Fetch the SMTPCredentials instance
	instance := &v1alpha1.SMTPCredentialSet{}
	err := r.client.Get(r.ctx, request.NamespacedName, instance)
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

	r.logger.Infof("checking for provider for deployment strategy %s", stratMap.SMTPCredentials)
	for _, p := range r.providerList {
		if !p.SupportsStrategy(stratMap.SMTPCredentials) {
			r.logger.Debugf("provider %s does not support deployment strategy %s, skipping", p.GetName(), stratMap.SMTPCredentials)
			continue
		}

		if instance.GetDeletionTimestamp() != nil {
			r.logger.Infof("running deletion handler on smtp credential instance %s", instance.Name)
			msg, err := p.DeleteSMTPCredentials(r.ctx, instance)
			if err != nil {
				if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, msg); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, errorUtil.Wrapf(err, "failed to run delete handler for smtp credentials instance %s", instance.Name)
			}

			r.logger.Infof("Waiting for SMTP credentials to successfully delete")
			if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseDeleteInProgress, msg); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 30}, nil
		}

		smtpCredentialSetInst, msg, err := p.CreateSMTPCredentials(r.ctx, instance)
		if err != nil {
			if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, msg); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to create smtp credential set for instance %s", instance.Name)
		}

		sec := &v1.Secret{
			ObjectMeta: controllerruntime.ObjectMeta{
				Name:      instance.Spec.SecretRef.Name,
				Namespace: instance.Namespace,
			},
		}
		_, err = controllerruntime.CreateOrUpdate(r.ctx, r.client, sec, func(existing runtime.Object) error {
			e := existing.(*v1.Secret)
			if err = controllerutil.SetControllerReference(instance, e, r.scheme); err != nil {
				if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, msg); err != nil {
					return err
				}
				return errorUtil.Wrapf(err, "failed to set owner on secret %s", sec.Name)
			}
			e.Data = smtpCredentialSetInst.DeploymentDetails.Data()
			e.Type = v1.SecretTypeOpaque
			return nil
		})
		if err != nil {
			if err = resources.UpdatePhase(r.ctx, r.client, instance, v1alpha1.PhaseFailed, "failed to reconcile instance secret"); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to reconcile blob storage instance secret %s", sec.Name)
		}

		instance.Status.Phase = v1alpha1.PhaseComplete
		instance.Status.Message = msg
		instance.Status.SecretRef = instance.Spec.SecretRef
		instance.Status.Strategy = stratMap.BlobStorage
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
	return reconcile.Result{Requeue: true}, nil
}
