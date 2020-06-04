package installation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/webhooks"
	"github.com/integr8ly/integreatly-operator/version"

	"k8s.io/apimachinery/pkg/types"

	usersv1 "github.com/openshift/api/user/v1"

	"github.com/sirupsen/logrus"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/products"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	deletionFinalizer                = "finalizer/configmaps"
	DefaultInstallationName          = "rhmi"
	DefaultInstallationConfigMapName = "installation-config"
	DefaultInstallationPrefix        = "redhat-rhmi-"
	DefaultCloudResourceConfigName   = "cloud-resource-config"
)

// Add creates a new Installation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, products []string) error {
	return add(mgr, newReconciler(mgr, products))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, products []string) ReconcileInstallation {
	ctx, cancel := context.WithCancel(context.Background())
	restConfig := controllerruntime.GetConfigOrDie()
	return ReconcileInstallation{
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		restConfig:        restConfig,
		productsToInstall: products,
		context:           ctx,
		cancel:            cancel,
		mgr:               mgr,
		customInformers:   make(map[string]map[string]*cache.Informer),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r ReconcileInstallation) error {
	// Create a new controller
	c, err := controller.New("installation-controller", mgr, controller.Options{Reconciler: reconcile.Reconciler(&r)})
	if err != nil {
		return err
	}
	r.controller = c

	// Creates a new managed install CR if it is not available
	kubeConfig := controllerruntime.GetConfigOrDie()
	client, err := k8sclient.New(kubeConfig, k8sclient.Options{})
	err = createInstallationCR(context.Background(), client)
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Installation
	err = c.Watch(&source.Kind{Type: &integreatlyv1alpha1.RHMI{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// custom event handler to enqueue reconcile requests for all installation CRs
	enqueueAllInstallations := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: installationMapper{context: context.TODO(), client: mgr.GetClient()},
	}

	// Watch for changes to users
	err = c.Watch(&source.Kind{Type: &usersv1.User{}}, enqueueAllInstallations)
	if err != nil {
		return err
	}

	// Watch for changes to Secrets (important for SMTP Secret)
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, enqueueAllInstallations)
	if err != nil {
		return err
	}

	// Watch for changes to groups
	err = c.Watch(&source.Kind{Type: &usersv1.Group{}}, enqueueAllInstallations)
	if err != nil {
		return err
	}

	return nil
}

func createInstallationCR(ctx context.Context, serverClient k8sclient.Client) error {
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	logrus.Infof("Looking for rhmi CR in %s namespace", namespace)

	installationList := &integreatlyv1alpha1.RHMIList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	err = serverClient.List(ctx, installationList, listOpts...)
	if err != nil {
		return fmt.Errorf("Could not get a list of rhmi CR: %w", err)
	}

	installation := &integreatlyv1alpha1.RHMI{}
	// Creates installation CR in case there is none
	if len(installationList.Items) == 0 {

		useClusterStorage, _ := os.LookupEnv("USE_CLUSTER_STORAGE")
		alertingEmailAddress, _ := os.LookupEnv("ALERTING_EMAIL_ADDRESS")

		logrus.Infof("Creating a %s rhmi CR with USC %s, as no CR rhmis were found in %s namespace", string(integreatlyv1alpha1.InstallationTypeManaged), useClusterStorage, namespace)

		installation = &integreatlyv1alpha1.RHMI{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DefaultInstallationName,
				Namespace: namespace,
			},
			Spec: integreatlyv1alpha1.RHMISpec{
				Type:                        string(integreatlyv1alpha1.InstallationTypeManaged),
				NamespacePrefix:             DefaultInstallationPrefix,
				SelfSignedCerts:             false,
				SMTPSecret:                  DefaultInstallationPrefix + "smtp",
				DeadMansSnitchSecret:        DefaultInstallationPrefix + "deadmanssnitch",
				PagerDutySecret:             DefaultInstallationPrefix + "pagerduty",
				UseClusterStorage:           useClusterStorage,
				AlertingEmailAddress:        alertingEmailAddress,
				OperatorsInProductNamespace: false, // e2e tests and Makefile need to be updated when default is changed
			},
		}

		err = serverClient.Create(ctx, installation)
		if err != nil {
			return fmt.Errorf("Could not create rhmi CR in %s namespace: %w", namespace, err)
		}
	} else if len(installationList.Items) == 1 {
		installation = &installationList.Items[0]
	} else {
		return fmt.Errorf("Too many rhmi resources found. Expecting 1, found %s rhmi resources in %s namespace", string(len(installationList.Items)), namespace)
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileInstallation{}

// ReconcileInstallation reconciles a Installation object
type ReconcileInstallation struct {
	// This client, initialized using mgr.client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            k8sclient.Client
	scheme            *runtime.Scheme
	restConfig        *rest.Config
	productsToInstall []string
	context           context.Context
	cancel            context.CancelFunc
	mgr               manager.Manager
	controller        controller.Controller
	customInformers   map[string]map[string]*cache.Informer
}

// Reconcile reads that state of the cluster for a Installation object and makes changes based on the state read
// and what is in the Installation.Spec
func (r *ReconcileInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	installInProgress := false
	installation := &integreatlyv1alpha1.RHMI{}
	err := r.client.Get(context.TODO(), request.NamespacedName, installation)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	retryRequeue := reconcile.Result{
		Requeue:      true,
		RequeueAfter: 10 * time.Second,
	}

	//context is cancelled on delete of an installation, so if context is cancelled but there is no deletion timestamp
	//the installation must be created after one was deleted, so recreate a context to use for the new installation.
	if r.context.Err() == context.Canceled && installation.DeletionTimestamp == nil {
		r.context, r.cancel = context.WithCancel(context.Background())
	}

	installType, err := TypeFactory(installation.Spec.Type, r.productsToInstall)
	if err != nil {
		return reconcile.Result{}, err
	}
	installationCfgMap := os.Getenv("INSTALLATION_CONFIG_MAP")
	if installationCfgMap == "" {
		installationCfgMap = installation.Spec.NamespacePrefix + DefaultInstallationConfigMapName
	}

	// gets the products from the install type to expose rhmi status metric
	stages := make([]integreatlyv1alpha1.RHMIStageStatus, 0)
	for _, stage := range installType.GetStages() {
		stages = append(stages, integreatlyv1alpha1.RHMIStageStatus{
			Name:     stage.Name,
			Phase:    "",
			Products: stage.Products,
		})
	}
	metrics.SetRHMIStatus(installation)

	configManager, err := config.NewManager(r.context, r.client, request.NamespacedName.Namespace, installationCfgMap, installation)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = resources.AddFinalizer(r.context, installation, r.client, deletionFinalizer)
	if err != nil {
		return reconcile.Result{}, err
	}

	if installation.Status.Stages == nil {
		installation.Status.Stages = map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{}
	}

	// either not checked, or rechecking preflight checks
	if installation.Status.PreflightStatus == integreatlyv1alpha1.PreflightInProgress ||
		installation.Status.PreflightStatus == integreatlyv1alpha1.PreflightFail {
		return r.preflightChecks(installation, installType, configManager)
	}

	// If the CR is being deleted, cancel the current context
	// and attempt to clean up the products with finalizers
	if installation.DeletionTimestamp != nil {
		// Cancel this context to kill all ongoing requests to the API
		// and use a new context to handle deletion logic
		r.cancel()

		// Set metrics status to unavailable
		metrics.RHMIStatusAvailable.Set(0)
		installation.Status.Stage = integreatlyv1alpha1.StageName("deletion")
		installation.Status.LastError = ""
		// updates rhmi status metric to deletion
		metrics.SetRHMIStatus(installation)
		_ = r.client.Status().Update(context.TODO(), installation)

		// Clean up the products which have finalizers associated to them
		merr := &multiErr{}
		for _, productFinalizer := range installation.Finalizers {
			if !strings.Contains(productFinalizer, "integreatly") {
				continue
			}
			productName := strings.Split(productFinalizer, ".")[1]
			product := installation.GetProductStatusObject(integreatlyv1alpha1.ProductName(productName))
			reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, installation, r.mgr)
			if err != nil {
				merr.Add(fmt.Errorf("Failed to build reconciler for product %s: %w", product.Name, err))
			}
			serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{})
			if err != nil {
				merr.Add(fmt.Errorf("Failed to create server client for %s: %w", product.Name, err))
			}
			phase, err := reconciler.Reconcile(context.TODO(), installation, product, serverClient)
			if err != nil {
				merr.Add(fmt.Errorf("Failed to reconcile product %s: %w", product.Name, err))
			}
			logrus.Infof("current phase for %s is: %s", product.Name, phase)
		}

		if len(merr.errors) == 0 && len(installation.Finalizers) == 1 && installation.Finalizers[0] == deletionFinalizer {
			// delete ConfigMap after all product finalizers finished
			if err := r.client.Delete(context.TODO(), &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: installationCfgMap, Namespace: request.NamespacedName.Namespace}}); err != nil && !k8serr.IsNotFound(err) {
				merr.Add(fmt.Errorf("failed to remove installation ConfigMap: %w", err))
				return retryRequeue, merr
			}

			if err = r.handleCROConfigDeletion(*installation); err != nil && !k8serr.IsNotFound(err) {
				merr.Add(fmt.Errorf("failed to remove Cloud Resource ConfigMap: %w", err))
				return retryRequeue, merr
			}

			err = resources.RemoveFinalizer(context.TODO(), installation, r.client, deletionFinalizer)
			if err != nil {
				merr.Add(fmt.Errorf("Failed to remove finalizer: %w", err))
				return retryRequeue, merr
			}
			logrus.Infof("uninstall completed")
			return reconcile.Result{}, nil
		}

		return retryRequeue, nil
	}

	// If no current or target version is set this is the first installation of rhmi.
	if installation.Status.Version == "" && installation.Status.ToVersion == "" {
		installation.Status.ToVersion = version.IntegreatlyVersion
		logrus.Infof("Setting installation.Status.ToVersion on initial install %s", version.IntegreatlyVersion)
		if err := r.client.Status().Update(r.context, installation); err != nil {
			return reconcile.Result{}, err
		}
	}

	// It's important to set the metric values at this point to account for the upgrade scenario. The ToVersion
	// is set on the CR when the install plan is approved, however, the operator pod is terminated shortly
	// after this point which may not be enough time for prometheus to scrape the metric.
	// needs to add check for stage complete to avoid setting the metric when installation is happening
	if string(installation.Status.Stage) == "complete" {
		metrics.SetRhmiVersions(string(installation.Status.Stage), installation.Status.Version, installation.Status.ToVersion, installation.CreationTimestamp.Unix())
	}

	// Reconcile the webhooks
	if err := webhooks.Config.Reconcile(r.context, r.client, installation); err != nil {
		return reconcile.Result{}, err
	}

	for _, stage := range installType.GetStages() {
		var err error
		var stagePhase integreatlyv1alpha1.StatusPhase
		if stage.Name == integreatlyv1alpha1.BootstrapStage {
			stagePhase, err = r.bootstrapStage(installation, configManager)
		} else {
			stagePhase, err = r.processStage(installation, &stage, configManager)
		}

		if installation.Status.Stages == nil {
			installation.Status.Stages = make(map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus)
		}
		installation.Status.Stages[stage.Name] = integreatlyv1alpha1.RHMIStageStatus{
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
		if stagePhase != integreatlyv1alpha1.PhaseCompleted {
			installInProgress = true
			break
		}
	}

	// UPDATE STATUS
	// updates rhmi status metric according to the status of the products
	if installInProgress {
		metrics.SetRHMIStatus(installation)
	}

	err = r.client.Status().Update(r.context, installation)
	if err != nil {
		// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
		if k8serr.IsConflict(err) {
			// If there is a conflict, requeue the resource and retry Update
			logrus.Info("Error updating Installation resource status. Requeue and retry.", err)
			return retryRequeue, nil
		}

		logrus.Error(err, "error reconciling installation installation")
		if installInProgress {
			return retryRequeue, err
		}
		return reconcile.Result{}, err
	}

	// UPDATE OBJECT
	err = r.client.Update(r.context, installation)
	if err != nil {
		// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
		if k8serr.IsConflict(err) {
			// If there is a conflict, requeue the resource and retry Update
			logrus.Info("Error updating Installation resource. Requeue and retry.", err)
			return retryRequeue, nil
		}

		logrus.Error(err, "error reconciling installation installation")
		if installInProgress {
			return retryRequeue, err
		}
		return reconcile.Result{}, err
	}
	if installInProgress {
		return retryRequeue, nil
	}
	installation.Status.Stage = integreatlyv1alpha1.StageName("complete")
	// updates rhmi status metric to complete
	metrics.SetRHMIStatus(installation)

	if (isFirstInstallReconcile(installation) || isUpgradeReconcile(installation)) && allProductsReconciled(installation) {
		installation.Status.Version = installation.Status.ToVersion
		installation.Status.ToVersion = ""
		metrics.SetRhmiVersions(string(installation.Status.Stage), installation.Status.Version, installation.Status.ToVersion, installation.CreationTimestamp.Unix())
	}

	_ = r.client.Status().Update(r.context, installation)

	metrics.RHMIStatusAvailable.Set(1)
	logrus.Infof("installation completed succesfully")
	return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, nil
}

func isFirstInstallReconcile(installation *integreatlyv1alpha1.RHMI) bool {
	return installation.Status.Version == ""
}

func isUpgradeReconcile(installation *integreatlyv1alpha1.RHMI) bool {
	status := installation.Status
	if status.ToVersion != "" {
		return true
	}
	return false
}

func allProductsReconciled(installation *integreatlyv1alpha1.RHMI) bool {
	stages := installation.Status.Stages

	return (verifyClusterRHSSOVersions(stages) &&
		verifyCloudResourceVersions(stages) &&
		verifyAMQOnlineVersions(stages) &&
		verifyApicuritoVersions(stages) &&
		verifyCodeReadyWorkspacesVersions(stages) &&
		verifyFuseOnOpenshiftVersions(stages) &&
		verifyMonitoringVersions(stages) &&
		verify3ScaleVersions(stages) &&
		verifyFuseOnlineVersions(stages) &&
		verifyDataSyncVersions(stages) &&
		verifyRHSSOUserVersions(stages) &&
		verifyUPSVersions(stages) &&
		verifySolutionExplorerVersions(stages))
}

func verifySolutionExplorerVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.SolutionExplorerStage].Products[integreatlyv1alpha1.ProductSolutionExplorer]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionSolutionExplorer)
	installedOpVersion := string(versions.OperatorVersion)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("Solution Explorer Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	return true
}

func verifyClusterRHSSOVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.AuthenticationStage].Products[integreatlyv1alpha1.ProductRHSSO]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionRHSSO)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionRHSSO)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("Cluster RHSSO Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("Cluster RHSSO Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyMonitoringVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.MonitoringStage].Products[integreatlyv1alpha1.ProductMonitoring]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionMonitoring)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionMonitoring)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("Monitoring Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("Monitoring Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyCloudResourceVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.CloudResourcesStage].Products[integreatlyv1alpha1.ProductCloudResources]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionCloudResources)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionCloudResources)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("Cloud Resource Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("Cloud Resource Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyApicuritoVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductApicurito]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionApicurito)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionApicurito)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("Apicurito Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("Apicurito Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyRHSSOUserVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductRHSSOUser]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionRHSSOUser)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionRHSSOUser)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("RHSSOUser Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("RHSSOUser Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyUPSVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductUps]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionUPS)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionUps)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("UPS Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("UPS Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verify3ScaleVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.Product3Scale]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersion3Scale)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.Version3Scale)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("3Scale Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("3Scale Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyDataSyncVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductDataSync]
	expectedProductVersion := string(integreatlyv1alpha1.VersionDataSync)
	installedProductVersion := string(versions.Version)

	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("Data Sync Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyFuseOnlineVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductFuse]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionFuse)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionFuseOnline)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("Fuse Online Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("Fuse Online Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyCodeReadyWorkspacesVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductCodeReadyWorkspaces]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionCodeReadyWorkspaces)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionCodeReadyWorkspaces)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("CodeReady Workspaces Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("CodeReady Workspaces Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyAMQOnlineVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductAMQOnline]
	expectedOpVersion := string(integreatlyv1alpha1.OperatorVersionAMQOnline)
	installedOpVersion := string(versions.OperatorVersion)
	expectedProductVersion := string(integreatlyv1alpha1.VersionAMQOnline)
	installedProductVersion := string(versions.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("AMQOnline Operator Version is not as expected. Expected %s, Actual %s", expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("AMQOnline Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func verifyFuseOnOpenshiftVersions(stages map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus) bool {
	versions := stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductFuseOnOpenshift]
	expectedProductVersion := string(integreatlyv1alpha1.VersionFuseOnOpenshift)
	installedProductVersion := string(versions.Version)

	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("FuseOnOpenShift Version is not as expected. Expected %s, Actual %s", expectedProductVersion, installedProductVersion)
		return false
	}
	return true
}

func (r *ReconcileInstallation) preflightChecks(installation *integreatlyv1alpha1.RHMI, installationType *Type, configManager *config.Manager) (reconcile.Result, error) {
	logrus.Info("Running preflight checks..")
	installation.Status.Stage = integreatlyv1alpha1.StageName("Preflight Checks")
	result := reconcile.Result{
		Requeue:      true,
		RequeueAfter: 10 * time.Second,
	}

	eventRecorder := r.mgr.GetEventRecorderFor("Preflight Checks")

	if strings.ToLower(installation.Spec.UseClusterStorage) != "true" && strings.ToLower(installation.Spec.UseClusterStorage) != "false" {
		installation.Status.PreflightStatus = integreatlyv1alpha1.PreflightFail
		installation.Status.PreflightMessage = "Spec.useClusterStorage must be set to either 'true' or 'false' to continue"
		_ = r.client.Status().Update(r.context, installation)
		logrus.Infof("preflight checks failed on useClusterStorage value")
		return result, nil
	}

	if installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManaged) {
		requiredSecrets := []string{installation.Spec.SMTPSecret, installation.Spec.PagerDutySecret, installation.Spec.DeadMansSnitchSecret}

		for _, secretName := range requiredSecrets {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: installation.Namespace,
				},
			}
			if exists, err := resources.Exists(r.context, r.client, secret); err != nil {
				return reconcile.Result{}, err
			} else if !exists {
				preflightMessage := fmt.Sprintf("Could not find %s secret in %s namespace", secret.Name, installation.Namespace)
				logrus.Info(preflightMessage)
				eventRecorder.Event(installation, "Warning", integreatlyv1alpha1.EventProcessingError, preflightMessage)

				installation.Status.PreflightStatus = integreatlyv1alpha1.PreflightFail
				installation.Status.PreflightMessage = preflightMessage
				_ = r.client.Status().Update(r.context, installation)

				return reconcile.Result{}, err
			}
			logrus.Infof("found required secret: %s", secretName)
			eventRecorder.Eventf(installation, "Normal", integreatlyv1alpha1.EventPreflightCheckPassed,
				"found required secret: %s", secretName)
		}
	}

	logrus.Infof("getting namespaces")
	namespaces := &corev1.NamespaceList{}
	err := r.client.List(r.context, namespaces)
	if err != nil {
		// could not list namespaces, keep trying
		logrus.Infof("error listing namespaces, will retry")
		return result, err
	}

	for _, ns := range namespaces.Items {
		products, err := r.checkNamespaceForProducts(ns, installation, installationType, configManager)
		if err != nil {
			// error searching for existing products, keep trying
			logrus.Infof("error looking for existing deployments, will retry")
			return result, err
		}
		if len(products) != 0 {
			//found one or more conflicting products
			installation.Status.PreflightStatus = integreatlyv1alpha1.PreflightFail
			installation.Status.PreflightMessage = "found conflicting packages: " + strings.Join(products, ", ") + ", in namespace: " + ns.GetName()
			logrus.Infof("found conflicting packages: " + strings.Join(products, ", ") + ", in namespace: " + ns.GetName())
			_ = r.client.Status().Update(r.context, installation)
			return result, err
		}
	}

	installation.Status.PreflightStatus = integreatlyv1alpha1.PreflightSuccess
	installation.Status.PreflightMessage = "preflight checks passed"
	err = r.client.Status().Update(r.context, installation)
	if err != nil {
		logrus.Infof("error updating status: %s", err.Error())
	}
	return result, nil
}

func (r *ReconcileInstallation) checkNamespaceForProducts(ns corev1.Namespace, installation *integreatlyv1alpha1.RHMI, installationType *Type, configManager *config.Manager) ([]string, error) {
	foundProducts := []string{}
	if strings.HasPrefix(ns.Name, "openshift-") {
		return foundProducts, nil
	}
	if strings.HasPrefix(ns.Name, "kube-") {
		return foundProducts, nil
	}
	// new client to avoid caching issues
	serverClient, _ := k8sclient.New(r.restConfig, k8sclient.Options{})
	for _, stage := range installationType.Stages {
		for _, product := range stage.Products {
			reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, installation, r.mgr)
			if err != nil {
				return foundProducts, err
			}
			search := reconciler.GetPreflightObject(ns.Name)
			if search == nil {
				continue
			}
			exists, err := resources.Exists(r.context, serverClient, search)
			if err != nil {
				return foundProducts, err
			} else if exists {
				logrus.Infof("found conflicting product: %s", product.Name)
				foundProducts = append(foundProducts, string(product.Name))
			}
		}
	}
	return foundProducts, nil
}

func (r *ReconcileInstallation) bootstrapStage(installation *integreatlyv1alpha1.RHMI, configManager config.ConfigReadWriter) (integreatlyv1alpha1.StatusPhase, error) {
	installation.Status.Stage = integreatlyv1alpha1.BootstrapStage
	mpm := marketplace.NewManager()

	reconciler, err := NewBootstrapReconciler(configManager, installation, mpm, r.mgr.GetEventRecorderFor(string(integreatlyv1alpha1.BootstrapStage)))
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to build a reconciler for Bootstrap: %w", err)
	}
	serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create server client: %w", err)
	}
	phase, err := reconciler.Reconcile(r.context, installation, serverClient)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Bootstrap stage reconcile failed: %w", err)
	}

	return phase, nil
}

func (r *ReconcileInstallation) processStage(installation *integreatlyv1alpha1.RHMI, stage *Stage, configManager config.ConfigReadWriter) (integreatlyv1alpha1.StatusPhase, error) {
	incompleteStage := false
	var mErr error
	productsAux := make(map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus)
	installation.Status.Stage = stage.Name

	for _, product := range stage.Products {

		reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, installation, r.mgr)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to build a reconciler for %s: %w", product.Name, err)
		}
		serverClient, err := k8sclient.New(r.restConfig, k8sclient.Options{})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create server client: %w", err)
		}
		product.Status, err = reconciler.Reconcile(r.context, installation, &product, serverClient)
		if err != nil {
			if mErr == nil {
				mErr = &multiErr{}
			}
			mErr.(*multiErr).Add(fmt.Errorf("failed installation of %s: %w", product.Name, err))
		}

		// Verify that watches for this product CRDs have been created
		config, err := configManager.ReadProduct(product.Name)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to read product config for %s: %v", string(product.Name), err)
		}
		if product.Status == integreatlyv1alpha1.PhaseCompleted {
			for _, crd := range config.GetWatchableCRDs() {
				namespace := config.GetNamespace()
				gvk := crd.GetObjectKind().GroupVersionKind().String()
				if r.customInformers[gvk] == nil {
					r.customInformers[gvk] = make(map[string]*cache.Informer)
				}
				if r.customInformers[gvk][config.GetNamespace()] == nil {
					err = r.addCustomInformer(crd, namespace)
					if err != nil {
						return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to create a %s CRD watch for %s: %v", gvk, string(product.Name), err)
					}
				} else if !(*r.customInformers[gvk][config.GetNamespace()]).HasSynced() {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("A %s CRD Informer for %s has not synced", gvk, string(product.Name))
				}
			}
		}

		//found an incomplete product
		if product.Status != integreatlyv1alpha1.PhaseCompleted {
			incompleteStage = true
		}
		productsAux[product.Name] = product
		*stage = Stage{Name: stage.Name, Products: productsAux}
	}

	//some products in this stage have not installed successfully yet
	if incompleteStage {
		return integreatlyv1alpha1.PhaseInProgress, mErr
	}
	return integreatlyv1alpha1.PhaseCompleted, mErr
}

// handle the deletion of CRO config map
func (r *ReconcileInstallation) handleCROConfigDeletion(rhmi integreatlyv1alpha1.RHMI) error {
	// get cloud resource config map
	croConf := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: rhmi.Namespace, Name: DefaultCloudResourceConfigName}, croConf)
	if err != nil {
		return err
	}

	// remove cloud resource config deletion finalizer if it exists
	if resources.Contains(croConf.Finalizers, deletionFinalizer) {
		croConf.SetFinalizers(resources.Remove(croConf.Finalizers, deletionFinalizer))

		if err := r.client.Update(context.TODO(), croConf); err != nil {
			return fmt.Errorf("error occurred trying to update cro config map %w", err)
		}
	}

	// remove cloud resource config map
	err = r.client.Delete(context.TODO(), croConf)
	if err != nil && !k8serr.IsNotFound(err) {
		return fmt.Errorf("error occurred trying to delete cro config map, %w", err)
	}

	return nil
}

func (r *ReconcileInstallation) addCustomInformer(crd runtime.Object, namespace string) error {
	gvk := crd.GetObjectKind().GroupVersionKind().String()
	mapper, err := apiutil.NewDiscoveryRESTMapper(r.restConfig)
	if err != nil {
		return fmt.Errorf("Failed to get API Group-Resources: %v", err)
	}
	cache, err := cache.New(r.restConfig, cache.Options{Namespace: namespace, Scheme: r.mgr.GetScheme(), Mapper: mapper})
	if err != nil {
		return fmt.Errorf("Failed to create informer cache in %s namespace: %v", namespace, err)
	}
	informer, err := cache.GetInformerForKind(crd.GetObjectKind().GroupVersionKind())
	if err != nil {
		return fmt.Errorf("Failed to create informer for %v: %v", crd, err)
	}
	err = r.controller.Watch(&source.Informer{Informer: informer}, &EnqueueIntegreatlyOwner{})
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

	logrus.Infof("Cache synced. A %s watch in %s namespace successfully initialized.", gvk, namespace)
	return nil
}

type multiErr struct {
	errors []string
}

func (mer *multiErr) Error() string {
	return "product installation errors : " + strings.Join(mer.errors, ":")
}

//Add an error to the collection
func (mer *multiErr) Add(err error) {
	if mer.errors == nil {
		mer.errors = []string{}
	}
	mer.errors = append(mer.errors, err.Error())
}
