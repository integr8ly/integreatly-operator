/*
Copyright 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rhmiconfig

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"time"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/rhmiconfig/helpers"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "rhmi_config_controller"})

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RHMIConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	ctx, cancel := context.WithCancel(context.Background())
	return &ReconcileRHMIConfig{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		context: ctx,
		cancel:  cancel,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rhmiconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RHMIConfig
	err = c.Watch(&source.Kind{Type: &integreatlyv1alpha1.RHMIConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRHMIConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRHMIConfig{}

// ReconcileRHMIConfig reconciles a RHMIConfig object
type ReconcileRHMIConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	context context.Context
	cancel  context.CancelFunc
}

// Reconcile reads that state of the cluster for a RHMIConfig object and makes changes based on the state read
// and what is in the RHMIConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRHMIConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling RHMIConfig")

	// Fetch the RHMIConfig instance
	rhmiConfig := &integreatlyv1alpha1.RHMIConfig{}
	err := r.client.Get(r.context, request.NamespacedName, rhmiConfig)
	if err != nil {
		if k8sErr.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//	Checks if there is an upgrade available, if there is upgrade available upgrades the status of RHMIConfig
	if rhmiConfig.Status.UpgradeAvailable != nil {
		err = helpers.UpdateStatus(context.TODO(), r.client, rhmiConfig)
		if err != nil {
			return reconcile.Result{}, err
		}

		err = r.client.Update(context.TODO(), rhmiConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		rhmiConfig.Status.Upgrade = integreatlyv1alpha1.RHMIConfigStatusUpgrade{}
		err = r.client.Status().Update(context.TODO(), rhmiConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	retryRequeue := reconcile.Result{
		Requeue:      true,
		RequeueAfter: 10 * time.Second,
	}

	// ensure values are as expected
	if err := r.reconcileValues(rhmiConfig, reconcileBackupAndMaintenanceValues, reconcileUpgradeValues); err != nil {
		log.Error("failed to reconcile rhmi config values", err)
		return retryRequeue, err
	}

	// create cloud resource operator override config map
	if err := r.ReconcileCloudResourceStrategies(rhmiConfig); err != nil {
		log.Error("rhmi config failure while reconciling cloud resource strategies", err)
		return retryRequeue, err
	}

	log.Info("rhmi config reconciled successfully")
	return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, nil
}

// reconciles cloud resource strategies, setting backup and maintenance values for postgres and redis instances
func (r *ReconcileRHMIConfig) ReconcileCloudResourceStrategies(config *integreatlyv1alpha1.RHMIConfig) error {
	log.Info("reconciling cloud resource maintenance strategies")

	// validate the applyOn and applyFrom values are correct
	// we also perform this validation on the validate web-hook
	// we should also provide validation in the controller prior to provisioning/updating the cloud resources (Postgres and Redis)
	// this is to avoid any chance of a badly formatted value making it to CRO
	// cloud providers are consistently good at obscure error messages
	backupApplyOn, maintenanceApplyFrom, err := integreatlyv1alpha1.ValidateBackupAndMaintenance(config.Spec.Backup.ApplyOn, config.Spec.Maintenance.ApplyFrom)
	if err != nil {
		return fmt.Errorf("failure validating backup and maintenance values : %v", err)
	}

	// build time config expected by CRO
	timeConfig := &croUtil.StrategyTimeConfig{
		BackupStartTime:      backupApplyOn,
		MaintenanceStartTime: maintenanceApplyFrom,
	}

	// reconcile cro strategy config map, RHMI operator does not care what infrastructure the cluster is running in
	// as we support different cloud providers this CRO Reconcile Function will ensure the correct infrastructure strategies are provisioned
	if err := croUtil.ReconcileStrategyMaps(r.context, r.client, timeConfig, croUtil.TierProduction, config.Namespace); err != nil {
		return fmt.Errorf("failure to reconcile aws strategy map : %v", err)
	}

	return nil
}

// we require that blank applyOn and applyFrom values be set to defaults
// we expect a user to set their own times, but in the case where times are not set
// we set our maintenance applyFrom values to be Thu 02:00
// we set out backup applyOn values to be 03:01
func reconcileBackupAndMaintenanceValues(rhmiConfig *integreatlyv1alpha1.RHMIConfig) error {
	if rhmiConfig.Spec.Maintenance.ApplyFrom == "" {
		rhmiConfig.Spec.Maintenance.ApplyFrom = integreatlyv1alpha1.DefaultMaintenanceApplyFrom
	}
	if rhmiConfig.Spec.Backup.ApplyOn == "" {
		rhmiConfig.Spec.Backup.ApplyOn = integreatlyv1alpha1.DefaultBackupApplyOn
	}
	return nil
}

func reconcileUpgradeValues(rhmiConfig *integreatlyv1alpha1.RHMIConfig) error {
	rhmiConfig.Spec.Upgrade.DefaultIfEmpty()
	return nil
}

func (r *ReconcileRHMIConfig) reconcileValues(rhmiConfig *integreatlyv1alpha1.RHMIConfig, mutateFns ...func(*integreatlyv1alpha1.RHMIConfig) error) error {
	if _, err := controllerutil.CreateOrUpdate(r.context, r.client, rhmiConfig, func() error {
		for _, mutateFn := range mutateFns {
			if err := mutateFn(rhmiConfig); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create or update rhmiConfig : %v", err)
	}

	return nil
}
