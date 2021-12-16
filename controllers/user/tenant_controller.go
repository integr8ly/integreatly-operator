package controllers

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	usersv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"time"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	rhoamTenantConfigMap = "tenant-cm"
)

var logt = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "tenant_controller"})

type TenantReconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
	controller      controller.Controller
	log      l.Logger
}

func (r *TenantReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logt.Info("############  TENANT Controller ########")

	// new client to avoid caching issues
	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10
	scheme := runtime.NewScheme()
	err := rhmiv1alpha1.AddToSchemes.AddToScheme(scheme)

	//client, _ := k8sclient.New(restConfig, k8sclient.Options{Scheme: scheme})
	//ctx := context.TODO()

	return ctrl.Result{}, err
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logt.Info("############  TENANT SetupWithManager  ########")
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&usersv1.User{}).
		//Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(namePredicate(rhoamTenantConfigMap))).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(newObjectPredicate(isName(rhoamTenantConfigMap)))).
		Build(r)
	if err != nil {
		return err
	}
	r.controller = controller
	return nil
}

//func namePredicate(name string) predicate.Predicate {
//	return predicate.NewPredicateFuncs(func(m metav1.Object, _ runtime.Object) bool {
//		return m.GetName() == name
//	})
//}

type objectPredicate struct {
	Predicate func(mo handler.MapObject) bool
}

func newObjectPredicate(predicate func(mo handler.MapObject) bool) *objectPredicate {
	return &objectPredicate{
		Predicate: predicate,
	}
}

func (p *objectPredicate) Create(e event.CreateEvent) bool {
	return p.Predicate(handler.MapObject{Meta: e.Meta, Object: e.Object})
}

func (p *objectPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p *objectPredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (p *objectPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func isName(name string) func(handler.MapObject) bool {
	return func(mo handler.MapObject) bool {
		return mo.Meta.GetName() == name
	}
}
