package config

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RateLimitConfigMapName = "sku-limits-managed-api-service"
	AlertConfigMapName     = "rate-limit-alerts"
	ManagedApiServiceQuota = "RHOAM SERVICE SKU"

	AlertTypeThreshold = "Threshold"
	AlertTypeSpike     = "Spike"

	DefaultRateLimitUnit     = "minute"
	DefaultRateLimitRequests = 13860
)

type RateLimitConfig struct {
	Unit            string `json:"unit"`
	RequestsPerUnit uint32 `json:"requests_per_unit"`
}

type AlertConfig struct {
	Type      string                `json:"type"`
	Level     string                `json:"level"`
	RuleName  string                `json:"ruleName"`
	Period    string                `json:"period"`
	Threshold *AlertThresholdConfig `json:"threshold,omitempty"`
}

type AlertThresholdConfig struct {
	MinRate string  `json:"minRate,omitempty"`
	MaxRate *string `json:"maxRate,omitempty"`
}

func GetAlertConfig(ctx context.Context, client k8sclient.Client, namespace string) (map[string]*AlertConfig, error) {
	alertsConfig := map[string]*AlertConfig{}
	err := getFromJSONConfigMap(
		ctx, client,
		AlertConfigMapName, namespace, "alerts",
		&alertsConfig,
	)

	return alertsConfig, err
}

func GetQuota(_ context.Context, _ k8sclient.Client) (string, error) {
	return ManagedApiServiceQuota, nil
}

func getFromJSONConfigMap(ctx context.Context, client k8sclient.Client, cmName, namespace, configkey string, v interface{}) error {
	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      cmName,
		Namespace: namespace,
	}, configMap); err != nil {
		return err
	}

	configJSON, ok := configMap.Data[configkey]
	if !ok {
		return fmt.Errorf("%s not found in %s ConfigMap data", configkey, cmName)
	}

	return json.Unmarshal([]byte(configJSON), v)
}
