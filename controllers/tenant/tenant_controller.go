package controllers

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	usersv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

const (
	rhoamTenantConfigMap = "tenant-cm"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "tenant_controller"})


// +kubebuilder:rbac:groups=user.openshift.io,resources=users,verbs=watch;get;list;update

func New(mgr manager.Manager) (*TenantReconciler, error) {
	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10

	client, err := k8sclient.New(restConfig, k8sclient.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		return nil, err
	}

	return &TenantReconciler{
		Client: client,
		Scheme: mgr.GetScheme(),
		mgr:    mgr,
		log:    l.Logger{},
	}, nil
}

type TenantReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme
	mgr    manager.Manager
	log    l.Logger
}

func (r *TenantReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	log.Info("############  TENANT Controller ########")
	log.Info("############ reconcile Request, cm name: " + request.Name + ", namespace: " + request.Namespace)

	err := r.addAnnotationToUser(request.Namespace)
	if err != nil {
		r.log.Error("error in addAnnotationToUser: %s ", err)
	}
	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log.Info("############  TENANT SetupWithManager  ########")
	return ctrl.NewControllerManagedBy(mgr).
		For(&usersv1.User{}).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(
			newObjectPredicate(isName(rhoamTenantConfigMap)))).
		Complete(r)
}

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

func (r *TenantReconciler) addAnnotationToUser(namespaceName string) error {
	//userName == namespaceName
	log.Info("#### addAnnotationToUser")
	usersList := &usersv1.UserList{}
	err := r.Client.List(context.TODO(), usersList)
	if err != nil {
		return fmt.Errorf("error getting users list")
	}
	for _, user := range usersList.Items {
		if user.Name == namespaceName {
			log.Info("#### Adding Annotation to user: " + user.Name)
			_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &user, func() error {
				if user.Annotations == nil {
					user.Annotations = map[string]string{}
				}
				user.Annotations["tenant"] = "yes"
				return nil
			})
		}
	}
	return nil
}
