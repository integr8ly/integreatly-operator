package controllers

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	usersv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,resourceNames=rhmi-developers,verbs=update;delete
// +kubebuilder:rbac:groups=user.openshift.io,resources=users,verbs=watch;get;list

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "user_controller"})

// UserReconciler reconciles a User object
type UserReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme
	mgr    manager.Manager
}

func New(mgr manager.Manager) *UserReconciler {
	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10

	client, err := k8sclient.New(restConfig, k8sclient.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		panic("could not setup k8s client for user controller")
	}

	return &UserReconciler{
		Client: client,
		Scheme: mgr.GetScheme(),
		mgr:    mgr,
	}
}

func (r *UserReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	log.Info("Reconciling User")

	rhmiGroup := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhmi-developers",
		},
	}

	users := &usersv1.UserList{}
	err := r.Client.List(ctx, users)
	if err != nil {
		return ctrl.Result{}, err
	}

	groups := &usersv1.GroupList{}
	err = r.Client.List(ctx, groups)
	if err != nil {
		return ctrl.Result{}, err
	}

	or, err := controllerutil.CreateOrUpdate(ctx, r.Client, rhmiGroup, func() error {

		rhmiGroup.Users = mapUserNames(users, groups)

		return nil
	})
	log.Infof("Operation Result", l.Fields{"groupName": rhmiGroup.Name, "result": string(or)})

	return ctrl.Result{}, err
}

func mapUserNames(users *usersv1.UserList, groups *usersv1.GroupList) []string {
	var result = []string{}
	for _, user := range users.Items {
		// Certain users such as sre do not need to be added
		if !userHelper.IsInExclusionGroup(user, groups) {
			result = append(result, user.Name)
		}
	}

	return result
}

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&usersv1.User{}).
		Complete(r)
}
