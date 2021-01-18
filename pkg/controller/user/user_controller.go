package user

import (
	"context"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"time"

	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	usersv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "subscription_controller"})

// Add creates a new User Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileUser{}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("user-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource User
	err = c.Watch(&source.Kind{Type: &usersv1.User{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileUser implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileUser{}

// ReconcileUser reconciles a User object
type ReconcileUser struct {
}

func (r *ReconcileUser) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Logger = log.WithContext(l.Fields{l.ControllerLogContext: "user_controller"})
	log.Info("Reconciling User")

	// new client to avoid caching issues
	restConfig := controllerruntime.GetConfigOrDie()
	restConfig.Timeout = time.Second * 10
	c, _ := k8sclient.New(restConfig, k8sclient.Options{})
	ctx := context.TODO()

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

	return reconcile.Result{}, err
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
