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
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/resources/poddistribution"
	"github.com/integr8ly/integreatly-operator/pkg/webhooks"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	usersv1 "github.com/openshift/api/user/v1"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/products"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/version"
)

const (
	deletionFinalizer                = "finalizer/configmaps"
	DefaultInstallationName          = "rhmi"
	ManagedApiInstallationName       = "rhoam"
	DefaultInstallationConfigMapName = "installation-config"
	DefaultCloudResourceConfigName   = "cloud-resource-config"
	alertingEmailAddressEnvName      = "ALERTING_EMAIL_ADDRESS"
	buAlertingEmailAddressEnvName    = "BU_ALERTING_EMAIL_ADDRESS"
	installTypeEnvName               = "INSTALLATION_TYPE"
	priorityClassNameEnvName         = "PRIORITY_CLASS_NAME"
	managedServicePriorityClassName  = "rhoam-pod-priority"
)

var (
	productVersionMismatchFound bool
	log                         = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "installation_controller"})
)

// RHMIReconciler reconciles a RHMI object
type RHMIReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme

	mgr             ctrl.Manager
	controller      controller.Controller
	restConfig      *rest.Config
	customInformers map[string]map[string]*cache.Informer

	productsInstallationLoader marketplace.ProductsInstallationLoader
}

func New(mgr ctrl.Manager) *RHMIReconciler {
	restconfig := controllerruntime.GetConfigOrDie()
	restconfig.Timeout = 10 * time.Second
	return &RHMIReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		mgr:             mgr,
		restConfig:      restconfig,
		customInformers: make(map[string]map[string]*cache.Informer),

		productsInstallationLoader: marketplace.NewFSProductInstallationLoader(
			marketplace.GetProductsInstallationPath(),
		),
	}
}

// ClusterRole permissions

// +kubebuilder:rbac:groups=integreatly.org;applicationmonitoring.integreatly.org,resources=*,verbs=*
// +kubebuilder:rbac:groups=integreatly.org,resources=rhmis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=integreatly.org,resources=rhmis/status,verbs=get;update;patch

// We need to create consolelinks which are cluster level objects
// +kubebuilder:rbac:groups=console.openshift.io,resources=consolelinks,verbs=get;create;update;delete

// We are using ProjectRequests API to create namespaces where we automatically become admins
// +kubebuilder:rbac:groups="";project.openshift.io,resources=projectrequests,verbs=create

// Preflight check for existing installations of products
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;get;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=list;get;watch
// +kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs,verbs=list;get;watch

// We need to get console route for solution explorer
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get

// Reconciling Fuse templates and image streams
// +kubebuilder:rbac:groups=template.openshift.io,resources=templates,verbs=get;create;update;delete
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;create;update;delete

// Registry pull secret needs to be read to be then copied into some RHMI namespaces
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get,resourceNames=pull-secret

// We need to read this Secret from openshift-monitoring namespace in order to setup our monitoring stack
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get,resourceNames=grafana-datasources

// OAuthClients are used for login into products with OpenShift User identity
// +kubebuilder:rbac:groups=oauth.openshift.io,resources=oauthclients,verbs=create;get;update;delete

// Updating the samples operator config cr to ignore fuse imagestreams and templates
// +kubebuilder:rbac:groups=samples.operator.openshift.io,resources=configs,verbs=get;update,resourceNames=cluster

// Permissions needed for our namespaces, but not given by "admin" role
// - Namespace update permissions are needed for setting labels
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=update
// - Installation of product operators
// +kubebuilder:rbac:groups=operators.coreos.com,resources=catalogsources;operatorgroups,verbs=create;list;get
// +kubebuilder:rbac:groups=operators.coreos.com,resources=catalogsources,verbs=update,resourceNames=rhmi-registry-cs
// +kubebuilder:rbac:groups=operators.coreos.com,resources=installplans,verbs=update

// Monitoring resources not covered by namespace "admin" permissions
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules;servicemonitors;podmonitors,verbs=get;list;create;update;delete

// Adding a rolebinding to the monitoring federation namespace
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;create;update;delete

// Permission to fetch identity to get email for created Keycloak users in openshift realm
// +kubebuilder:rbac:groups=user.openshift.io,resources=identities,verbs=get

// Permission to manage ValidatingWebhookConfiguration CRs pointing to the webhook server
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations;mutatingwebhookconfigurations,verbs=get;watch;list;create;update;delete

// Permission to get the ConfigMap that embeds the CSV for an InstallPlan
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get

// Permission for marin3r resources
// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoyconfigs,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservices,verbs=get;list;watch;create;update;delete

// +kubebuilder:rbac:groups=scheduling.k8s.io,resources=*,verbs=*

// Permission to list nodes in order to determine if a cluster is multi-az
// +kubebuilder:rbac:groups="",resources=nodes,verbs=list

// Permission to get cluster infrastructure details for alerting
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions;infrastructures;oauths,verbs=get;list

// Permission to remove crd for the marin3r operator upgrade from 0.5.1 to 0.7.0
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=delete;get;list

// Role permissions

// +kubebuilder:rbac:groups="",resources=pods;events;configmaps;secrets,verbs=list;get;watch;create;update;patch,namespace=integreatly-operator

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups="",resources=services;services/finalizers,verbs=get;create;list;watch;update;delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create,namespace=integreatly-operator

// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers;replicasets;statefulsets,verbs=update;get,namespace=integreatly-operator

// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;create;update;delete;watch,namespace=integreatly-operator

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;create;update;delete;watch,namespace=integreatly-operator

// +kubebuilder:rbac:groups="",resources=pods;services;endpoints,verbs=get;list;watch,namespace=integreatly-operator

// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoyconfigs,verbs=get;list;watch;create;update;delete,namespace=integreatly-operator

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservices,verbs=get;list;watch;create;update;delete,namespace=integreatly-operator

func (r *RHMIReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	// your logic here
	installInProgress := false
	installation := &rhmiv1alpha1.RHMI{}
	err := r.Get(context.TODO(), request.NamespacedName, installation)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	originalInstallation := installation.DeepCopy()

	retryRequeue := ctrl.Result{
		Requeue:      true,
		RequeueAfter: 10 * time.Second,
	}

	installType, err := TypeFactory(installation.Spec.Type)
	if err != nil {
		return ctrl.Result{}, err
	}
	installationCfgMap := os.Getenv("INSTALLATION_CONFIG_MAP")
	if installationCfgMap == "" {
		installationCfgMap = installation.Spec.NamespacePrefix + DefaultInstallationConfigMapName
	}

	cssreAlertingEmailAddress := os.Getenv(alertingEmailAddressEnvName)
	if installation.Spec.AlertingEmailAddresses.CSSRE == "" && cssreAlertingEmailAddress != "" {
		log.Info("Adding CS-SRE alerting email address to RHMI CR")
		installation.Spec.AlertingEmailAddresses.CSSRE = cssreAlertingEmailAddress
		err = r.Update(context.TODO(), installation)
		if err != nil {
			log.Error("Error while copying alerting email addresses to RHMI CR", err)
		}
	}

	buAlertingEmailAddress := os.Getenv(buAlertingEmailAddressEnvName)
	if installation.Spec.AlertingEmailAddresses.BusinessUnit == "" && buAlertingEmailAddress != "" {
		log.Info("Adding BU alerting email address to RHMI CR")
		installation.Spec.AlertingEmailAddresses.BusinessUnit = buAlertingEmailAddress
		err = r.Update(context.TODO(), installation)
		if err != nil {
			log.Error("Error while copying alerting email addresses to RHMI CR", err)
		}
	}

	customerAlertingEmailAddress, ok, err := addon.GetStringParameterByInstallType(
		context.TODO(),
		r.Client,
		rhmiv1alpha1.InstallationType(installation.Spec.Type),
		installation.Namespace,
		"notification-email",
	)
	if err != nil {
		log.Error("failed while retrieving addon parameter", err)
	} else if ok && customerAlertingEmailAddress != "" && installation.Spec.AlertingEmailAddress != customerAlertingEmailAddress {
		log.Info("Updating customer email address from parameter")
		installation.Spec.AlertingEmailAddress = customerAlertingEmailAddress
		if err := r.Update(context.TODO(), installation); err != nil {
			log.Error("Error while updating customer email address to RHMI CR", err)
		}
	}

	// gets the products from the install type to expose rhmi status metric
	stages := make([]rhmiv1alpha1.RHMIStageStatus, 0)
	for _, stage := range installType.GetInstallStages() {
		stages = append(stages, rhmiv1alpha1.RHMIStageStatus{
			Name:     stage.Name,
			Phase:    "",
			Products: stage.Products,
		})
	}
	metrics.SetRHMIStatus(installation)

	configManager, err := config.NewManager(context.TODO(), r.Client, request.NamespacedName.Namespace, installationCfgMap, installation)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile the webhooks
	if err := webhooks.Config.Reconcile(context.TODO(), r.Client, installation); err != nil {
		return ctrl.Result{}, err
	}

	if !resources.Contains(installation.GetFinalizers(), deletionFinalizer) && installation.GetDeletionTimestamp() == nil {
		installation.SetFinalizers(append(installation.GetFinalizers(), deletionFinalizer))
	}

	if installation.Status.Stages == nil {
		installation.Status.Stages = map[rhmiv1alpha1.StageName]rhmiv1alpha1.RHMIStageStatus{}
	}

	installationQuota := &quota.Quota{}
	if installation.Spec.Type == string(rhmiv1alpha1.InstallationTypeManagedApi) {
		if err = r.processQuota(installation, request.Namespace, installationQuota); err != nil {
			log.Error("Error while processing the Quota", err)
			if err = r.Status().Update(context.TODO(), installation); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		if err = r.Status().Update(context.TODO(), installation); err != nil {
			return ctrl.Result{}, err
		}
		metrics.SetQuota(string(installation.Status.Stage), installation.Status.Quota, installation.Status.ToQuota)
	}

	// either not checked, or rechecking preflight checks
	if installation.Status.PreflightStatus == rhmiv1alpha1.PreflightInProgress ||
		installation.Status.PreflightStatus == rhmiv1alpha1.PreflightFail {
		return r.preflightChecks(installation, installType, configManager)
	}

	// If the CR is being deleted, handle uninstall and return
	if installation.DeletionTimestamp != nil {
		return r.handleUninstall(installation, installType)
	}

	// If no current or target version is set this is the first installation of rhmi.
	if upgradeFirstReconcile(installation) || firstInstallFirstReconcile(installation) {
		installation.Status.ToVersion = version.GetVersionByType(installation.Spec.Type)
		log.Infof("Setting installation.Status.ToVersion on initial install", l.Fields{"version": version.GetVersionByType(installation.Spec.Type)})
		if err := r.Status().Update(context.TODO(), installation); err != nil {
			return ctrl.Result{}, err
		}
		metrics.SetRhmiVersions(string(installation.Status.Stage), installation.Status.Version, installation.Status.ToVersion, installation.CreationTimestamp.Unix())
	}

	// Check for stage complete to avoid setting the metric when installation is happening
	if string(installation.Status.Stage) == "complete" {
		metrics.SetRhmiVersions(string(installation.Status.Stage), installation.Status.Version, installation.Status.ToVersion, installation.CreationTimestamp.Unix())

		metrics.SetQuota(string(installation.Status.Stage), installation.Status.Quota, installation.Status.ToQuota)
	}

	alertsClient, err := k8sclient.New(r.mgr.GetConfig(), k8sclient.Options{
		Scheme: r.mgr.GetScheme(),
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error creating client for alerts: %v", err)
	}
	// reconciles rhmi installation alerts
	_, err = r.newAlertsReconciler(installation).ReconcileAlerts(context.TODO(), alertsClient)
	if err != nil {
		log.Error("Error reconciling alerts for the rhmi installation", err)
	}

	for _, stage := range installType.GetInstallStages() {
		var err error
		var stagePhase rhmiv1alpha1.StatusPhase
		var stageLog = l.NewLoggerWithContext(l.Fields{l.StageLogContext: stage.Name})

		if stage.Name == rhmiv1alpha1.BootstrapStage {
			stagePhase, err = r.bootstrapStage(installation, configManager, stageLog)
		} else {
			stagePhase, err = r.processStage(installation, &stage, configManager, installationQuota, stageLog)
		}

		if installation.Status.Stages == nil {
			installation.Status.Stages = make(map[rhmiv1alpha1.StageName]rhmiv1alpha1.RHMIStageStatus)
		}
		installation.Status.Stages[stage.Name] = rhmiv1alpha1.RHMIStageStatus{
			Name:     stage.Name,
			Phase:    stagePhase,
			Products: stage.Products,
		}

		if err != nil {
			installation.Status.LastError = err.Error()
		} else {
			installation.Status.LastError = ""
		}

		//don't move to next stage until current stage is complete
		if stagePhase != rhmiv1alpha1.PhaseCompleted {
			stageLog.Infof("Status", l.Fields{"stage.Name": stage.Name, "stagePhase": stagePhase})
			installInProgress = true
			break
		}
	}

	// Entered on first reconcile where all stages reported complete after an upgrade / install
	if installation.Status.ToVersion == version.GetVersionByType(installation.Spec.Type) && !installInProgress && !productVersionMismatchFound {
		installation.Status.Version = version.GetVersionByType(installation.Spec.Type)
		installation.Status.ToVersion = ""
		metrics.SetRhmiVersions(string(installation.Status.Stage), installation.Status.Version, installation.Status.ToVersion, installation.CreationTimestamp.Unix())
		if installation.Spec.Type == string(rhmiv1alpha1.InstallationTypeManagedApi) {
			installation.Status.Quota = installationQuota.GetName()
			installation.Status.ToQuota = ""
		}
		log.Info("installation completed successfully")
	}

	// Entered on every reconcile where all stages reported complete
	if !installInProgress {
		installation.Status.Stage = rhmiv1alpha1.StageName("complete")
		metrics.RHMIStatusAvailable.Set(1)
		retryRequeue.RequeueAfter = 5 * time.Minute
		if installation.Spec.RebalancePods {
			r.reconcilePodDistribution(installation)
		}

		if installation.Spec.Type == string(rhmiv1alpha1.InstallationTypeManagedApi) {
			if installationQuota.IsUpdated() {
				installation.Status.Quota = installationQuota.GetName()
				installation.Status.ToQuota = ""
				metrics.SetQuota(string(installation.Status.Stage), installation.Status.Quota,
					installation.Status.ToQuota)
			}
		}
	}
	metrics.SetRHMIStatus(installation)

	err = r.updateStatusAndObject(originalInstallation, installation)
	return retryRequeue, err
}

func (r *RHMIReconciler) reconcilePodDistribution(installation *rhmiv1alpha1.RHMI) {

	serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{})
	if err != nil {
		log.Error("Error getting server client for pod distribution", err)
		installation.Status.LastError = err.Error()
		return
	}
	mErr := poddistribution.ReconcilePodDistribution(context.TODO(), serverClient, installation.Spec.NamespacePrefix, installation.Spec.Type)
	if mErr != nil && len(mErr.Errors) > 0 {
		logrus.Errorf("Error reconciling pod distributions %v", mErr)
		installation.Status.LastError = mErr.Error()
	}
}

func (r *RHMIReconciler) updateStatusAndObject(original, installation *rhmiv1alpha1.RHMI) error {
	if !reflect.DeepEqual(original.Status, installation.Status) {
		log.Info("updating status")
		err := r.Status().Update(context.TODO(), installation)
		if err != nil {
			return err
		}
	}

	if !reflect.DeepEqual(original, installation) {
		log.Info("updating object")
		err := r.Update(context.TODO(), installation)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RHMIReconciler) handleUninstall(installation *rhmiv1alpha1.RHMI, installationType *Type) (ctrl.Result, error) {
	retryRequeue := ctrl.Result{
		Requeue:      true,
		RequeueAfter: 10 * time.Second,
	}
	installationCfgMap := os.Getenv("INSTALLATION_CONFIG_MAP")
	if installationCfgMap == "" {
		installationCfgMap = installation.Spec.NamespacePrefix + DefaultInstallationConfigMapName
	}
	configManager, err := config.NewManager(context.TODO(), r.Client, installation.Namespace, installationCfgMap, installation)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get the PrometheusRules with the integreatly label
	// and delete them to ensure no alerts are firing during
	// installation
	//
	// We have to use unstructured instead of the typed
	// structs as the Items field contains pointers and there's
	// a bug on the client library:
	// https://github.com/kubernetes-sigs/controller-runtime/issues/656
	alerts := &unstructured.UnstructuredList{}
	alerts.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Kind:    "PrometheusRule",
		Version: "v1",
	})
	ls, _ := labels.Parse("integreatly=yes")
	if err := r.Client.List(context.TODO(), alerts, &k8sclient.ListOptions{
		LabelSelector: ls,
	}); err != nil {
		return ctrl.Result{}, err
	}

	for _, alert := range alerts.Items {
		if err := r.Client.Delete(context.TODO(), &alert); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set metrics status to unavailable
	metrics.RHMIStatusAvailable.Set(0)

	installation.Status.Stage = rhmiv1alpha1.StageName("deletion")
	installation.Status.LastError = ""

	// updates rhmi status metric to deletion
	metrics.SetRHMIStatus(installation)

	// Clean up the products which have finalizers associated to them
	merr := &resources.MultiErr{}
	finalizers := []string{}
	for _, finalizer := range installation.Finalizers {
		finalizers = append(finalizers, finalizer)
	}
	for _, stage := range installationType.UninstallStages {
		pendingUninstalls := false
		for product := range stage.Products {
			productName := string(product)
			log.Infof("Uninstalling ", l.Fields{"product": productName, "stage": stage.Name})
			productStatus := installation.GetProductStatusObject(product)
			//if the finalizer for this product is not present, move to the next product
			for _, productFinalizer := range finalizers {
				if !strings.Contains(productFinalizer, productName) {
					continue
				}
				reconciler, err := products.NewReconciler(product, r.restConfig, configManager, installation, r.mgr, log, r.productsInstallationLoader)
				if err != nil {
					merr.Add(fmt.Errorf("Failed to build reconciler for product %s: %w", productName, err))
				}
				serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{
					Scheme: r.mgr.GetScheme(),
				})
				if err != nil {
					merr.Add(fmt.Errorf("Failed to create server client for %s: %w", productName, err))
				}
				phase, err := reconciler.Reconcile(context.TODO(), installation, productStatus, serverClient,
					quota.QuotaProductConfig{})
				if err != nil {
					merr.Add(fmt.Errorf("Failed to reconcile product %s: %w", productName, err))
				}
				if phase != rhmiv1alpha1.PhaseCompleted {
					pendingUninstalls = true
				}
				log.Infof("Current phase ", l.Fields{"productName": productName, "phase": phase})
			}
		}
		//don't move to next stage until all products in this stage are removed
		//update CR and return
		if pendingUninstalls {
			if len(merr.Errors) > 0 {
				installation.Status.LastError = merr.Error()
				r.Client.Status().Update(context.TODO(), installation)
			}
			err = r.Client.Update(context.TODO(), installation)
			if err != nil {
				merr.Add(err)
			}
			return retryRequeue, nil
		}
	}

	//all products gone and no errors, tidy up bootstrap stuff
	if len(installation.Finalizers) == 1 && installation.Finalizers[0] == deletionFinalizer {
		log.Infof("Finalizers: ", l.Fields{"length": len(installation.Finalizers)})
		// delete ConfigMap after all product finalizers finished
		if err := r.Client.Delete(context.TODO(), &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: installationCfgMap, Namespace: installation.Namespace}}); err != nil && !k8serr.IsNotFound(err) {
			merr.Add(fmt.Errorf("failed to remove installation ConfigMap: %w", err))
			installation.Status.LastError = merr.Error()
			err = r.Client.Update(context.TODO(), installation)
			if err != nil {
				merr.Add(err)
			}
			return retryRequeue, merr
		}

		if err = r.handleCROConfigDeletion(*installation); err != nil && !k8serr.IsNotFound(err) {
			merr.Add(fmt.Errorf("failed to remove Cloud Resource ConfigMap: %w", err))
			installation.Status.LastError = merr.Error()
			err = r.Update(context.TODO(), installation)
			if err != nil {
				merr.Add(err)
			}
			return retryRequeue, merr
		}

		installation.SetFinalizers(resources.Remove(installation.GetFinalizers(), deletionFinalizer))

		err = r.Update(context.TODO(), installation)
		if err != nil {
			merr.Add(err)
			return retryRequeue, merr
		}

		if err := addon.UninstallOperator(context.TODO(), r.Client, installation); err != nil {
			merr.Add(err)
			return retryRequeue, merr
		}

		log.Info("uninstall completed")
		return ctrl.Result{}, nil
	}

	log.Info("updating uninstallation object")
	// no finalizers left, update object
	err = r.Update(context.TODO(), installation)
	return retryRequeue, err
}

func firstInstallFirstReconcile(installation *rhmiv1alpha1.RHMI) bool {
	status := installation.Status
	return status.Version == "" && status.ToVersion == ""
}

// An upgrade is one in which the install plan was manually approved.
// In which case the toVersion field has not been set
func upgradeFirstReconcile(installation *rhmiv1alpha1.RHMI) bool {
	status := installation.Status
	return status.Version != "" && status.ToVersion == "" && status.Version != version.GetVersionByType(installation.Spec.Type)
}

func (r *RHMIReconciler) preflightChecks(installation *rhmiv1alpha1.RHMI, installationType *Type, configManager *config.Manager) (ctrl.Result, error) {
	log.Info("Running preflight checks..")
	installation.Status.Stage = rhmiv1alpha1.StageName("Preflight Checks")
	result := ctrl.Result{
		Requeue:      true,
		RequeueAfter: 10 * time.Second,
	}

	eventRecorder := r.mgr.GetEventRecorderFor("Preflight Checks")

	// Validate the env vars used by the operator
	if err := checkEnvVars(map[string]func(string, bool) error{
		resources.AntiAffinityRequiredEnvVar: optionalEnvVar(func(s string) error {
			_, err := strconv.ParseBool(s)
			return err
		}),
		rhmiv1alpha1.EnvKeyAlertSMTPFrom: requiredEnvVar(func(s string) error {
			if s == "" {
				return fmt.Errorf(" env var %s is required ", rhmiv1alpha1.EnvKeyAlertSMTPFrom)
			}
			return nil
		}),
	}); err != nil {
		return result, err
	}

	if strings.ToLower(installation.Spec.UseClusterStorage) != "true" && strings.ToLower(installation.Spec.UseClusterStorage) != "false" {
		installation.Status.PreflightStatus = rhmiv1alpha1.PreflightFail
		installation.Status.PreflightMessage = "Spec.useClusterStorage must be set to either 'true' or 'false' to continue"
		_ = r.Status().Update(context.TODO(), installation)
		log.Warning("preflight checks failed on useClusterStorage value")
		return result, nil
	}

	if installation.Spec.Type == string(rhmiv1alpha1.InstallationTypeManaged) || installation.Spec.Type == string(rhmiv1alpha1.InstallationTypeManagedApi) {
		requiredSecrets := []string{installation.Spec.PagerDutySecret, installation.Spec.DeadMansSnitchSecret}

		for _, secretName := range requiredSecrets {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: installation.Namespace,
				},
			}
			if exists, err := resources.Exists(context.TODO(), r.Client, secret); err != nil {
				return ctrl.Result{}, err
			} else if !exists {
				preflightMessage := fmt.Sprintf("Could not find %s secret in %s namespace", secret.Name, installation.Namespace)
				log.Info(preflightMessage)
				eventRecorder.Event(installation, "Warning", rhmiv1alpha1.EventProcessingError, preflightMessage)

				installation.Status.PreflightStatus = rhmiv1alpha1.PreflightFail
				installation.Status.PreflightMessage = preflightMessage
				_ = r.Status().Update(context.TODO(), installation)

				return ctrl.Result{}, err
			}
			log.Infof("found required secret", l.Fields{"secret": secretName})
			eventRecorder.Eventf(installation, "Normal", rhmiv1alpha1.EventPreflightCheckPassed,
				"found required secret: %s", secretName)
		}
	}

	log.Info("getting namespaces")
	namespaces := &corev1.NamespaceList{}
	err := r.List(context.TODO(), namespaces)
	if err != nil {
		// could not list namespaces, keep trying
		log.Warningf("error listing namespaces", l.Fields{"error": err.Error()})
		return result, err
	}

	for _, ns := range namespaces.Items {
		products, err := r.checkNamespaceForProducts(ns, installation, installationType, configManager)
		if err != nil {
			// error searching for existing products, keep trying
			log.Info("error looking for existing deployments, will retry")
			return result, err
		}
		if len(products) != 0 {
			//found one or more conflicting products
			installation.Status.PreflightStatus = rhmiv1alpha1.PreflightFail
			installation.Status.PreflightMessage = "found conflicting packages: " + strings.Join(products, ", ") + ", in namespace: " + ns.GetName()
			log.Info("found conflicting packages: " + strings.Join(products, ", ") + ", in namespace: " + ns.GetName())
			_ = r.Status().Update(context.TODO(), installation)
			return result, err
		}
	}

	installation.Status.PreflightStatus = rhmiv1alpha1.PreflightSuccess
	installation.Status.PreflightMessage = "preflight checks passed"
	err = r.Status().Update(context.TODO(), installation)
	if err != nil {
		log.Infof("error updating status", l.Fields{"error": err.Error()})
	}
	return result, nil
}

func (r *RHMIReconciler) checkNamespaceForProducts(ns corev1.Namespace, installation *rhmiv1alpha1.RHMI, installationType *Type, configManager *config.Manager) ([]string, error) {
	foundProducts := []string{}
	if strings.HasPrefix(ns.Name, "openshift-") {
		return foundProducts, nil
	}
	if strings.HasPrefix(ns.Name, "kube-") {
		return foundProducts, nil
	}
	// new client to avoid caching issues
	serverClient, _ := k8sclient.New(r.restConfig, k8sclient.Options{
		Scheme: r.mgr.GetScheme(),
	})
	for _, stage := range installationType.InstallStages {
		for _, product := range stage.Products {
			reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, installation, r.mgr, log, r.productsInstallationLoader)
			if err != nil {
				return foundProducts, err
			}
			search := reconciler.GetPreflightObject(ns.Name)
			if search == nil {
				continue
			}
			exists, err := resources.Exists(context.TODO(), serverClient, search)
			if err != nil {
				return foundProducts, err
			} else if exists {
				log.Infof("Found conflicts ", l.Fields{"product": product.Name})
				foundProducts = append(foundProducts, string(product.Name))
			}
		}
	}
	return foundProducts, nil
}

func (r *RHMIReconciler) bootstrapStage(installation *rhmiv1alpha1.RHMI, configManager config.ConfigReadWriter, log l.Logger) (rhmiv1alpha1.StatusPhase, error) {
	installation.Status.Stage = rhmiv1alpha1.BootstrapStage
	mpm := marketplace.NewManager()

	reconciler, err := NewBootstrapReconciler(configManager, installation, mpm, r.mgr.GetEventRecorderFor(string(rhmiv1alpha1.BootstrapStage)), log)
	if err != nil {
		return rhmiv1alpha1.PhaseFailed, fmt.Errorf("failed to build a reconciler for Bootstrap: %w", err)
	}
	serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{
		Scheme: r.mgr.GetScheme(),
	})
	if err != nil {
		return rhmiv1alpha1.PhaseFailed, fmt.Errorf("could not create server client: %w", err)
	}
	phase, err := reconciler.Reconcile(context.TODO(), installation, serverClient)
	if err != nil || phase == rhmiv1alpha1.PhaseFailed {
		return rhmiv1alpha1.PhaseFailed, fmt.Errorf("Bootstrap stage reconcile failed: %w", err)
	}

	return phase, nil
}

func (r *RHMIReconciler) processStage(installation *rhmiv1alpha1.RHMI, stage *Stage,
	configManager config.ConfigReadWriter, quotaconfig *quota.Quota, _ l.Logger) (rhmiv1alpha1.StatusPhase, error) {
	incompleteStage := false
	productVersionMismatchFound = false

	var mErr error
	productsAux := make(map[rhmiv1alpha1.ProductName]rhmiv1alpha1.RHMIProductStatus)
	installation.Status.Stage = stage.Name

	for productName, product := range stage.Products {
		productLog := l.NewLoggerWithContext(l.Fields{l.ProductLogContext: product.Name})

		reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, installation, r.mgr, productLog, r.productsInstallationLoader)

		if err != nil {
			return rhmiv1alpha1.PhaseFailed, fmt.Errorf("failed to build a reconciler for %s: %w", product.Name, err)
		}

		if !reconciler.VerifyVersion(installation) {
			productVersionMismatchFound = true
		}

		serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{
			Scheme: r.mgr.GetScheme(),
		})
		if err != nil {
			return rhmiv1alpha1.PhaseFailed, fmt.Errorf("could not create server client: %w", err)
		}
		product.Status, err = reconciler.Reconcile(context.TODO(), installation, &product, serverClient, quotaconfig.GetProduct(productName))

		if err != nil {
			if mErr == nil {
				mErr = &resources.MultiErr{}
			}
			mErr.(*resources.MultiErr).Add(fmt.Errorf("failed installation of %s: %w", product.Name, err))
		}

		// Verify that watches for this product CRDs have been created
		config, err := configManager.ReadProduct(product.Name)
		if err != nil {
			return rhmiv1alpha1.PhaseFailed, fmt.Errorf("Failed to read product config for %s: %v", string(product.Name), err)
		}

		if product.Status == rhmiv1alpha1.PhaseCompleted {
			for _, crd := range config.GetWatchableCRDs() {
				namespace := config.GetNamespace()
				gvk := crd.GetObjectKind().GroupVersionKind().String()
				if r.customInformers[gvk] == nil {
					r.customInformers[gvk] = make(map[string]*cache.Informer)
				}
				if r.customInformers[gvk][config.GetNamespace()] == nil {
					err = r.addCustomInformer(crd, namespace)
					if err != nil {
						return rhmiv1alpha1.PhaseFailed, fmt.Errorf("Failed to create a %s CRD watch for %s: %v", gvk, string(product.Name), err)
					}
				} else if !(*r.customInformers[gvk][config.GetNamespace()]).HasSynced() {
					return rhmiv1alpha1.PhaseFailed, fmt.Errorf("A %s CRD Informer for %s has not synced", gvk, string(product.Name))
				}
			}
		}

		//found an incomplete product
		if product.Status != rhmiv1alpha1.PhaseCompleted {
			incompleteStage = true
		}
		productsAux[product.Name] = product
		*stage = Stage{Name: stage.Name, Products: productsAux}
	}

	//some products in this stage have not installed successfully yet
	if incompleteStage {
		return rhmiv1alpha1.PhaseInProgress, mErr
	}
	return rhmiv1alpha1.PhaseCompleted, mErr
}

// handle the deletion of CRO config map
func (r *RHMIReconciler) handleCROConfigDeletion(rhmi rhmiv1alpha1.RHMI) error {
	// get cloud resource config map
	croConf := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: rhmi.Namespace, Name: DefaultCloudResourceConfigName}, croConf)
	if err != nil {
		return err
	}

	// remove cloud resource config deletion finalizer if it exists
	if resources.Contains(croConf.Finalizers, deletionFinalizer) {
		croConf.SetFinalizers(resources.Remove(croConf.Finalizers, deletionFinalizer))

		if err := r.Update(context.TODO(), croConf); err != nil {
			return fmt.Errorf("error occurred trying to update cro config map %w", err)
		}
	}

	// remove cloud resource config map
	err = r.Delete(context.TODO(), croConf)
	if err != nil && !k8serr.IsNotFound(err) {
		return fmt.Errorf("error occurred trying to delete cro config map, %w", err)
	}

	return nil
}

func (r *RHMIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Creates a new managed install CR if it is not available
	kubeConfig := mgr.GetConfig()
	client, err := k8sclient.New(kubeConfig, k8sclient.Options{
		Scheme: mgr.GetScheme(),
	})
	err = r.createInstallationCR(context.Background(), client)
	if err != nil {
		return err
	}

	enqueueAllInstallations := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: installationMapper{context: context.TODO(), client: mgr.GetClient()},
	}

	// Instead of calling .Complete(r), we call .Build(r), which
	// does the same but returns the controller instance, to be
	// stored in the reconciler
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&rhmiv1alpha1.RHMI{}).
		Watches(&source.Kind{Type: &usersv1.User{}}, enqueueAllInstallations).
		Watches(&source.Kind{Type: &corev1.Secret{}}, enqueueAllInstallations).
		Watches(&source.Kind{Type: &usersv1.Group{}}, enqueueAllInstallations).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, enqueueAllInstallations, builder.WithPredicates(newObjectPredicate(isName(marin3rconfig.RateLimitConfigMapName)))).
		Build(r)

	if err != nil {
		return err
	}

	r.controller = controller

	return nil
}

func (r *RHMIReconciler) createInstallationCR(ctx context.Context, serverClient k8sclient.Client) error {
	namespace, err := resources.GetWatchNamespace()
	if err != nil {
		return err
	}

	logrus.Infof("Looking for rhmi CR in %s namespace", namespace)

	installationList := &rhmiv1alpha1.RHMIList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	err = serverClient.List(ctx, installationList, listOpts...)
	if err != nil {
		return fmt.Errorf("Could not get a list of rhmi CR: %w", err)
	}

	installation := &rhmiv1alpha1.RHMI{}
	// Creates installation CR in case there is none
	if len(installationList.Items) == 0 {
		useClusterStorage, _ := os.LookupEnv("USE_CLUSTER_STORAGE")
		rebalancePods := getRebalancePods()
		cssreAlertingEmailAddress, _ := os.LookupEnv(alertingEmailAddressEnvName)
		buAlertingEmailAddress, _ := os.LookupEnv(buAlertingEmailAddressEnvName)

		installType, _ := os.LookupEnv(installTypeEnvName)
		priorityClassName, _ := os.LookupEnv(priorityClassNameEnvName)

		logrus.Infof("Creating a %s rhmi CR with USC %s, as no CR rhmis were found in %s namespace", installType, useClusterStorage, namespace)

		if installType == "" {
			installType = string(rhmiv1alpha1.InstallationTypeManaged)
		}

		if installType == string(rhmiv1alpha1.InstallationTypeManagedApi) && priorityClassName == "" {
			priorityClassName = managedServicePriorityClassName
		}

		customerAlertingEmailAddress, _, err := addon.GetStringParameterByInstallType(
			ctx,
			serverClient,
			rhmiv1alpha1.InstallationType(installType),
			namespace,
			"notification-email",
		)
		if err != nil {
			return fmt.Errorf("failed while retrieving addon parameter: %w", err)
		}

		namespaceSegments := strings.Split(namespace, "-")
		namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"

		installation = &rhmiv1alpha1.RHMI{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getCrName(installType),
				Namespace: namespace,
			},
			Spec: rhmiv1alpha1.RHMISpec{
				Type:                 installType,
				NamespacePrefix:      namespacePrefix,
				RebalancePods:        rebalancePods,
				SelfSignedCerts:      false,
				SMTPSecret:           namespacePrefix + "smtp",
				DeadMansSnitchSecret: namespacePrefix + "deadmanssnitch",
				PagerDutySecret:      namespacePrefix + "pagerduty",
				UseClusterStorage:    useClusterStorage,
				AlertingEmailAddress: customerAlertingEmailAddress,
				AlertingEmailAddresses: rhmiv1alpha1.AlertingEmailAddresses{
					BusinessUnit: buAlertingEmailAddress,
					CSSRE:        cssreAlertingEmailAddress,
				},
				OperatorsInProductNamespace: false, // e2e tests and Makefile need to be updated when default is changed
				PriorityClassName:           priorityClassName,
			},
		}

		err = serverClient.Create(ctx, installation)
		if err != nil {
			return fmt.Errorf("Could not create rhmi CR in %s namespace: %w", namespace, err)
		}
	} else if len(installationList.Items) == 1 {
		installation = &installationList.Items[0]
	} else {
		return fmt.Errorf("too many rhmi resources found. Expecting 1, found %d rhmi resources in %s namespace", len(installationList.Items), namespace)
	}

	return nil
}
func getRebalancePods() bool {
	rebalance, exists := os.LookupEnv("REBALANCE_PODS")
	if !exists || rebalance == "true" {
		return true
	}
	return false
}

func getCrName(installType string) string {
	if installType == string(rhmiv1alpha1.InstallationTypeManagedApi) {
		return ManagedApiInstallationName
	} else {
		return DefaultInstallationName
	}
}

func (r *RHMIReconciler) addCustomInformer(crd runtime.Object, namespace string) error {
	gvk := crd.GetObjectKind().GroupVersionKind().String()
	mapper, err := apiutil.NewDynamicRESTMapper(r.restConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return fmt.Errorf("Failed to get API Group-Resources: %v", err)
	}
	cache, err := cache.New(r.restConfig, cache.Options{Namespace: namespace, Scheme: r.mgr.GetScheme(), Mapper: mapper})
	if err != nil {
		return fmt.Errorf("Failed to create informer cache in %s namespace: %v", namespace, err)
	}
	informer, err := cache.GetInformerForKind(context.TODO(), crd.GetObjectKind().GroupVersionKind())
	if err != nil {
		return fmt.Errorf("Failed to create informer for %v: %v", crd, err)
	}
	err = r.controller.Watch(&source.Informer{Informer: informer}, &EnqueueIntegreatlyOwner{log: log})
	if err != nil {
		return fmt.Errorf("Failed to create a %s watch in %s namespace: %v", gvk, namespace, err)
	}
	// Adding to Manager, which will start it for us with a correct stop channel
	err = r.mgr.Add(cache)
	if err != nil {
		return fmt.Errorf("Failed to add a %s cache in %s namespace into Manager: %v", gvk, namespace, err)
	}
	r.customInformers[gvk][namespace] = &informer

	// Create a timeout channel for CacheSync as not to block the reconcile
	timeoutChannel := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Second)
		close(timeoutChannel)
	}()
	if !cache.WaitForCacheSync(timeoutChannel) {
		return fmt.Errorf("Failed to sync cache for %s watch in %s namespace", gvk, namespace)
	}

	log.Infof("Cache synced. Successfully initialized.", l.Fields{"watch": gvk, "ns": namespace})
	return nil
}

func checkEnvVars(checks map[string]func(string, bool) error) error {
	for env, check := range checks {
		value, exists := os.LookupEnv(env)
		if err := check(value, exists); err != nil {
			log.Errorf("Validation failure for env var", l.Fields{"envVar": env}, err)
			return fmt.Errorf("validation failure for env var %s: %w", env, err)
		}
	}

	return nil
}

func optionalEnvVar(check func(string) error) func(string, bool) error {
	return func(value string, ok bool) error {
		if !ok {
			return nil
		}

		return check(value)
	}
}

func requiredEnvVar(check func(string) error) func(string, bool) error {
	return func(value string, ok bool) error {
		if !ok {
			return errors.New("required env var not present")
		}

		return check(value)
	}
}

func (r *RHMIReconciler) processQuota(installation *rhmiv1alpha1.RHMI, namespace string,
	installationQuota *quota.Quota) error {
	isQuotaUpdated := false
	quotaParam, found, err := addon.GetStringParameterByInstallType(context.TODO(), r.Client, rhmiv1alpha1.InstallationTypeManagedApi, namespace, addon.QuotaParamName)
	if err != nil {
		return errors.Wrap(err, "Error checking for Quota secret")
	}

	//!found means the param wasn't found so we want to default rather than return
	//if the param is not found after the installation is 1 minute old it means that it wasn't provided to the installation
	//a quota value is required for the installation to begin, the dev value is provided by the make cluster/prepare/quota for local installs
	//or as a required value from an ocm add-on installation
	if !found && !installation.ObjectMeta.CreationTimestamp.Time.Before(time.Now().Add(-(1 * time.Minute))) {
		return errors.Wrap(err, "no quota value provided, quota is a required parameter for a managed-api install")
	}

	// get a configmap from the cluster
	cm := &corev1.ConfigMap{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: quota.ConfigMapName}, cm)
	if err != nil {
		return errors.Wrap(err, "Error getting quota config map")
	}

	// if both are toQuota and Quota are empty this indicates that it's either
	// the first reconcile of an installation or it's the first reconcile of an upgrade to 1.6.0
	// if the secretname is not the same as status.Quota this indicates there has been a quota change
	// to an installation which is already using the Quota functionality.
	// if either case is true set toQuota in the rhmi cr and update the status object and set isQuotaUpdated to true
	if (installation.Status.ToQuota == "" && installation.Status.Quota == "") ||
		quotaParam != installation.Status.Quota {
		installation.Status.ToQuota = quotaParam
		isQuotaUpdated = true
	}

	err = quota.GetQuota(quotaParam, cm, installationQuota, isQuotaUpdated)
	if err != nil {
		return err
	}

	return nil
}
