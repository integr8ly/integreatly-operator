package webhooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	pkgerr "github.com/pkg/errors"

	"k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// WebhookReconciler knows how to reconcile webhook configuration CRs
type WebhookReconciler interface {
	SetName(name string)
	SetRule(rule RuleWithOperations)
	Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error
}

type CompositeWebhookReconciler struct {
	Reconcilers []WebhookReconciler
}

func (reconciler *CompositeWebhookReconciler) SetName(name string) {
	for _, innerReconciler := range reconciler.Reconcilers {
		innerReconciler.SetName(name)
	}
}

func (reconciler *CompositeWebhookReconciler) SetRule(rule RuleWithOperations) {
	for _, innerReconciler := range reconciler.Reconcilers {
		innerReconciler.SetRule(rule)
	}
}

func (reconciler *CompositeWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	for _, innerReconciler := range reconciler.Reconcilers {
		if err := innerReconciler.Reconcile(ctx, client, caBundle); err != nil {
			return err
		}
	}

	return nil
}

type ValidatingWebhookReconciler struct {
	Path string
	name string
	rule RuleWithOperations
}

type MutatingWebhookReconciler struct {
	Path string
	name string
	rule RuleWithOperations
}

func (reconciler *MutatingWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	var (
		sideEffects    = v1beta1.SideEffectClassNone
		port           = int32(servicePort)
		matchPolicy    = v1beta1.Exact
		failurePolicy  = v1beta1.Fail
		timeoutSeconds = int32(30)
	)
	watchNS, err := resources.GetWatchNamespace()
	if err != nil {
		return pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	cr := &v1beta1.MutatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s.integreatly.org", reconciler.name),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.MutatingWebhook{
			{
				Name:        fmt.Sprintf("%s-mutating-config.integreatly.org", reconciler.name),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: namespacePrefix + "operator",
						Name:      operatorPodServiceName,
						Path:      &reconciler.Path,
						Port:      &port,
					},
				},
				Rules: []v1beta1.RuleWithOperations{
					{
						Operations: reconciler.rule.Operations,
						Rule: v1beta1.Rule{
							APIGroups:   reconciler.rule.APIGroups,
							APIVersions: reconciler.rule.APIVersions,
							Resources:   reconciler.rule.Resources,
							Scope:       &reconciler.rule.Scope,
						},
					},
				},
				MatchPolicy:             &matchPolicy,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &timeoutSeconds,
			},
		}
		return nil
	})
	return err
}

func (reconciler *ValidatingWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	var (
		sideEffects    = v1beta1.SideEffectClassNone
		port           = int32(servicePort)
		matchPolicy    = v1beta1.Exact
		failurePolicy  = v1beta1.Fail
		timeoutSeconds = int32(30)
	)
	watchNS, err := resources.GetWatchNamespace()
	if err != nil {
		return pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	cr := &v1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s.integreatly.org", reconciler.name),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.ValidatingWebhook{
			{
				Name:        fmt.Sprintf("%s-validating-config.integreatly.org", reconciler.name),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: namespacePrefix + "operator",
						Name:      operatorPodServiceName,
						Path:      &reconciler.Path,
						Port:      &port,
					},
				},
				Rules: []v1beta1.RuleWithOperations{
					{
						Operations: reconciler.rule.Operations,
						Rule: v1beta1.Rule{
							APIGroups:   reconciler.rule.APIGroups,
							APIVersions: reconciler.rule.APIVersions,
							Resources:   reconciler.rule.Resources,
							Scope:       &reconciler.rule.Scope,
						},
					},
				},
				MatchPolicy:             &matchPolicy,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &timeoutSeconds,
			},
		}
		return nil
	})
	return err
}

func (reconciler *ValidatingWebhookReconciler) SetName(name string) {
	reconciler.name = name
}

func (reconciler *MutatingWebhookReconciler) SetName(name string) {
	reconciler.name = name
}

func (reconciler *ValidatingWebhookReconciler) SetRule(rule RuleWithOperations) {
	reconciler.rule = rule
}

func (reconciler *MutatingWebhookReconciler) SetRule(rule RuleWithOperations) {
	reconciler.rule = rule
}
