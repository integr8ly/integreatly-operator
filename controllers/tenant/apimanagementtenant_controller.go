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
	"strings"
	"time"
)

var log = l.NewLoggerWithContext(l.Fields{l.ControllerLogContext: "tenant_controller"})

// +kubebuilder:rbac:groups=integreatly.org,resources=apimanagementtenant,verbs=get;list;watch
// +kubebuilder:rbac:groups=integreatly.org,resources=apimanagementtenant/status,verbs=get;update;patch
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

func (r *TenantReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log.Info(fmt.Sprintf("TenantReconciler request: %s", request))

	tenant, err := r.getAPIManagementTenant(request.Name, request.Namespace)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("failed to get APIManagementTenant", err)
		return ctrl.Result{}, err
	}

	isTenantVerified, rejectionReason, err := r.verifyAPIManagementTenant(tenant)
	if err != nil {
		log.Error("error verifying the APIManagementTenant CR", err)
		if err1 := r.updateLastError(tenant, err.Error()); err1 != nil {
			return ctrl.Result{}, err1
		}
		return ctrl.Result{}, err
	}
	if !isTenantVerified {
		// Update APIManagementTenant fields to reflect failed verification
		if err1 := r.updateProvisioningStatus(tenant, v1alpha1.WontProvisionTenant); err1 != nil {
			return ctrl.Result{}, err1
		}
		if err1 := r.updateLastError(tenant, rejectionReason); err1 != nil {
			return ctrl.Result{}, err1
		}

		log.Warning(fmt.Sprintf("tenant %s in namespace %s will not be reconciled because %s", tenant.Name, tenant.Namespace, rejectionReason))
		return ctrl.Result{}, nil
	}

	err = r.addAnnotationToUser(tenant)
	if err != nil {
		if err1 := r.updateLastError(tenant, err.Error()); err1 != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second}, err1
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second}, err
	}

	wasTenantUrlReconciled, err := r.reconcileTenantUrl(tenant)
	if err == nil && !wasTenantUrlReconciled {
		return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
	}
	if err != nil {
		log.Error("error reconciling tenant URL", err)
		if err1 := r.updateLastError(tenant, err.Error()); err1 != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second}, err1
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second}, err
	}

	// Clear out LastError since reconcile finished successfully.
	if err1 := r.updateLastError(tenant, ""); err1 != nil {
		return ctrl.Result{}, err1
	}

	log.Info(fmt.Sprintf("TenantReconciler finished: %s", request))
	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.APIManagementTenant{}).
		Watches(&v1alpha1.APIManagementTenant{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func (r *TenantReconciler) getAPIManagementTenant(crName string, crNamespace string) (*v1alpha1.APIManagementTenant, error) {
	tenant := &v1alpha1.APIManagementTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: crNamespace,
		},
	}
	key := k8sclient.ObjectKeyFromObject(tenant)
	err := r.Get(context.TODO(), key, tenant)
	if err != nil {
		return nil, fmt.Errorf("error getting tenant %s: %v", crName, err)
	}
	return tenant, nil
}

// The purpose of this method is to verify an APIManagementTenant CR is valid and should be reconciled
func (r *TenantReconciler) verifyAPIManagementTenant(tenant *v1alpha1.APIManagementTenant) (bool, string, error) {
	// Skip verification if the tenant has already been verified
	if tenant.Status.ProvisioningStatus != v1alpha1.ThreeScaleAccountReady && tenant.Status.ProvisioningStatus != v1alpha1.ThreeScaleAccountRequested && tenant.Status.ProvisioningStatus != v1alpha1.UserAnnotated {
		log.Info(fmt.Sprintf("TenantReconciler verifyAPIManagementTenant: %v", tenant))

		// Fails if APIManagementTenant isn't from a namespace ending in -dev or -stage
		if !strings.HasSuffix(tenant.Namespace, "-dev") && !strings.HasSuffix(tenant.Namespace, "-stage") {
			return false, "tenant not created in a namespace ending in {USERNAME}-dev or {USERNAME}-stage", nil
		}
		// Check if User exists based on APIManagementTenant's namespace
		user, err := r.getUserByTenantNamespace(tenant.Namespace)
		if err != nil {
			if k8serr.IsNotFound(err) {
				return false, "the user extracted by the APIManagementTenant's namespace does not exist", nil
			}
			return false, "an error occurred while trying to get the user by the APIManagementTenant's namespace", err
		}
		// Check if a tenant already exists in either of the users namespaces
		namespacesToCheck := []string{
			user.Name + "-dev",
			user.Name + "-stage",
		}
		for _, ns := range namespacesToCheck {
			tenants := &v1alpha1.APIManagementTenantList{}
			listOpts := []k8sclient.ListOption{
				k8sclient.InNamespace(ns),
			}
			err := r.Client.List(context.TODO(), tenants, listOpts...)
			if err != nil {
				return false, "an error occurred while trying to check if another reconciled APIManagementTenant CR already exists", err
			}
			for _, t := range tenants.Items {
				if t.Status.ProvisioningStatus == v1alpha1.ThreeScaleAccountReady || t.Status.ProvisioningStatus == v1alpha1.ThreeScaleAccountRequested || t.Status.ProvisioningStatus == v1alpha1.UserAnnotated {
					return false, "can't create more than 1 APIManagementTenant CR in -dev or -stage namespace", nil
				}

			}
		}
	}

	return true, "", nil
}

func (r *TenantReconciler) addAnnotationToUser(tenant *v1alpha1.APIManagementTenant) error {
	// Only add the annotation to the User if its APIManagementTenant's ProvisioningStatus hasn't been set to a value yet.
	if tenant.Status.ProvisioningStatus == "" {
		log.Info(fmt.Sprintf("TenantReconciler addAnnotationToUser: %v", tenant))

		user, err := r.getUserByTenantNamespace(tenant.Namespace)
		if err != nil {
			return fmt.Errorf("error getting user for tenant %s: %v", tenant.Name, err)
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
		err = r.updateProvisioningStatus(tenant, v1alpha1.UserAnnotated)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TenantReconciler) reconcileTenantUrl(tenant *v1alpha1.APIManagementTenant) (bool, error) {
	tenantUrlReconciled := true
	if tenant.Status.ProvisioningStatus != v1alpha1.ThreeScaleAccountReady {
		log.Info(fmt.Sprintf("TenantReconciler reconcileTenantUrl: %v", tenant))

		tenantUrlReconciled = false // Reset value because tenant hasn't been reconciled yet
		selector, err := labels.Parse("zync.3scale.net/route-to=system-provider")
		if err != nil {
			return tenantUrlReconciled, err
		}
		opts := k8sclient.ListOptions{
			LabelSelector: selector,
			Namespace:     "sandbox-rhoam-3scale",
		}

		routes := routev1.RouteList{}
		err = r.Client.List(context.TODO(), &routes, &opts)
		if err != nil {
			return tenantUrlReconciled, err
		}
		if len(routes.Items) == 0 {
			return tenantUrlReconciled, fmt.Errorf("failed to find any system-developer routes in namespace %s", opts.Namespace)
		}

		var foundRoute *routev1.Route
		user, err := r.getUserByTenantNamespace(tenant.Namespace)
		if err != nil {
			return tenantUrlReconciled, err
		}
		for i := range routes.Items {
			rt := routes.Items[i]
			if strings.Contains(rt.Spec.Host, user.Name) {
				foundRoute = &rt
				break
			}
		}
		if foundRoute == nil {
			// If no matching route was found, then the account is still being created
			// Set the provisioningStatus to ThreeScaleAccountRequested
			err := r.updateProvisioningStatus(tenant, v1alpha1.ThreeScaleAccountRequested)
			if err != nil {
				return tenantUrlReconciled, err
			}
			// Update the value of lastError to reflect the user's SSO not being ready
			err = r.updateLastError(tenant, fmt.Sprintf("waiting for route creation for tenant %s", tenant.Name))
			if err != nil {
				return tenantUrlReconciled, err
			}
			return tenantUrlReconciled, nil
		}

		// Get the value of the `ssoReady` annotation if it exists
		ssoReady := user.Annotations["ssoReady"]
		if ssoReady != "yes" {
			// If the user's SSO is not ready, then the tenantUrl isn't ready
			// Set the provisioningStatus to ThreeScaleAccountRequested
			err := r.updateProvisioningStatus(tenant, v1alpha1.ThreeScaleAccountRequested)
			if err != nil {
				return tenantUrlReconciled, err
			}
			// Update the value of lastError to reflect the user's SSO not being ready
			err = r.updateLastError(tenant, fmt.Sprintf("waiting for SSO for user %s to be ready", user.Name))
			if err != nil {
				return tenantUrlReconciled, err
			}
			return tenantUrlReconciled, nil
		}

		// Since the user's SSO is ready, update the tenant's tenantUrl and provisioningStatus
		err = r.updateTenantUrl(tenant, foundRoute.Spec.Host)
		if err != nil {
			return tenantUrlReconciled, err
		}
		err = r.updateProvisioningStatus(tenant, v1alpha1.ThreeScaleAccountReady)
		if err != nil {
			return tenantUrlReconciled, err
		}
		tenantUrlReconciled = true
	}
	return tenantUrlReconciled, nil
}

func (r *TenantReconciler) updateLastError(tenant *v1alpha1.APIManagementTenant, message string) error {
	tenant.Status.LastError = message
	err := r.Client.Status().Update(context.TODO(), tenant)
	if err != nil {
		log.Error("error updating status of APIManagementTenant CR", err)
	}
	return err
}

func (r *TenantReconciler) updateProvisioningStatus(tenant *v1alpha1.APIManagementTenant, status v1alpha1.ProvisioningStatus) error {
	tenant.Status.ProvisioningStatus = status
	err := r.Client.Status().Update(context.TODO(), tenant)
	if err != nil {
		return fmt.Errorf("error updating the provisioningStatus to %s for tenant %s: %v", status, tenant.Name, err)
	}

	return nil
}

func (r *TenantReconciler) updateTenantUrl(tenant *v1alpha1.APIManagementTenant, url string) error {
	tenant.Status.TenantUrl = url
	err := r.Client.Status().Update(context.TODO(), tenant)
	if err != nil {
		return fmt.Errorf("error updating the tenantUrl to %s for tenant %s: %v", url, tenant.Name, err)
	}

	return nil
}

func (r *TenantReconciler) getUserByTenantNamespace(ns string) (*usersv1.User, error) {
	// Extract name from namespace
	username := ns
	username = strings.TrimSuffix(username, "-dev")
	username = strings.TrimSuffix(username, "-stage")

	user := &usersv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: username,
		},
	}
	key := k8sclient.ObjectKeyFromObject(user)
	err := r.Client.Get(context.TODO(), key, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
