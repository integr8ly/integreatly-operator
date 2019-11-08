package smtpcredentialset

import (
	"context"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

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
	logger := logrus.WithFields(logrus.Fields{"controller": "controller_smtpcredentialset"})
	providerList := []providers.SMTPCredentialsProvider{aws.NewAWSSMTPCredentialProvider(client, logger)}
	rp := resources.NewResourceProvider(client, mgr.GetScheme(), logger)
	return &ReconcileSMTPCredentialSet{
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
	client           client.Client
	scheme           *runtime.Scheme
	logger           *logrus.Entry
	resourceProvider *resources.ReconcileResourceProvider
	providerList     []providers.SMTPCredentialsProvider
}

// Reconcile reads that state of the cluster for a SMTPCredentials object and makes changes based on the state read
// and what is in the SMTPCredentials.Spec
func (r *ReconcileSMTPCredentialSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("Reconciling SMTPCredentials")
	ctx := context.TODO()
	cfgMgr := providers.NewConfigManager(providers.DefaultProviderConfigMapName, request.Namespace, r.client)

	// Fetch the SMTPCredentials instance
	instance := &v1alpha1.SMTPCredentialSet{}
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
		if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, v1alpha1.StatusDeploymentConfigNotFound.WrapError(err)); updateErr != nil {
			return reconcile.Result{}, updateErr
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
			msg, err := p.DeleteSMTPCredentials(ctx, instance)
			if err != nil {
				if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, msg.WrapError(err)); updateErr != nil {
					return reconcile.Result{}, updateErr
				}
				return reconcile.Result{}, errorUtil.Wrapf(err, "failed to run delete handler for smtp credentials instance %s", instance.Name)
			}

			r.logger.Infof("Waiting for SMTP credentials to successfully delete")
			if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseDeleteInProgress, msg); updateErr != nil {
				return reconcile.Result{}, updateErr
			}
			return reconcile.Result{Requeue: true, RequeueAfter: p.GetReconcileTime(instance)}, nil
		}

		smtpCredentialSetInst, msg, err := p.CreateSMTPCredentials(ctx, instance)
		if err != nil {
			if updateErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, msg.WrapError(err)); updateErr != nil {
				return reconcile.Result{}, updateErr
			}
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to create smtp credential set for instance %s", instance.Name)
		}

		if err := r.resourceProvider.ReconcileResultSecret(ctx, instance, smtpCredentialSetInst.DeploymentDetails.Data()); err != nil {
			return reconcile.Result{}, errorUtil.Wrap(err, "failed to reconcile secret")
		}
		instance.Status.Phase = v1alpha1.PhaseComplete
		instance.Status.Message = msg
		instance.Status.SecretRef = instance.Spec.SecretRef
		instance.Status.Strategy = stratMap.BlobStorage
		instance.Status.Provider = p.GetName()
		if err = r.client.Status().Update(ctx, instance); err != nil {
			return reconcile.Result{}, errorUtil.Wrapf(err, "failed to update instance %s in namespace %s", instance.Name, instance.Namespace)
		}
		return reconcile.Result{Requeue: true, RequeueAfter: p.GetReconcileTime(instance)}, nil
	}

	// unsupported strategy
	if updatePhaseErr := resources.UpdatePhase(ctx, r.client, instance, v1alpha1.PhaseFailed, v1alpha1.StatusUnsupportedType.WrapError(err)); updatePhaseErr != nil {
		return reconcile.Result{}, updatePhaseErr
	}
	return reconcile.Result{Requeue: true}, nil
}
