package client

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
)

// AddonInstanceClient provides methods for interacting with
// AddonInstance resources.
type AddonInstanceClient interface {
	// SendPulse updates the LastHeartbeatTime for the AddonInstance
	// and applies optional conditions if provided.
	SendPulse(ctx context.Context, instance av1alpha1.AddonInstance, opts ...SendPulseOption) error
}

// NewAddonInstanceClient returns a configured AddonInstanceClient implementation
// using the given client instance as a base.
func NewAddonInstanceClient(client client.Client) *AddonInstanceClientImpl {
	return &AddonInstanceClientImpl{
		client: client,
	}
}

type AddonInstanceClientImpl struct {
	client client.Client
}

func (c *AddonInstanceClientImpl) SendPulse(ctx context.Context, instance av1alpha1.AddonInstance, opts ...SendPulseOption) error {
	var cfg sendPulseConfig

	cfg.Option(opts...)

	instance.Status.LastHeartbeatTime = metav1.Now()

	for _, c := range cfg.Conditions {
		c.ObservedGeneration = instance.Generation

		meta.SetStatusCondition(&instance.Status.Conditions, c)
	}

	if err := c.client.Status().Update(ctx, &instance); err != nil {
		return fmt.Errorf("setting status for AddonInstance %s/%s: %w", instance.Namespace, instance.Name, err)
	}

	return nil
}

type sendPulseConfig struct {
	Conditions []metav1.Condition
}

func (c *sendPulseConfig) Option(opts ...SendPulseOption) {
	for _, opt := range opts {
		opt.ConfigureSendPulse(c)
	}
}

type SendPulseOption interface {
	ConfigureSendPulse(*sendPulseConfig)
}

// NewAddonInstanceConditionDegraded returns an AddonInstanceDegraded status condition
// with the given status, reason, and message.
func NewAddonInstanceConditionDegraded(status metav1.ConditionStatus, reason, msg string) metav1.Condition {
	return newAddonInstanceCondition(av1alpha1.AddonInstanceConditionDegraded, status, reason, msg)
}

// NewAddonInstanceConditionInstalled returns an AddonInstanceInstalled status condition
// with the given status, reason, and message.
func NewAddonInstanceConditionInstalled(status metav1.ConditionStatus, reason av1alpha1.AddonInstanceInstalledReason, msg string) metav1.Condition {
	return newAddonInstanceCondition(av1alpha1.AddonInstanceConditionInstalled, status, reason.String(), msg)
}

func newAddonInstanceCondition(cond av1alpha1.AddonInstanceCondition, status metav1.ConditionStatus, reason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:    cond.String(),
		Status:  status,
		Reason:  reason,
		Message: msg,
	}
}
