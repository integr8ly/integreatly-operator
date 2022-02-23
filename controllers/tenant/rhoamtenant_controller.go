package controllers

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "tenant_controller"})

// +kubebuilder:rbac:groups=integreatly.org,resources=rhoamtenant,verbs=get;list;watch
// +kubebuilder:rbac:groups=integreatly.org,resources=rhoamtenant/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=user.openshift.io,resources=users,verbs=watch;get;list;update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

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
	tenant, err := r.getRhoamTenant(request.Name)
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
			log.Error("error updating status of RhoamTenant CR", err1)
		}
		return ctrl.Result{}, err
	}

	err = r.reconcileTenantUrl(request.Name)
	if err != nil {
		tenant.Status.LastError = err.Error()
		err1 := r.Client.Status().Update(context.TODO(), tenant)
		if err1 != nil {
			log.Error("error updating status of RhoamTenant CR", err1)
		}
		return ctrl.Result{}, err
	}

	// Clear out LastError since reconcile finished successfully.
	tenant.Status.LastError = ""
	err = r.Client.Status().Update(context.TODO(), tenant)
	if err != nil {
		log.Error("error updating status of RhoamTenant CR", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {

	//enqueueAllInstallations := &handler.EnqueueRequestsFromMapFunc{
	//	ToRequests: installationMapper{context: context.TODO(), client: mgr.GetClient()},
	//}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RhoamTenant{}).
		Watches(&source.Kind{Type: &v1alpha1.RhoamTenant{}}, &handler.EnqueueRequestForObject{}).
		//Watches(&source.Kind{Type: &v1alpha1.RhoamTenant{}}, enqueueAllInstallations).
		Complete(r)
}

func (r *TenantReconciler) addAnnotationToUser(crName string) error {
	tenant, err := r.getRhoamTenant(crName)
	if err != nil {
		return err
	}

	// Only add the annotation to the User if its RhoamTenant's ProvisioningStatus hasn't been set to a value yet.
	if tenant.Status.ProvisioningStatus == "" {
		user := &usersv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: crName,
			},
		}
		key, err := k8sclient.ObjectKeyFromObject(user)
		if err != nil {
			return fmt.Errorf("error getting ObjectKey for user %s: %v", crName, err)
		}
		err = r.Client.Get(context.TODO(), key, user)
		if err != nil {
			return fmt.Errorf("error getting user %s: %v", crName, err)
		}
		_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, user, func() error {
			if user.Annotations == nil {
				user.Annotations = map[string]string{}
			}
			user.Annotations["tenant"] = "yes"
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to add tenant annotation to user %s: %v", user.Name, err)
		}

		// Update tenant's ProvisioningStatus to UserAnnotated
		err = r.updateProvisioningStatus(crName, v1alpha1.UserAnnotated)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TenantReconciler) reconcileTenantUrl(crName string) error {
	tenant, err := r.getRhoamTenant(crName)
	if err != nil {
		return err
	}

	// Only check for the 3scale account and route once per rhoam-tenant
	if tenant.Status.ProvisioningStatus != v1alpha1.ThreeScaleAccountReady {
		selector, err := labels.Parse("zync.3scale.net/route-to=system-provider")
		if err != nil {
			return err
		}
		opts := k8sclient.ListOptions{
			LabelSelector: selector,
			Namespace:     "sandbox-rhoam-3scale",
		}

		routes := routev1.RouteList{}
		err = r.Client.List(context.TODO(), &routes, &opts)
		if err != nil {
			return err
		}
		if len(routes.Items) == 0 {
			return fmt.Errorf("failed to find any system-developer routes in namespace %s", opts.Namespace)
		}

		var foundRoute *routev1.Route
		for _, rt := range routes.Items {
			if strings.Contains(rt.Spec.Host, crName) {
				foundRoute = &rt
				break
			}
		}
		if foundRoute == nil {
			// If no matching route was found, then the account is still being created.
			// Set the provisioningStatus to ThreeScaleAccountRequested and return an error.
			err = r.updateProvisioningStatus(crName, v1alpha1.ThreeScaleAccountRequested)
			if err != nil {
				return err
			}
			return fmt.Errorf("failed to find matching route in namespace %s for tenant %s", opts.Namespace, crName)
		}

		// Since the matching route was found, update the tenant's tenantUrl and provisioningStatus.
		err = r.updateTenantUrl(crName, foundRoute.Spec.Host)
		if err != nil {
			return err
		}
		err = r.updateProvisioningStatus(crName, v1alpha1.ThreeScaleAccountReady)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *TenantReconciler) updateProvisioningStatus(crName string, status v1alpha1.ProvisioningStatus) error {
	tenant, err := r.getRhoamTenant(crName)
	if err != nil {
		return err
	}
	tenant.Status.ProvisioningStatus = status
	err = r.Client.Status().Update(context.TODO(), tenant)
	if err != nil {
		return fmt.Errorf("error updating the provisioningStatus to %s for tenant %s: %v", status, crName, err)
	}

	return nil
}

func (r *TenantReconciler) updateTenantUrl(crName string, url string) error {
	tenant, err := r.getRhoamTenant(crName)
	if err != nil {
		return err
	}
	tenant.Status.TenantUrl = url
	err = r.Client.Status().Update(context.TODO(), tenant)
	if err != nil {
		return fmt.Errorf("error updating the tenantUrl to %s for tenant %s: %v", url, crName, err)
	}

	return nil
}

func (r *TenantReconciler) getRhoamTenant(crName string) (*v1alpha1.RhoamTenant, error) {
	tenant := &v1alpha1.RhoamTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
	}
	key, err := k8sclient.ObjectKeyFromObject(tenant)
	if err != nil {
		return tenant, fmt.Errorf("error getting ObjectKey for tenant %s: %v", crName, err)
	}
	err = r.Get(context.TODO(), key, tenant)
	if err != nil {
		return tenant, fmt.Errorf("error getting tenant %s: %v", crName, err)
	}
	return tenant, nil
}
