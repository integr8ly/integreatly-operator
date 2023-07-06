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

package status

import (
	"context"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
	"github.com/integr8ly/integreatly-operator/utils"
	addonv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// StatusReconciler reconciles a AddonInstance object
type StatusReconciler struct {
	client.Client
	Log                 l.Logger
	Scheme              *runtime.Scheme
	addonInstanceClient *addoninstance.AddonInstanceClientImpl
	cfg                 ControllerOptions
}

func New(mgr manager.Manager, opts ...ControllerConfig) (*StatusReconciler, error) {
	var cfg ControllerOptions

	cfg.Option(opts...)
	cfg.Default()

	return &StatusReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		Log:                 l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "status_controller"}),
		addonInstanceClient: addoninstance.NewAddonInstanceClient(mgr.GetClient()),
		cfg:                 cfg,
	}, nil
}

//+kubebuilder:rbac:groups=addons.managed.openshift.io,resources=addoninstances,verbs=get;list;patch;watch;
//+kubebuilder:rbac:groups=addons.managed.openshift.io,resources=addoninstances/status,verbs=get;update;patch

func (r *StatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	addonInstance, err := r.getAddonInstance(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	if addonInstance.Spec.HeartbeatUpdatePeriod.Duration != r.cfg.HeartBeatInterval {
		addonInstance.Spec.HeartbeatUpdatePeriod.Duration = r.cfg.HeartBeatInterval
		if err := r.Client.Patch(ctx, addonInstance, client.Merge); err != nil {
			return ctrl.Result{}, err
		}
	}

	installation, err := rhmi.GetRhmiCr(r.Client, ctx, req.Namespace, r.Log)
	if err != nil {
		return ctrl.Result{}, err
	}

	conditions := r.buildAddonInstanceConditions(installation)

	if err := r.updateAddonInstanceWithConditions(ctx, addonInstance, conditions); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.cfg.HeartBeatInterval}, nil
}

func (r *StatusReconciler) getAddonInstance(ctx context.Context, req ctrl.Request) (*addonv1alpha1.AddonInstance, error) {
	// addon instance key
	objectKey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      r.cfg.AddonInstanceName,
	}

	// Obtain current addon instance
	addonInstance := &addonv1alpha1.AddonInstance{}
	if err := r.Client.Get(ctx, objectKey, addonInstance); err != nil {
		return nil, err
	}

	return addonInstance, nil
}

// BuildAddonInstanceConditions returns the conditions to update the addon instance with
func (r *StatusReconciler) buildAddonInstanceConditions(installation *v1alpha1.RHMI) []metav1.Condition {
	var conditions []metav1.Condition

	// Addon uninstall complete
	if installation == nil {
		conditions = append(conditions, installation.UninstalledCondition())
		r.Log.Info("Addon Successfully uninstalled")
		return conditions
	}

	conditions = append(conditions, r.appendInstalledConditions(installation)...)
	conditions = append(conditions, r.appendHealthConditions(installation)...)
	conditions = append(conditions, r.appendDegradedConditions(installation)...)

	return conditions
}

func (r *StatusReconciler) appendInstalledConditions(installation *v1alpha1.RHMI) []metav1.Condition {
	var conditions []metav1.Condition

	if installation.IsInstalled() {
		conditions = append(conditions, installation.InstalledCondition())
	}

	if installation.IsInstallBlocked() {
		conditions = append(conditions, installation.InstallBlockedCondition())
	}

	if installation.IsUninstallBlocked() {
		conditions = append(conditions, installation.UninstallBlockedCondition())
	}

	return conditions
}

func (r *StatusReconciler) appendHealthConditions(installation *v1alpha1.RHMI) []metav1.Condition {
	var conditions []metav1.Condition

	if installation.IsCoreComponentsHealthy() {
		conditions = append(conditions, installation.HealthyCondition())
	} else {
		conditions = append(conditions, installation.UnHealthyCondition())
	}

	return conditions
}

func (r *StatusReconciler) appendDegradedConditions(installation *v1alpha1.RHMI) []metav1.Condition {
	var conditions []metav1.Condition

	if installation.IsDegraded() {
		r.Log.Warning("Installation degraded")
		conditions = append(conditions, installation.DegradedCondition())
	} else {
		conditions = append(conditions, installation.NonDegradedCondition())
	}

	return conditions
}

// UpdateAddonInstanceWithConditions finds the addon instance and updates the status with coniditions
func (r *StatusReconciler) updateAddonInstanceWithConditions(ctx context.Context, addonInstance *addonv1alpha1.AddonInstance, conditions []metav1.Condition) error {
	// Send Pulse to addon operator to report health of addon
	if err := r.addonInstanceClient.SendPulse(ctx, *addonInstance, addoninstance.WithConditions(conditions)); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch for addon instances in the specified namespace with the specified name
		For(&addonv1alpha1.AddonInstance{}, builder.WithPredicates(predicate.And(utils.NamePredicate(r.cfg.AddonInstanceName), utils.NamespacePredicate(r.cfg.AddonInstanceNamespace)))).
		// Watch for RHMI changes in the same namespace as the Addon Instance
		Watches(&source.Kind{Type: &v1alpha1.RHMI{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(utils.NamespacePredicate(r.cfg.AddonInstanceNamespace))).
		Complete(r)
}
