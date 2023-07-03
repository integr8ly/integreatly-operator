package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WithConditions applies the given conditions.
type WithConditions []metav1.Condition

func (w WithConditions) ConfigureSendPulse(c *sendPulseConfig) {
	c.Conditions = []metav1.Condition(w)
}
