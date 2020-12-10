package monitoringspec

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/version"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"

	rbac "k8s.io/api/rbac/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace              = "monitoring"
	packageName                               = "monitoringspec"
	roleBindingName                           = "rhmi-prometheus-k8s"
	clusterMonitoringPrometheusServiceAccount = "prometheus-k8s"
	clusterMonitoringNamespace                = "openshift-monitoring"
	roleRefAPIGroup                           = "rbac.authorization.k8s.io"
	roleRefName                               = "rhmi-prometheus-k8s"
	labelSelector                             = "monitoring-key=middleware"
	clonedServiceMonitorLabelKey              = "integreatly.org/cloned-servicemonitor"
	clonedServiceMonitorLabelValue            = "true"
)

type Reconciler struct {
	Config        *config.MonitoringSpec
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	Log           l.Logger
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.MonitoringStage].Products[integreatlyv1alpha1.ProductMonitoringSpec],
		string(integreatlyv1alpha1.VersionMonitoringSpec),
		"",
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI,
	mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger) (*Reconciler, error) {

	config, err := configManager.ReadMonitoringSpec()
	if err != nil {
		return nil, err
	}
	config.SetNamespacePrefix(installation.Spec.NamespacePrefix)
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	return &Reconciler{
		Config:        config,
		extraParams:   make(map[string]string),
		ConfigManager: configManager,
		Log:           logger,
		installation:  installation,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

// Reconcile method for monitorspec
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI,
	product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()),
		func() (integreatlyv1alpha1.StatusPhase, error) {
			r.Log.Info("Phase: Monitoringspec ReconcileFinalizer")

			// Check if namespace is still present before trying to delete it resources
			_, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
			if err != nil && k8serr.IsNotFound(err) {
				r.Log.Info("Spec phase completed")
				//namespace is gone, return complete
				return integreatlyv1alpha1.PhaseCompleted, nil
			}
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace(), r.Log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				if err != nil {
					r.Log.Error("Spec phase removal failure", err)
				}
				return phase, err
			}
			return integreatlyv1alpha1.PhaseInProgress, nil
		},
		r.Log)

	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.Log.Error("failed to reconcile finalizer:", err)
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		}
		return phase, err
	}

	phase, err = r.createNamespace(ctx, serverClient, installation)
	r.Log.Infof("Phase: createNamespace", l.Fields{"status": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.Log.Error("failed to create namespace", err)
			events.HandleError(r.recorder, installation, phase, "Failed to create namespace", err)
		}
		return phase, err
	}

	phase, err = r.reconcileMonitoring(ctx, serverClient, installation)
	r.Log.Infof("Phase: reconcileMonitoring", l.Fields{"status": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.Log.Error("failed to reconcile", err)
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile:", err)
		}
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		r.Log.Error("failed to write config", err)
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Failed to update monitoring config", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not update monitoring config: %w", err)
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.MonitoringStage, r.Config.GetProductName())
	r.Log.Infof("Reconciled successfully", l.Fields{"installation": packageName})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// make the federation namespace discoverable by cluster monitoring
func (r *Reconciler) createNamespace(ctx context.Context, serverClient k8sclient.Client,
	installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		_, err := resources.CreateNSWithProjectRequest(ctx, r.Config.GetNamespace(),
			serverClient, installation, false, true)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	resources.PrepareObject(namespace, installation, false, true)
	err = serverClient.Update(ctx, namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileMonitoring(ctx context.Context, serverClient k8sclient.Client,
	installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {

	//Get list of service monitors in the namespace that has
	//label "integreatly.org/cloned-servicemonitor" set to "true"
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
		k8sclient.MatchingLabels(getClonedServiceMonitorLabel()),
	}

	//Get list of service monitors in the monitoring namespace
	monSermonMap, err := r.getServiceMonitors(ctx, serverClient, listOpts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	//Get the list of namespaces with the given label selector "monitoring-key=middleware"
	namespaces, err := r.getMWMonitoredNamespaces(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	for _, ns := range namespaces.Items {
		//Get list of service monitors in each name space
		listOpts := []k8sclient.ListOption{
			k8sclient.InNamespace(ns.Name),
		}
		serviceMonitorsMap, err := r.getServiceMonitors(ctx, serverClient, listOpts)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		for _, sm := range serviceMonitorsMap {
			//Create a copy of service monitors in the monitoring namespace
			//Create the corresponding rolebindings at each of the service namespace
			key := sm.Namespace + `-` + sm.Name
			delete(monSermonMap, key) // Servicemonitor exists, remove it from the local map
			err := r.reconcileServiceMonitor(ctx, serverClient, sm)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}
			err = r.reconcileRoleBindingsForServiceMonitor(ctx, serverClient, key)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}
		}
	}

	//Clean-up the stale service monitors and rolebindings if any
	if len(monSermonMap) > 0 {
		for _, sm := range monSermonMap {
			//Remove servicemonitor
			err = r.removeServiceMonitor(ctx, serverClient, sm.Namespace, sm.Name)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}
			//Remove rolebindings
			for _, namespace := range sm.Spec.NamespaceSelector.MatchNames {
				err := r.removeRoleandRoleBinding(ctx, serverClient, namespace, roleRefName, roleBindingName)
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, err
				}
			}
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, err
}

func (r *Reconciler) reconcileServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, serviceMonitor *monitoringv1.ServiceMonitor) (err error) {

	if serviceMonitor.Spec.NamespaceSelector.Any {
		r.Log.Warningf("servicemonitor cannot be copied to namespace. Namespace selector has been set to any",
			l.Fields{"serviceMonitor": serviceMonitor.Name, "ns": r.Config.GetNamespace()})
		return nil
	}
	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceMonitor.Namespace + `-` + serviceMonitor.Name,
			Namespace: r.Config.GetNamespace(),
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, sm, func() error {
		// Check if the servicemonitor has no  namespace selectors defined,
		// if not add the namespace
		sm.Spec = serviceMonitor.Spec
		if len(sm.Spec.NamespaceSelector.MatchNames) == 0 {
			sm.Spec.NamespaceSelector.MatchNames = []string{serviceMonitor.Namespace}
		}
		//Add all the original labels and append cloned servicemonitor label
		sm.Labels = serviceMonitor.Labels
		if len(sm.Labels) == 0 {
			sm.Labels = make(map[string]string)
		}
		sm.Labels[clonedServiceMonitorLabelKey] = clonedServiceMonitorLabelValue
		return nil
	})
	if err != nil {
		return err
	}
	if opRes != controllerutil.OperationResultNone {
		r.Log.Infof("Operation result", l.Fields{"serviceMonitor": sm.Name, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileRoleBindingsForServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, serviceMonitorName string) (err error) {
	//Get the service monitor - that was created/updated
	sermon := &monitoringv1.ServiceMonitor{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: serviceMonitorName, Namespace: r.Config.GetNamespace()}, sermon)
	if err != nil {
		return err
	}
	//Create role binding for each of the namespace label selectors
	for _, namespace := range sermon.Spec.NamespaceSelector.MatchNames {
		err := r.reconcileRole(ctx, serverClient, namespace)
		if err != nil {
			return err
		}
		err = r.reconcileRoleBinding(ctx, serverClient, namespace)
		if err != nil {
			return err
		}
	}
	return err
}

func (r *Reconciler) reconcileRole(ctx context.Context,
	serverClient k8sclient.Client, namespace string) (err error) {

	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleRefName,
			Namespace: namespace,
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, role, func() error {
		resources := []string{
			"services",
			"endpoints",
			"pods",
		}

		verbs := []string{
			"get",
			"list",
			"watch",
		}
		apiGroups := []string{""}

		role.Rules = []rbac.PolicyRule{
			{
				APIGroups: apiGroups,
				Resources: resources,
				Verbs:     verbs,
			},
		}
		return nil
	})
	if opRes != controllerutil.OperationResultNone {
		r.Log.Infof("Operation result", l.Fields{"role": roleRefName, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileRoleBinding(ctx context.Context,
	serverClient k8sclient.Client, namespace string) (err error) {

	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: namespace,
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, roleBinding, func() error {
		roleBinding.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      clusterMonitoringPrometheusServiceAccount,
				Namespace: clusterMonitoringNamespace,
			},
		}
		roleBinding.RoleRef = rbac.RoleRef{
			APIGroup: roleRefAPIGroup,
			Kind:     bundle.RoleKind,
			Name:     roleRefName,
		}
		return nil
	})
	if opRes != controllerutil.OperationResultNone {
		r.Log.Infof("Operation result", l.Fields{"roleBinding": roleBindingName, "result": opRes})
	}
	return err
}

func (r *Reconciler) removeServiceMonitor(ctx context.Context,
	serverClient k8sclient.Client, namespace, name string) (err error) {
	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	//Delete the servicemonitor
	err = serverClient.Delete(ctx, sm)
	if err != nil && k8serr.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *Reconciler) removeRoleandRoleBinding(ctx context.Context,
	serverClient k8sclient.Client, namespace, roleName, rbName string) (err error) {

	// Check if the namespace has service monitors
	// if so do not delete the rolebinding
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	serviceMonitorsMap, err := r.getServiceMonitors(ctx, serverClient, listOpts)
	if err != nil {
		return err
	}

	if len(serviceMonitorsMap) > 0 {
		return nil
	}

	//Get the role
	role := &rbac.Role{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: roleName, Namespace: namespace}, role)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the role
	err = serverClient.Delete(ctx, role)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Get the rolebinding
	rb := &rbac.RoleBinding{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: rbName, Namespace: namespace}, rb)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	//Delete the rolebinding
	err = serverClient.Delete(ctx, rb)
	if err != nil && k8serr.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *Reconciler) getServiceMonitors(ctx context.Context,
	serverClient k8sclient.Client,
	listOpts []k8sclient.ListOption) (serviceMonitorsMap map[string]*monitoringv1.ServiceMonitor, err error) {

	if len(listOpts) == 0 {
		return serviceMonitorsMap, fmt.Errorf("List options is empty")
	}
	serviceMonitors := &monitoringv1.ServiceMonitorList{}
	err = serverClient.List(ctx, serviceMonitors, listOpts...)
	if err != nil {
		return serviceMonitorsMap, err
	}
	serviceMonitorsMap = make(map[string]*monitoringv1.ServiceMonitor)
	for _, sm := range serviceMonitors.Items {
		serviceMonitorsMap[sm.Name] = sm
	}
	return serviceMonitorsMap, err
}

func getClonedServiceMonitorLabel() map[string]string {
	return map[string]string{
		clonedServiceMonitorLabelKey: clonedServiceMonitorLabelValue,
	}
}

func (r *Reconciler) getMWMonitoredNamespaces(ctx context.Context,
	serverClient k8sclient.Client) (namespaces *corev1.NamespaceList, err error) {
	ls, err := labels.Parse(labelSelector)
	if err != nil {
		return namespaces, err
	}
	opts := &k8sclient.ListOptions{
		LabelSelector: ls,
	}
	//Get the list of namespaces with the given label selector
	namespaces = &corev1.NamespaceList{}
	err = serverClient.List(ctx, namespaces, opts)
	return namespaces, err
}
