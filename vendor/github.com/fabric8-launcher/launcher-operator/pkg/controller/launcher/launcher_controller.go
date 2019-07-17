package launcher

import (
	"context"
	launcherApi "fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"
	"fabric8-launcher/launcher-operator/pkg/helper"
	"fmt"
	"os"
	"reflect"

	"github.com/integr8ly/operator-sdk-openshift-utils/pkg/api/template"
	appsv1 "github.com/openshift/api/apps/v1"
	appsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_launcher")

var templateName = "fabric8-launcher"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Launcher Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLauncher{client: mgr.GetClient(), scheme: mgr.GetScheme(), config: mgr.GetConfig()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("launcher-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Launcher
	err = c.Watch(&source.Kind{Type: &launcherApi.Launcher{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Launcher
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &launcherApi.Launcher{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileLauncher{}

// ReconcileLauncher reconciles a Launcher object
type ReconcileLauncher struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config *rest.Config
}

// Reconcile reads that state of the cluster for a Launcher object and makes changes based on the state read
// and what is in the Launcher.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileLauncher) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconciling Launcher\n", "Namespace", request.Namespace, "Name", request.Name)

	// Fetch the Launcher instance
	instance := &launcherApi.Launcher{}
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

	var tpl *template.Tmpl
	tpl, err = r.loadLauncherTemplate()

	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Launcher template has been loaded")

	err = r.processLauncherTemplate(tpl, instance.Namespace)

	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Launcher template has been processed", "Namespace", instance.Namespace)

	resourceObjects := tpl.GetObjects(r.filterResourcesObjects)
	configMap := r.findObjectByKindAndName(resourceObjects, "launcher", "ConfigMap").(*corev1.ConfigMap)

	if configMap == nil {
		return reconcile.Result{}, fmt.Errorf("ConfigMap not found in the launcher template")
	}

	url := r.config.Host
	data := configMap.Data
	data["launcher.frontend.targetenvironment.skip"] = "true"
	data["launcher.missioncontrol.openshift.api.url"] = url
	if &instance.Spec.OpenShift != nil && instance.Spec.OpenShift.ConsoleURL != "" {
		data["launcher.missioncontrol.openshift.console.url"] = instance.Spec.OpenShift.ConsoleURL
	}
	data["launcher.keycloak.url"] = ""
	data["launcher.keycloak.realm"] = ""

	if &instance.Spec.OAuth != nil && instance.Spec.OAuth.Enabled {
		if &instance.Spec.OpenShift == nil || instance.Spec.OpenShift.ConsoleURL == "" {
			return reconcile.Result{}, fmt.Errorf("OpenShift ConsoleUrl must be defined to use OAuth")
		}
		data["launcher.oauth.openshift.url"] = instance.Spec.OpenShift.ConsoleURL + "/oauth/authorize"
	} else if &instance.Spec.GitHub.Token != nil {
		token, err := r.getSensitiveValue(instance.Namespace, instance.Spec.GitHub.Token)

		if err != nil {
			return reconcile.Result{}, err
		}

		data["launcher.missioncontrol.github.token"] = token
	}

	isUpdated, err := r.updateConfigIfChanged(instance, configMap)

	if err != nil {
		return reconcile.Result{}, err
	}

	if isUpdated {
		log.Info("The config has been updated, a new deployment should be triggered")
		for _, deploymentConfigName := range []string{"launcher-backend", "launcher-creator-backend", "launcher-frontend"} {
			err = r.deployLatest(instance.Namespace, deploymentConfigName)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	for _, o := range resourceObjects {
		err = r.createLauncherResource(instance, o)
		if err != nil {
			return reconcile.Result{}, err
		}

	}
	return reconcile.Result{}, nil
}

func (r *ReconcileLauncher) filterResourcesObjects(obj *runtime.Object) error {
	if (*obj).(v1.Object).GetName() == "configmapcontroller" {
		return fmt.Errorf("ignoring confimapcontroller")
	}
	return nil
}

func (r *ReconcileLauncher) deployLatest(namespace string, deploymentConfigName string) error {
	appsClient, err := appsv1client.NewForConfig(r.config)
	if err != nil {
		return fmt.Errorf("error while creating appsv1client: %s", err)
	}
	request := &appsv1.DeploymentRequest{
		Name:   deploymentConfigName,
		Latest: true,
		Force:  true,
	}
	log.Info("Trigger deployment", "namespace", namespace, "deploymentConfigName", deploymentConfigName)
	_, err = appsClient.DeploymentConfigs(namespace).Instantiate(deploymentConfigName, request)
	if err != nil {
		return fmt.Errorf("error while deploying '%s': %s", deploymentConfigName, err)
	}
	return nil
}

func (r *ReconcileLauncher) loadLauncherTemplate() (*template.Tmpl, error) {
	templatePath := os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "./templates"
	}
	templateHelper := helper.NewOpenshiftTemplateHelper()

	return templateHelper.Load(r.config, templatePath, templateName)
}

func (r *ReconcileLauncher) processLauncherTemplate(template *template.Tmpl, namespace string) error {
	return template.Process(nil, namespace)
}

func (r *ReconcileLauncher) findObjectByKindAndName(objects []runtime.Object, name string, kind string) runtime.Object {
	for _, o := range objects {
		if o.GetObjectKind().GroupVersionKind().Kind == kind && o.(v1.Object).GetName() == name {
			return o
		}
	}
	return nil
}

func (r *ReconcileLauncher) updateConfigIfChanged(cr *launcherApi.Launcher, configMapResource runtime.Object) (bool, error) {
	// Try to find the template, it may already exist
	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      configMapResource.(v1.Object).GetName(),
	}
	configMapResource.(v1.Object).SetNamespace(cr.Namespace)
	err := controllerutil.SetControllerReference(cr, configMapResource.(v1.Object), r.scheme)
	if err != nil {
		return false, fmt.Errorf("error setting the custom resource as owner: %s", err)
	}
	retrievedResource := configMapResource.DeepCopyObject()
	err = r.client.Get(context.TODO(), selector, retrievedResource)

	data := configMapResource.(*corev1.ConfigMap).Data
	if err == nil && !reflect.DeepEqual(data, retrievedResource.(*corev1.ConfigMap).Data) {
		log.Info("Launcher data has been updated, updating ConfigMap",
			"name", configMapResource.(v1.Object).GetName(),
			"kind", configMapResource.GetObjectKind().GroupVersionKind().Kind)
		err = r.client.Update(context.TODO(), configMapResource)
		if err != nil {
			return false, fmt.Errorf("error updating ConfigMap: %s", err)
		}
		return true, nil
	}

	return false, nil
}

func (r *ReconcileLauncher) createLauncherResource(cr *launcherApi.Launcher, resource runtime.Object) error {

	resource.(v1.Object).SetNamespace(cr.Namespace)
	// Set the CR as the owner of this resource so that when
	// the CR is deleted this resource also gets removed
	err := controllerutil.SetControllerReference(cr, resource.(v1.Object), r.scheme)

	if err != nil {
		return fmt.Errorf("error setting the custom resource as owner: %s", err)
	}

	// Try to find the template, it may already exist
	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      resource.(v1.Object).GetName(),
	}
	retrievedResource := resource.DeepCopyObject()
	err = r.client.Get(context.TODO(), selector, retrievedResource)

	if err == nil {
		log.Info("Resource exists, do nothing",
			"name", resource.(v1.Object).GetName(),
			"kind", resource.GetObjectKind().GroupVersionKind().Kind)
		return nil
	}

	// Resource does not exist or something went wrong
	if errors.IsNotFound(err) {
		log.Info("Resource is missing. Creating it.",
			"name", resource.(v1.Object).GetName(),
			"kind", resource.GetObjectKind().GroupVersionKind().Kind)
		err = r.client.Create(context.TODO(), resource)
		if err != nil {
			return fmt.Errorf("error creating resource: %s", err)
		}
	} else {
		return fmt.Errorf("error reading resource '%s': %s", resource.(v1.Object).GetName(), err)
	}

	return nil
}

func (r *ReconcileLauncher) getSensitiveValue(namespace string, sensitiveValue launcherApi.SensitiveValue) (string, error) {
	if sensitiveValue.ValueFrom.SecretKeyRef != (launcherApi.SecretKeyRef{}) {
		secret := &corev1.Secret{}
		key := sensitiveValue.ValueFrom.SecretKeyRef.Key
		name := sensitiveValue.ValueFrom.SecretKeyRef.Name
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, secret)
		if err != nil {
			return "", err
		}
		value := secret.Data[key]
		if value == nil {
			return "", fmt.Errorf("key '%s' not found in secret '%s'", key, name)
		}
		return string(value), nil
	}
	if sensitiveValue.ValueFrom.Text != "" {
		return sensitiveValue.ValueFrom.Text, nil
	}
	return "", fmt.Errorf("invalid sensitive value definition")
}
