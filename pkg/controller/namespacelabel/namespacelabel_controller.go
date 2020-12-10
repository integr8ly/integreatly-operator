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

package namespacelabel

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log           = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "namespacelabel_controller"})
	configMapName = "cloud-resources-aws-strategies"
)

//  patchStringValue specifies a patch operation for a string.
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type network struct {
	Production struct {
		CreateStrategy struct {
			CidrBlock string `json:"CidrBlock"`
		} `json:"createStrategy"`
	} `json:"production"`
}

// Add creates a new namespacelabel Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconcile, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconcile)
}

var (
	// Map that associates the labels in the namespace to actions to perform
	// - Keys are labels that might be set to the namespace object
	// - Values are functions that receive the value of the label, the reconcile request,
	//   and the reconciler instance.
	namespaceLabelBasedActions = map[string]func(string, reconcile.Request, *ReconcileNamespaceLabel) error{
		// Uninstall RHMI
		"api.openshift.com/addon-rhmi-operator-delete": Uninstall,
		// Uninstall MAO
		"api.openshift.com/addon-managed-api-service-delete": Uninstall,
		// Update CIDR value
		"cidr": CheckCidrValueAndUpdate,
	}
)

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {

	ctx, cancel := context.WithCancel(context.Background())
	watchNS, err := k8sutil.GetWatchNamespace()
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "could not get watch namespace from namespacelabel controller")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	operatorNs := namespacePrefix + "operator"

	return &ReconcileNamespaceLabel{
		mgr:               mgr,
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		operatorNamespace: operatorNs,
		context:           ctx,
		cancel:            cancel,
		log:               log,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("namespace-label-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNamespace implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNamespaceLabel{}

// ReconcileNamespaceLabel reconciles a namespace label object
type ReconcileNamespaceLabel struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            k8sclient.Client
	scheme            *runtime.Scheme
	restConfig        *rest.Config
	mgr               manager.Manager
	operatorNamespace string
	controller        controller.Controller
	context           context.Context
	cancel            context.CancelFunc
	log               l.Logger
}

// Reconcile : The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespaceLabel) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	if request.NamespacedName.Name == r.operatorNamespace {
		r.log.Info("Reconciling namespace labels")

		ns, err := GetNS(r.context, r.operatorNamespace, r.client)
		if err != nil {
			r.log.Error("could not retrieve %s namespace:", err)
		}
		err = r.CheckLabel(ns, request)

		if err != nil {
			return reconcile.Result{}, err
		}

		r.log.Info("Reconciling namespace labels completed")
	}
	return reconcile.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil
}

// GetNS gets the specified corev1.Namespace from the k8s API server
func GetNS(ctx context.Context, namespace string, client k8sclient.Client) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err == nil {
		// workaround for https://github.com/kubernetes/client-go/issues/541
		ns.TypeMeta = metav1.TypeMeta{Kind: "Namespace", APIVersion: metav1.SchemeGroupVersion.Version}
	}
	return ns, err
}

// CheckLabel Checks namespace for labels and
func (r *ReconcileNamespaceLabel) CheckLabel(o metav1.Object, request reconcile.Request) error {
	for k, v := range o.GetLabels() {
		action, ok := namespaceLabelBasedActions[k]
		if !ok {
			continue
		}

		if err := action(v, request, r); err != nil {
			return err
		}
	}

	return nil
}

// Uninstall deletes rhmi cr when uninstall label is set
func Uninstall(v string, request reconcile.Request, r *ReconcileNamespaceLabel) error {
	if v != "true" {
		return nil
	}

	r.log.Info("Uninstall label has been set")

	rhmiCr, err := resources.GetRhmiCr(r.client, context.TODO(), request.NamespacedName.Namespace, log)
	if err != nil {
		// Error reading the object - requeue the request.
		return err
	}
	if rhmiCr == nil {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return nil
	}

	if rhmiCr.DeletionTimestamp == nil {
		r.log.Info("Deleting RHMI CR")
		err := r.client.Delete(r.context, rhmiCr)
		if err != nil {
			r.log.Error("failed to delete RHMI CR", err)
		}
	}
	return nil
}

// CheckCidrValueAndUpdate Checks cidr value and updates it in the configmap if the config map value is ""
func CheckCidrValueAndUpdate(value string, request reconcile.Request, r *ReconcileNamespaceLabel) error {
	r.log.Infof("Cidr value : passed in as a namespace label", l.Fields{"value": value})
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: request.NamespacedName.Name,
		},
	}

	err := r.client.Get(context.TODO(), k8sclient.ObjectKey{Name: configMapName, Namespace: request.NamespacedName.Name}, cfgMap)
	if err != nil {
		return err
	}
	data := []byte(cfgMap.Data["_network"])

	var cfgMapData network

	// Unmarshal or Decode the JSON to the interface.
	err = json.Unmarshal([]byte(data), &cfgMapData)
	if err != nil {
		r.log.Error("Failed to unmarshal cfgMapData", err)
	}

	cidr := cfgMapData.Production.CreateStrategy.CidrBlock

	if cidr != "" {
		r.log.Infof("Cidr value is already set, not updating", l.Fields{"value": cidr})
		return nil
	}

	// replace - character from label with / so that the cidr value is set correctly.
	// / is not a valid character in namespace label values.
	newCidr := strings.Replace(value, "-", "/", -1)
	r.log.Infof("No cidr has been set in configmap yet, setting cidr from namespace label", l.Fields{"newCidr": newCidr})

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: ""}).ClientConfig()
	if err != nil {
		return err
	}

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	cfgMapData.Production.CreateStrategy.CidrBlock = newCidr
	dataValue, err := json.Marshal(cfgMapData)

	if err != nil {
		return err
	}

	payload := []patchStringValue{{
		Op:    "add",
		Path:  "/data/_network",
		Value: string(dataValue),
	}}

	payloadBytes, _ := json.Marshal(payload)
	_, err = clientset.
		CoreV1().
		ConfigMaps(request.NamespacedName.Name).
		Patch(configMapName, types.JSONPatchType, payloadBytes)

	if err != nil {
		return err
	}
	return nil
}
