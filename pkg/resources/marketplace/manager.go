package marketplace

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	IntegreatlyChannel = "rhmi"
	CatalogSourceName  = "rhmi-registry-cs"
	OperatorGroupName  = "rhmi-registry-og"
	Publisher          = "RHMI"
)

var log = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "marketplace"})

//go:generate moq -out MarketplaceManager_moq.go . MarketplaceInterface
type MarketplaceInterface interface {
	InstallOperator(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error
	GetSubscriptionInstallPlan(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*operatorsv1alpha1.InstallPlan, *operatorsv1alpha1.Subscription, error)
}

type Manager struct{}

var _ MarketplaceInterface = &Manager{}

func NewManager() *Manager {
	return &Manager{}
}

type Target struct {
	Namespace,
	SubscriptionName,
	Package,
	Channel string
}

func (m *Manager) InstallOperator(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler CatalogSourceReconciler) error {
	res, err := catalogSourceReconciler.Reconcile(ctx, t.SubscriptionName)
	if res.Requeue {
		return fmt.Errorf("Requeue")
	}
	if err != nil {
		return err
	}

	// catalog source is ready to create the other stuff
	if err := m.reconcileOperatorGroup(ctx, serverClient, t, operatorGroupNamespaces); err != nil {
		return err
	}

	log.Infof("Creating subscription in ns if it doesn't already exist", l.Fields{"ns": t.Namespace})
	sub := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      t.SubscriptionName,
		},
	}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: t.Namespace,
		Name:      t.SubscriptionName,
	}, sub)

	if err != nil {
		log.Error("getting subscription error", l.Fields{"subscription": t.SubscriptionName, "namespace": t.Namespace}, err)
	}

	mutateSub := func() error {

		if sub.Spec == nil {
			sub.Spec = &operatorsv1alpha1.SubscriptionSpec{}
		}

		sub.Spec.InstallPlanApproval = approvalStrategy
		sub.Spec.Channel = t.Channel
		sub.Spec.Package = t.Package
		sub.Spec.CatalogSource = catalogSourceReconciler.CatalogSourceName()
		sub.Spec.CatalogSourceNamespace = catalogSourceReconciler.CatalogSourceNamespace()

		if sub.Spec.Config == nil {
			sub.Spec.Config = &operatorsv1alpha1.SubscriptionConfig{}
		}
		if sub.Spec.Config.Resources == nil {
			sub.Spec.Config.Resources = &corev1.ResourceRequirements{}
		}

		if t.SubscriptionName == constants.ThreeScaleSubscriptionName {
			preflightsSkipEnvVar := corev1.EnvVar{
				Name:  "PREFLIGHT_CHECKS_BYPASS",
				Value: "true",
			}

			if sub.Spec.Config.Env == nil {
				sub.Spec.Config.Env = []corev1.EnvVar{}
			}

			var exists bool
			for i, env := range sub.Spec.Config.Env {
				if env.Name == preflightsSkipEnvVar.Name {
					sub.Spec.Config.Env[i].Value = preflightsSkipEnvVar.Value
					exists = true
					break
				}
			}

			if !exists {
				sub.Spec.Config.Env = append(sub.Spec.Config.Env, preflightsSkipEnvVar)
			}
		}

		return nil
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, sub, mutateSub)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error(fmt.Sprintf("Error creating sub: %v", err), nil, err)
		return err
	}

	return nil

}

func (m *Manager) reconcileOperatorGroup(ctx context.Context, serverClient k8sclient.Client, t Target, operatorGroupNamespaces []string) error {
	og := &v1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.Namespace,
			Name:      OperatorGroupName,
			Labels:    map[string]string{"integreatly": t.SubscriptionName},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, og, func() error {
		og.Spec = v1.OperatorGroupSpec{
			TargetNamespaces: operatorGroupNamespaces,
		}

		return nil
	})

	if err != nil {
		log.Error(fmt.Sprintf("Error creating or updating operator group: %v", err), nil, err)
		return err
	}

	return nil
}

func (m *Manager) getSubscription(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*operatorsv1alpha1.Subscription, error) {
	sub := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      subName,
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: sub.Name, Namespace: sub.Namespace}, sub)
	if err != nil {
		log.Error("Error getting subscription", l.Fields{"name": subName, "ns": ns}, err)
		return nil, err
	}
	return sub, nil
}

func (m *Manager) GetSubscriptionInstallPlan(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*operatorsv1alpha1.InstallPlan, *operatorsv1alpha1.Subscription, error) {
	log.Infof("Get", l.Fields{"Subscription Name": subName, "ns": ns})

	sub, err := m.getSubscription(ctx, serverClient, subName, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSubscriptionInstallPlan: %w", err)
	}
	if sub.Status.Install == nil || sub.Status.InstallPlanRef == nil {
		err = k8serr.NewNotFound(operatorsv1alpha1.Resource("installplan"), "")
		log.Error(fmt.Sprintf("Error getting install plan ref on subscription, %v", err), nil, err)
		return nil, sub, err
	}

	ip := &operatorsv1alpha1.InstallPlan{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      sub.Status.InstallPlanRef.Name,
		Namespace: sub.Status.InstallPlanRef.Namespace,
	}, ip); err != nil {
		if k8serr.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	return ip, sub, err
}
