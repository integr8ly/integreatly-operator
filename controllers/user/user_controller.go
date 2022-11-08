package controllers

import (
	"context"
	"time"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "user_controller"})

// UserReconciler reconciles a User object
type UserReconciler struct {
}

// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,resourceNames=rhmi-developers,verbs=update;delete
// +kubebuilder:rbac:groups=user.openshift.io,resources=users,verbs=watch;get;list

func (r *UserReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log.Info("Reconciling User")

	// new client to avoid caching issues
	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10
	scheme := runtime.NewScheme()
	err := rhmiv1alpha1.AddToSchemes.AddToScheme(scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	c, _ := k8sclient.New(restConfig, k8sclient.Options{Scheme: scheme})

	rhmiGroup := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhmi-developers",
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, c, rhmiGroup, func() error {
		users := &usersv1.UserList{}
		err := c.List(ctx, users)
		if err != nil {
			return err
		}

		groups := &usersv1.GroupList{}
		err = c.List(ctx, groups)
		if err != nil {
			return err
		}

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
		if !userHelper.UserInExclusionGroup(user, groups) {
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
