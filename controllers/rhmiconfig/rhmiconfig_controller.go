/*


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

package controllers

import (
	"context"
	"fmt"
	"time"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/controllers/rhmiconfig/helpers"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rhmiconfigv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "rhmi_config_controller"})

// RHMIConfigReconciler reconciles a RHMIConfig object
type RHMIConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=integreatly.org,resources=rhmiconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=integreatly.org,resources=rhmiconfigs/status,verbs=get;update;patch

func (r *RHMIConfigReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	log.Info("reconciling RHMIConfig")
	// Fetch the RHMIConfig instance
	rhmiConfig := &rhmiconfigv1alpha1.RHMIConfig{}
	err := r.Get(context.TODO(), request.NamespacedName, rhmiConfig)
	if err != nil {
		if k8sErr.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	//	Checks if there is an upgrade available, if there is upgrade available upgrades the status of RHMIConfig
	if rhmiConfig.Status.UpgradeAvailable != nil {
		err = helpers.UpdateStatus(context.TODO(), r.Client, rhmiConfig)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Update(context.TODO(), rhmiConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		rhmiConfig.Status.Upgrade = rhmiconfigv1alpha1.RHMIConfigStatusUpgrade{}
		err = r.Status().Update(context.TODO(), rhmiConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	retryRequeue := ctrl.Result{
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
	return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, nil
}

// reconciles cloud resource strategies, setting backup and maintenance values for postgres and redis instances
func (r *RHMIConfigReconciler) ReconcileCloudResourceStrategies(config *rhmiconfigv1alpha1.RHMIConfig) error {
	log.Info("reconciling cloud resource maintenance strategies")

	// validate the applyOn and applyFrom values are correct
	// we also perform this validation on the validate web-hook
	// we should also provide validation in the controller prior to provisioning/updating the cloud resources (Postgres and Redis)
	// this is to avoid any chance of a badly formatted value making it to CRO
	// cloud providers are consistently good at obscure error messages
	backupApplyOn, maintenanceApplyFrom, err := rhmiconfigv1alpha1.ValidateBackupAndMaintenance(config.Spec.Backup.ApplyOn, config.Spec.Maintenance.ApplyFrom)
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
	if err := croUtil.ReconcileStrategyMaps(context.TODO(), r.Client, timeConfig, croUtil.TierProduction, config.Namespace); err != nil {
		return fmt.Errorf("failure to reconcile aws strategy map : %v", err)
	}

	return nil
}

// we require that blank applyOn and applyFrom values be set to defaults
// we expect a user to set their own times, but in the case where times are not set
// we set our maintenance applyFrom values to be Thu 02:00
// we set out backup applyOn values to be 03:01
func reconcileBackupAndMaintenanceValues(rhmiConfig *rhmiconfigv1alpha1.RHMIConfig) error {
	if rhmiConfig.Spec.Maintenance.ApplyFrom == "" {
		rhmiConfig.Spec.Maintenance.ApplyFrom = rhmiconfigv1alpha1.DefaultMaintenanceApplyFrom
	}
	if rhmiConfig.Spec.Backup.ApplyOn == "" {
		rhmiConfig.Spec.Backup.ApplyOn = rhmiconfigv1alpha1.DefaultBackupApplyOn
	}
	return nil
}

func reconcileUpgradeValues(rhmiConfig *rhmiconfigv1alpha1.RHMIConfig) error {
	rhmiConfig.Spec.Upgrade.DefaultIfEmpty()
	return nil
}

func (r *RHMIConfigReconciler) reconcileValues(rhmiConfig *rhmiconfigv1alpha1.RHMIConfig, mutateFns ...func(*rhmiconfigv1alpha1.RHMIConfig) error) error {
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, rhmiConfig, func() error {
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

func (r *RHMIConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rhmiconfigv1alpha1.RHMIConfig{}).
		Complete(r)
}
