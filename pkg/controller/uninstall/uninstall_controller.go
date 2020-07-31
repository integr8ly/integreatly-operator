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

package uninstall

import (
	"context"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_uninstall")

// Add creates a new Uninstall Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconcile, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconcile)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	ctx, cancel := context.WithCancel(context.Background())
	operatorNs := "redhat-rhmi-operator"

	return &ReconcileUninstall{
		mgr:               mgr,
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		operatorNamespace: operatorNs,
		context:           ctx,
		cancel:            cancel,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("uninstall-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Uninstall
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileUninstall implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileUninstall{}

// ReconcileUninstall reconciles a Uninstall object
type ReconcileUninstall struct {
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
}

// Reconcile : The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileUninstall) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	if request.NamespacedName.Name == r.operatorNamespace {
		logrus.Info("Reconciling Uninstall")

		ns, err := GetNS(r.context, r.operatorNamespace, r.client)
		if err != nil {
			logrus.Errorf("could not retrieve %s namespace: %v", ns.Name, err)
		}
		if CheckLabel(ns) {
			logrus.Info("Uninstall label has been set")
			rhmi := &integreatlyv1alpha1.RHMI{}
			err := r.client.Get(context.TODO(), k8sclient.ObjectKey{Name: "rhmi", Namespace: request.NamespacedName.Name}, rhmi)
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

			if rhmi.DeletionTimestamp == nil {
				logrus.Info("Deleting RHMI CR")
				err := r.client.Delete(r.context, rhmi)
				if err != nil {
					logrus.Errorf("failed to delete RHMI CR: %v", err)
				}
			}
		}
		logrus.Info("Reconciling Uninstall completed")
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

// CheckLabel Checks namespace for deletion label based on label decided in https://issues.redhat.com/browse/SDA-2434
func CheckLabel(o metav1.Object) bool {
	for k, v := range o.GetLabels() {
		if k == "api.openshift.com/addon-rhmi-operator-delete" && v == "true" {
			return true
		}
	}
	return false
}
