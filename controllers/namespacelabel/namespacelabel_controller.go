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

package controllers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	deletionRHOAM = "managed-api-service"
	deletionRHMI  = "rhmi"
)

var (
	log           = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "namespacelabel_controller"})
	configMapName = "cloud-resources-aws-strategies"
	//configMap name derived from the addons metadata id
	deletionConfigMap = deletionRHOAM
)

//  patchStringValue specifies a patch operation for a string.
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type TierCreateStrategy struct {
	CreateStrategy struct {
		CidrBlock string `json:"CidrBlock"`
	} `json:"createStrategy"`
}

var (
	// Map that associates the labels in the namespace to actions to perform
	// - Keys are labels that might be set to the namespace object
	// - Values are functions that receive the value of the label, the reconcile request,
	//   and the reconciler instance.
	namespaceLabelBasedActions = map[string]func(string, ctrl.Request, *NamespaceLabelReconciler) error{
		// Uninstall RHMI
		"api.openshift.com/addon-rhmi-operator-delete": Uninstall,
		// Uninstall MAO
		"api.openshift.com/addon-managed-api-service-delete": Uninstall,
		// Update CIDR value
		"cidr": CheckCidrValueAndUpdate,
	}

	// Map that associates the labels on the configMap to actions to perform
	// - Keys are labels that might be set to the configMap object
	// - Values are functions that receive the value of the label, the reconcile request,
	//   and the reconciler instance.
	configMapLabelBasedActions = map[string]func(string, reconcile.Request, *NamespaceLabelReconciler) error{
		// Uninstall RHMI
		"api.openshift.com/addon-rhmi-operator-delete": Uninstall,
		// Uninstall MAO
		"api.openshift.com/addon-managed-api-service-delete": Uninstall,
	}
)

func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watches exclusively the operator namespace
		For(&corev1.Namespace{}, builder.WithPredicates(namePredicate(r.operatorNamespace))).
		// Watches ConfigMaps and enqueues requests to their namespace.
		// Only watches ConfigMaps with the addon name and in the operator namespace
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &EnqueueNamespaceFromObject{}, builder.WithPredicates(
			predicate.And(
				namespacePredicate(r.operatorNamespace),
				predicate.Or(
					namePredicate(deletionRHOAM),
					namePredicate(deletionRHMI),
				),
			),
		)).
		Complete(r)
}

func New(mgr manager.Manager) *NamespaceLabelReconciler {
	watchNS, err := resources.GetWatchNamespace()
	if err != nil {
		panic("could not get watch namespace from namespacelabel controller")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	operatorNs := namespacePrefix + "operator"

	return &NamespaceLabelReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		operatorNamespace: operatorNs,
		log:               log,
	}
}

// NamespaceLabelReconciler reconciles a namespace label object
type NamespaceLabelReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	k8sclient.Client
	Scheme *runtime.Scheme

	operatorNamespace string
	controller        controller.Controller
	log               l.Logger
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;get;watch;update

// Reconcile : The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NamespaceLabelReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()

	if request.NamespacedName.Name == r.operatorNamespace {
		r.log.Info("Reconciling namespace labels")

		ns, err := GetNS(ctx, r.operatorNamespace, r.Client)
		if err != nil {
			r.log.Error("could not retrieve %s namespace:", err)
		}

		rhmiCr, err := resources.GetRhmiCr(r.Client, ctx, request.NamespacedName.Namespace, log)
		if err != nil || rhmiCr == nil {
			return reconcile.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil
		}

		if rhmiCr.Spec.Type == "managed" {
			deletionConfigMap = deletionRHMI
		}
		err = r.CheckConfigMap(ns, request, deletionConfigMap)
		if err != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil
		}

		err = r.CheckLabel(ns, request)

		if err != nil {
			return ctrl.Result{}, err
		}

		r.log.Info("Reconciling namespace labels completed")
	}
	return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil
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
func (r *NamespaceLabelReconciler) CheckLabel(o metav1.Object, request ctrl.Request) error {
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

// CheckConfigMap Checks configMap for labels determines what action to use
func (r *NamespaceLabelReconciler) CheckConfigMap(o metav1.Object, request ctrl.Request, deletionConfigMapName string) error {
	configMap := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: request.NamespacedName.Name, Name: deletionConfigMapName}, configMap)
	if err != nil {
		return err
	}
	for k, v := range configMap.GetLabels() {
		action, ok := configMapLabelBasedActions[k]
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
func Uninstall(v string, request ctrl.Request, r *NamespaceLabelReconciler) error {
	if v != "true" {
		return nil
	}

	r.log.Info("Uninstall label has been set")

	rhmiCr, err := resources.GetRhmiCr(r.Client, context.TODO(), request.NamespacedName.Namespace, log)
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
		err := r.Delete(context.TODO(), rhmiCr)
		if err != nil {
			r.log.Error("failed to delete RHMI CR", err)
		}
	}
	return nil
}

// CheckCidrValueAndUpdate Checks cidr value and updates it in the configmap if the config map value is ""
func CheckCidrValueAndUpdate(value string, request ctrl.Request, r *NamespaceLabelReconciler) error {
	r.log.Infof("Cidr value : passed in as a namespace label", l.Fields{"value": value})
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: request.NamespacedName.Name,
		},
	}

	err := r.Get(context.TODO(), k8sclient.ObjectKey{Name: configMapName, Namespace: request.NamespacedName.Name}, cfgMap)
	if err != nil {
		return err
	}
	data := []byte(cfgMap.Data["_network"])

	var cfgMapData map[string]*TierCreateStrategy

	// Unmarshal or Decode the JSON to the interface.
	err = json.Unmarshal([]byte(data), &cfgMapData)
	if err != nil {
		r.log.Error("Failed to unmarshal cfgMapData", err)
	}

	if cfgMapData == nil || cfgMapData["production"] == nil {
		return nil
	}

	cidr := cfgMapData["production"].CreateStrategy.CidrBlock

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

	cfgMapData["production"].CreateStrategy.CidrBlock = newCidr

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
		Patch(context.TODO(), configMapName, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})

	if err != nil {
		return err
	}
	return nil
}

// namespacePredicate is a reusable predicate to watch only resources on a given
// namespace
func namespacePredicate(namespace string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(m metav1.Object, _ runtime.Object) bool {
		return m.GetNamespace() == namespace
	})
}

// namePredicate is a reusable predicate to watch only resources on a given
// name
func namePredicate(name string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(m metav1.Object, _ runtime.Object) bool {
		return m.GetName() == name
	})
}
