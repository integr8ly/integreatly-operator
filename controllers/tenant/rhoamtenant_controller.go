package controllers

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	usersv1 "github.com/openshift/api/user/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "tenant_controller"})

// +kubebuilder:rbac:groups=integreatly.org,resources=rhoamtenant,verbs=get;list;watch
// +kubebuilder:rbac:groups=integreatly.org,resources=rhoamtenant/status,verbs=get;update;patch
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
	tenant := &v1alpha1.RhoamTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.Name,
		},
	}
	key, err := k8sclient.ObjectKeyFromObject(tenant)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error get ObjectKey for tenant: %v", err)
	}
	err = r.Get(context.TODO(), key, tenant)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	err = r.addAnnotationToUser(request.Name)
	if err != nil {
		tenant.Status.LastError = err.Error()
		err1 := r.Client.Status().Update(context.TODO(), tenant)
		if err1 != nil {
			log.Error("error update status of RhoamTenant CR", err1)
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RhoamTenant{}).
		Watches(&source.Kind{Type: &v1alpha1.RhoamTenant{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func (r *TenantReconciler) addAnnotationToUser(crName string) error {
	user := &usersv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
	}
	key, err := k8sclient.ObjectKeyFromObject(user)
	if err != nil {
		return fmt.Errorf("error get ObjectKey for user: %v", err)
	}
	err = r.Client.Get(context.TODO(), key, user)
	if err != nil {
		return fmt.Errorf("error get user with name as current CR, %v", err)
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, user, func() error {
		if user.Annotations == nil {
			user.Annotations = map[string]string{}
		}
		user.Annotations["tenant"] = "yes"
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to add tenant annotation to user %s, %w", user.Name, err)
	}
	return nil
}
