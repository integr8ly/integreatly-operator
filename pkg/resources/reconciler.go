package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	v1alpha12 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// This is the base reconciler that all the other reconcilers extend. It handles things like namespace creation, subscription creation etc

type Reconciler struct {
	mpm marketplace.MarketplaceInterface
}

func NewReconciler(mpm marketplace.MarketplaceInterface) *Reconciler {
	return &Reconciler{
		mpm: mpm,
	}
}

func (r *Reconciler) ReconcileNamespace(ctx context.Context, namespace string, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: v12.ObjectMeta{
			Name: namespace,
		},
	}
	err := client.Get(ctx, pkgclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		if !errors2.IsNotFound(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve namespace: %s", ns.Name))
		}
		prepareNS(ns, inst)
		if err = client.Create(ctx, ns); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create namespace: %s", ns.Name))
		}
		return v1alpha1.PhaseCompleted, nil
	}
	// ns exists so check it is our namespace
	if !NSIsOwnedBy(ns, inst) && ns.Status.Phase != v1.NamespaceTerminating {
		return v1alpha1.PhaseFailed, errors.New("existing namespace found with name " + ns.Name + " but it is not owned by the integreatly installation and it isn't being deleted")
	}
	if ns.Status.Phase == v1.NamespaceTerminating {
		logrus.Debugf("namespace %s is terminating, maintaining phase to try again on next reconcile", namespace)
		return v1alpha1.PhaseInProgress, nil
	}
	prepareNS(ns, inst)
	if err := client.Update(ctx, ns); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to update the ns definition ")
	}
	if ns.Status.Phase != v1.NamespaceActive {
		return v1alpha1.PhaseInProgress, nil
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcileSubscription(ctx context.Context, inst *v1alpha1.Installation, subscriptionName, namespace string, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling subscription %s from channel %s in namespace: %s", subscriptionName, "integreatly", namespace)
	err := r.mpm.CreateSubscription(
		ctx,
		client,
		inst,
		marketplace.GetOperatorSources().Integreatly,
		namespace,
		subscriptionName,
		marketplace.IntegreatlyChannel,
		[]string{namespace},
		v1alpha12.ApprovalAutomatic)
	if err != nil && !errors2.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create subscription in namespace: %s", namespace))
	}
	ip, sub, err := r.mpm.GetSubscriptionInstallPlan(ctx, client, subscriptionName, namespace)
	if err != nil {
		// this could be the install plan or subscription so need to check if sub nil or not TODO refactor
		if errors2.IsNotFound(err) {
			if sub != nil {
				logrus.Infof("time since created %v", time.Now().Sub(sub.CreationTimestamp.Time).Seconds())
			}
			if sub != nil && time.Now().Sub(sub.CreationTimestamp.Time) > config.SubscriptionTimeout {
				// delete subscription so it is recreated
				logrus.Info("removing subscription as no install plan ready yet will recreate", sub.Name)
				if err := client.Delete(ctx, sub, func(options *pkgclient.DeleteOptions) {
					gp := int64(0)
					options.GracePeriodSeconds = &gp
				}); err != nil && !errors2.IsNotFound(err) {
					return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to delete existing subscription "+subscriptionName)
				}
			}
			logrus.Debugf(fmt.Sprintf("installplan resource is not found in namespace: %s", namespace))
			return v1alpha1.PhaseAwaitingOperator, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve installplan and subscription in namespace: %s", namespace))
	}

	logrus.Debugf("installplan phase is %s", ip.Status.Phase)
	if ip.Status.Phase != v1alpha12.InstallPlanPhaseComplete {
		logrus.Infof("%s install plan is not complete yet ", subscriptionName)
		return v1alpha1.PhaseInProgress, nil
	}
	logrus.Infof("%s install plan is complete. Installation ready ", subscriptionName)
	return v1alpha1.PhaseCompleted, nil
}

func prepareNS(ns *v1.Namespace, install *v1alpha1.Installation) {
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ref := v12.NewControllerRef(install, v1alpha1.SchemaGroupVersionKind)
	ns.Labels["integreatly"] = "true"
	refExists := false
	for _, er := range ns.OwnerReferences {
		if er.Name == ref.Name {
			refExists = true
			break
		}
	}
	if !refExists {
		ns.OwnerReferences = append(ns.OwnerReferences, *ref)
	}
}

func NSIsOwnedBy(ns *v1.Namespace, owner *v1alpha1.Installation) bool {
	for _, or := range ns.OwnerReferences {
		if or.Name == owner.Name && or.APIVersion == owner.APIVersion {
			return true
		}
	}
	return false
}
